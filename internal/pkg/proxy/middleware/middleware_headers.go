package middleware

import (
	"net/http"
	"strings"

	"oci-proxy/internal/pkg/config"
	"oci-proxy/internal/pkg/registry"
)

const immutableMaxAge = "max-age=31536000, immutable"

type HeaderMiddleware struct {
	cfg *config.Config
}

func NewHeaderMiddleware(cfg *config.Config) *HeaderMiddleware {
	return &HeaderMiddleware{cfg: cfg}
}

func (m *HeaderMiddleware) Name() string {
	return "header"
}

func (m *HeaderMiddleware) Process(req *http.Request, next Handler) (*http.Response, error) {
	resp, err := next(req)
	if err != nil {
		return nil, err
	}
	m.setCacheControl(req, resp)
	return resp, nil
}

func (m *HeaderMiddleware) setCacheControl(req *http.Request, resp *http.Response) {
	if resp.StatusCode != http.StatusOK {
		return
	}
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		return
	}

	path := registry.Parse(req.URL.Path)
	if path.Kind != "blobs" && path.Kind != "manifests" {
		return
	}

	if path.Kind == "manifests" {
		// the manifest returned depends on Accept, so a shared cache must vary on it
		resp.Header.Set("Vary", "Accept")
	}

	visibility := "public"
	if m.cfg.Auth.IsSet() || m.cfg.GetRegistrySettings(req.URL.Host).Auth.IsSet() {
		visibility = "private"
	}

	// a tag cannot contain ":", so a colon means the reference is a digest
	if strings.Contains(path.Rest, ":") {
		resp.Header.Set("Cache-Control", visibility+", "+immutableMaxAge)
		return
	}
	resp.Header.Set("Cache-Control", visibility+", no-cache")
}
