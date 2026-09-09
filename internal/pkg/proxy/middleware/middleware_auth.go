package middleware

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"oci-proxy/internal/pkg/config"
	"oci-proxy/internal/pkg/logging"
	"oci-proxy/internal/pkg/registry"
)

type Handler func(*http.Request) (*http.Response, error)

const (
	tokenTimeout = 30 * time.Second
	tokenMinTTL  = 60 * time.Second
	tokenLeeway  = 10 * time.Second
)

type token struct {
	key       string
	value     string
	expiresAt time.Time
}

func (t *token) valid() bool {
	return time.Now().Before(t.expiresAt)
}

type AuthMiddleware struct {
	cfg    *config.Config
	tokens sync.Map
}

func NewAuthMiddleware(cfg *config.Config) *AuthMiddleware {
	return &AuthMiddleware{cfg: cfg}
}

func (m *AuthMiddleware) Name() string {
	return "auth"
}

func (m *AuthMiddleware) Process(req *http.Request, next Handler) (*http.Response, error) {
	sent, authorized := m.authorize(req)
	resp, err := next(authorized)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
		return resp, nil
	}

	challenge, ok := parseChallenge(resp.Header.Get("Www-Authenticate"))
	if !ok {
		return resp, nil
	}

	if sent != nil {
		// drop only the token that was just rejected, never one refreshed concurrently
		m.tokens.CompareAndDelete(sent.key, sent)
	}

	retried, err := m.retry(req, challenge, next)
	if err != nil {
		logging.Logger.Error("registry authentication failed", "registry", req.URL.Host, "error", err)
		return resp, nil
	}
	resp.Body.Close()
	return retried, nil
}

func (m *AuthMiddleware) authorize(req *http.Request) (*token, *http.Request) {
	host := req.URL.Host
	if name := registry.Parse(req.URL.Path).Name; name != "" {
		if v, ok := m.tokens.Load(tokenKey(host, pullScope(name))); ok {
			if t := v.(*token); t.valid() {
				out := req.Clone(req.Context())
				out.Header.Set("Authorization", "Bearer "+t.value)
				return t, out
			}
		}
	}
	if auth := m.cfg.GetRegistrySettings(host).Auth; auth.IsSet() {
		out := req.Clone(req.Context())
		auth.ApplyToRequest(out)
		return nil, out
	}
	return nil, req
}

func (m *AuthMiddleware) retry(req *http.Request, challenge map[string]string, next Handler) (*http.Response, error) {
	t, err := m.token(req.Context(), tokenKey(req.URL.Host, challenge["scope"]), req.URL.Host, challenge)
	if err != nil {
		return nil, err
	}

	out := req.Clone(req.Context())
	if req.Body != nil && req.Body != http.NoBody {
		// the first attempt consumed the body, so a replay must come from GetBody
		if req.GetBody == nil {
			return nil, fmt.Errorf("request body cannot be replayed")
		}
		if out.Body, err = req.GetBody(); err != nil {
			return nil, err
		}
	}
	out.Header.Set("Authorization", "Bearer "+t.value)
	return next(out)
}

func (m *AuthMiddleware) token(ctx context.Context, key, host string, challenge map[string]string) (*token, error) {
	if v, ok := m.tokens.Load(key); ok && v.(*token).valid() {
		return v.(*token), nil
	}

	value, expiresAt, err := m.fetch(ctx, host, challenge)
	if err != nil {
		return nil, err
	}

	t := &token{key: key, value: value, expiresAt: expiresAt}
	m.tokens.Store(key, t)
	return t, nil
}

func (m *AuthMiddleware) fetch(ctx context.Context, host string, challenge map[string]string) (string, time.Time, error) {
	endpoint, err := url.Parse(challenge["realm"])
	if err != nil {
		return "", time.Time{}, fmt.Errorf("invalid realm %q: %w", challenge["realm"], err)
	}

	settings := m.cfg.GetRegistrySettings(host)
	query := endpoint.Query()
	if service := challenge["service"]; service != "" {
		query.Set("service", service)
	}
	query.Del("scope")
	for _, scope := range strings.Fields(challenge["scope"]) {
		query.Add("scope", scope)
	}
	if settings.Auth.IsSet() {
		query.Set("account", settings.Auth.Username)
	}
	endpoint.RawQuery = query.Encode()

	ctx, cancel := context.WithTimeout(ctx, tokenTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return "", time.Time{}, err
	}
	anonymous := !settings.Auth.ApplyToRequest(req)

	client, err := settings.TokenClient()
	if err != nil {
		return "", time.Time{}, err
	}

	logging.Logger.Debug("fetching token", "url", endpoint.Redacted(), "anonymous", anonymous)
	resp, err := client.Do(req)
	if err != nil {
		return "", time.Time{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, fmt.Errorf("token request failed with status %s", resp.Status)
	}

	var body struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", time.Time{}, err
	}

	value := cmp.Or(body.Token, body.AccessToken)
	if value == "" {
		return "", time.Time{}, fmt.Errorf("token not found in response")
	}

	ttl := max(time.Duration(body.ExpiresIn)*time.Second, tokenMinTTL)
	return value, time.Now().Add(ttl - tokenLeeway), nil
}

func tokenKey(host, scope string) string {
	return host + "::" + scope
}

func pullScope(name string) string {
	return "repository:" + name + ":pull"
}

func parseChallenge(header string) (map[string]string, bool) {
	const scheme = "bearer "
	if len(header) < len(scheme) || !strings.EqualFold(header[:len(scheme)], scheme) {
		return nil, false
	}

	params := map[string]string{}
	for rest := header[len(scheme):]; rest != ""; {
		name, tail, ok := strings.Cut(rest, "=")
		if !ok {
			break
		}
		var value string
		value, rest = cutValue(tail)

		// only parameter names are case-insensitive; values such as scope are not
		params[strings.ToLower(strings.Trim(name, " \t,"))] = value
	}

	if params["realm"] == "" {
		return nil, false
	}
	return params, true
}

func cutValue(s string) (value, rest string) {
	s = strings.TrimLeft(s, " \t")
	if !strings.HasPrefix(s, `"`) {
		value, rest, _ = strings.Cut(s, ",")
		return strings.TrimRight(value, " \t"), rest
	}

	// quoted, so a comma inside the value is not a parameter separator
	var b strings.Builder
	for i := 1; i < len(s); i++ {
		switch s[i] {
		case '\\':
			if i++; i < len(s) {
				b.WriteByte(s[i])
			}
		case '"':
			return b.String(), s[i+1:]
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String(), ""
}
