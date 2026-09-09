package config

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"golang.org/x/net/proxy"
)

type clientKey struct {
	upstreamProxy   string
	followRedirects bool
}

var clients sync.Map

func (s RegistrySettings) Client() (*http.Client, error) {
	return client(clientKey{s.UpstreamProxy, s.FollowRedirects == nil || *s.FollowRedirects})
}

func (s RegistrySettings) TokenClient() (*http.Client, error) {
	return client(clientKey{s.UpstreamProxy, true})
}

func client(key clientKey) (*http.Client, error) {
	if c, ok := clients.Load(key); ok {
		return c.(*http.Client), nil
	}
	transport, err := newTransport(key.upstreamProxy)
	if err != nil {
		return nil, err
	}
	c := &http.Client{Transport: transport}
	if !key.followRedirects {
		c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	}
	shared, _ := clients.LoadOrStore(key, c)
	return shared.(*http.Client), nil
}

func newTransport(upstreamProxy string) (http.RoundTripper, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if upstreamProxy == "" {
		return transport, nil
	}

	proxyURL, err := url.Parse(upstreamProxy)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream_proxy %q: %w", upstreamProxy, err)
	}

	switch proxyURL.Scheme {
	case "http", "https":
		transport.Proxy = http.ProxyURL(proxyURL)
	case "socks5", "socks5h":
		dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("invalid upstream_proxy %q: %w", upstreamProxy, err)
		}
		ctxDialer, ok := dialer.(proxy.ContextDialer)
		if !ok {
			return nil, fmt.Errorf("upstream_proxy %q does not support context dialing", upstreamProxy)
		}
		transport.DialContext = ctxDialer.DialContext
	default:
		return nil, fmt.Errorf("unsupported upstream_proxy scheme %q", proxyURL.Scheme)
	}
	return transport, nil
}
