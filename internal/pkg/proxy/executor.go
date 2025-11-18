package proxy

import (
	"fmt"
	"net/http"
	"net/url"

	"oci-proxy/internal/pkg/config"
	"oci-proxy/internal/pkg/logging"

	"golang.org/x/net/proxy"
)

type Executor struct {
	cfg *config.Config
}

func NewExecutor(cfg *config.Config) *Executor {
	return &Executor{cfg: cfg}
}

func (e *Executor) Execute(req *http.Request) (*http.Response, error) {
	settings := e.cfg.GetRegistrySettings(req.URL.Host)
	client := e.getClientForRegistry(settings)
	logging.Logger.Debug("executing request", "url", req.URL.String())
	return client.Do(req)
}

func (e *Executor) getClientForRegistry(settings config.RegistrySettings) *http.Client {
	transport := http.DefaultTransport

	if settings.UpstreamProxy != "" {
		var err error
		transport, err = createTransportWithProxy(settings.UpstreamProxy)
		if err != nil {
			logging.Logger.Error("failed to create transport for upstream proxy", "error", err)
			transport = http.DefaultTransport
		}
	}

	client := &http.Client{Transport: transport}

	if settings.FollowRedirects != nil && !*settings.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return client
}

func createTransportWithProxy(upstreamProxy string) (http.RoundTripper, error) {
	proxyURL, err := url.Parse(upstreamProxy)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream_proxy URL: %w", err)
	}

	switch proxyURL.Scheme {
	case "http", "https":
		return &http.Transport{Proxy: http.ProxyURL(proxyURL)}, nil
	case "socks5":
		dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("failed to create socks5 dialer: %w", err)
		}
		return &http.Transport{Dial: dialer.Dial}, nil
	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", proxyURL.Scheme)
	}
}
