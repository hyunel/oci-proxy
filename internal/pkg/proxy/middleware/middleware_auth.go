package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"oci-proxy/internal/pkg/config"
	"oci-proxy/internal/pkg/logging"
)

type Handler func(*http.Request) (*http.Response, error)

type cachedToken struct {
	token     string
	expiresAt time.Time
}

type AuthMiddleware struct {
	cfg        *config.Config
	tokenCache sync.Map
}

func NewAuthMiddleware(cfg *config.Config) *AuthMiddleware {
	return &AuthMiddleware{cfg: cfg}
}

func (m *AuthMiddleware) Name() string {
	return "auth"
}

func (m *AuthMiddleware) Process(req *http.Request, next Handler) (*http.Response, error) {
	req = m.applyAuth(req)
	resp, err := next(req)
	if err != nil {
		return nil, err
	}
	return m.handleAuthChallenge(req, resp, next)
}

func (m *AuthMiddleware) applyAuth(req *http.Request) *http.Request {
	settings := m.cfg.GetRegistrySettings(req.URL.Host)
	if settings.Auth.Username != "" {
		newReq := req.Clone(req.Context())
		settings.Auth.ApplyToRequest(newReq)
		return newReq
	}
	return m.tryApplyCachedToken(req)
}

func (m *AuthMiddleware) handleAuthChallenge(req *http.Request, resp *http.Response, next Handler) (*http.Response, error) {
	if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
		return resp, nil
	}

	authHeader := resp.Header.Get("Www-Authenticate")
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return resp, nil
	}

	logging.Logger.Debug("attempting anonymous authentication", "status", resp.StatusCode, "registry", req.URL.Host)
	retryResp, err := m.fetchTokenAndRetry(req, resp, next)
	if err != nil {
		logging.Logger.Error("anonymous authentication failed", "error", err, "registry", req.URL.Host)
		return resp, nil
	}
	return retryResp, nil
}

func (m *AuthMiddleware) tryApplyCachedToken(req *http.Request) *http.Request {
	scope := getScopeFromRequest(req)
	if scope == "" {
		return req
	}

	cacheKey := fmt.Sprintf("%s::%s", req.URL.Host, scope)
	val, ok := m.tokenCache.Load(cacheKey)
	if !ok {
		return req
	}

	cached := val.(cachedToken)
	if time.Now().After(cached.expiresAt) {
		m.tokenCache.Delete(cacheKey)
		return req
	}

	logging.Logger.Debug("using cached token", "key", cacheKey)
	newReq := req.Clone(req.Context())
	newReq.Header.Set("Authorization", "Bearer "+cached.token)
	return newReq
}

func (m *AuthMiddleware) fetchTokenAndRetry(req *http.Request, origResp *http.Response, next Handler) (*http.Response, error) {
	authHeader := origResp.Header.Get("Www-Authenticate")
	params := parseAuthHeader(authHeader)

	realm, ok := params["realm"]
	if !ok {
		return nil, fmt.Errorf("missing realm in Www-Authenticate header")
	}

	token, expiresIn, err := m.getAnonymousToken(realm, params["service"], params["scope"])
	if err != nil {
		return nil, fmt.Errorf("failed to get anonymous token: %w", err)
	}

	if expiresIn == 0 {
		expiresIn = 60
	}
	cacheKey := fmt.Sprintf("%s::%s", req.URL.Host, params["scope"])
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)
	m.tokenCache.Store(cacheKey, cachedToken{token: token, expiresAt: expiresAt})
	logging.Logger.Debug("stored token in cache", "key", cacheKey, "expires_in", expiresIn)

	origResp.Body.Close()
	retryReq := req.Clone(req.Context())
	retryReq.Header.Set("Authorization", "Bearer "+token)
	return next(retryReq)
}

func (m *AuthMiddleware) getAnonymousToken(realm, service, scope string) (string, int, error) {
	authURL := fmt.Sprintf("%s?service=%s", realm, service)
	if scope != "" {
		authURL += "&scope=" + scope
	}

	logging.Logger.Debug("fetching anonymous token", "url", authURL)
	resp, err := http.Get(authURL)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("token request failed with status %s", resp.Status)
	}

	var tokenResp struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", 0, err
	}

	if tokenResp.Token != "" {
		return tokenResp.Token, tokenResp.ExpiresIn, nil
	}
	if tokenResp.AccessToken != "" {
		return tokenResp.AccessToken, tokenResp.ExpiresIn, nil
	}
	return "", 0, fmt.Errorf("token not found in response")
}

func getScopeFromRequest(req *http.Request) string {
	path := req.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 && parts[0] == "v2" {
		repo := strings.Join(parts[1:len(parts)-2], "/")
		lastPart := parts[len(parts)-2]
		if repo != "" && (lastPart == "manifests" || lastPart == "blobs") {
			return fmt.Sprintf("repository:%s:pull", repo)
		}
	}
	return ""
}

func parseAuthHeader(header string) map[string]string {
	params := make(map[string]string)
	parts := strings.Split(strings.TrimPrefix(strings.ToLower(header), "bearer "), ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		kv := strings.SplitN(p, "=", 2)
		if len(kv) == 2 {
			params[kv[0]] = strings.Trim(kv[1], "\"")
		}
	}
	return params
}
