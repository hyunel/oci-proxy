package proxy

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"strings"

	"oci-proxy/internal/pkg/config"
	"oci-proxy/internal/pkg/logging"
	"oci-proxy/internal/pkg/proxy/middleware"
)

//go:embed all:web
var webFS embed.FS

type ProxyServer struct {
	*http.Server
	cacheManager *CacheManager
}

func NewProxy(cfg *config.Config) (*ProxyServer, error) {
	cacheManager := NewCacheManager(cfg)
	executor := NewExecutor(cfg)

	pipeline := NewPipeline().
		Use(middleware.NewCacheMiddleware(cacheManager)).
		Use(middleware.NewAuthMiddleware(cfg)).
		SetFinalHandler(executor.Execute)

	transport := NewTransport(pipeline)

	proxy := &httputil.ReverseProxy{
		Director:  newDirector(cfg),
		Transport: transport,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logging.Logger.Debug("proxy error", "error", err, "path", r.URL.Path)
			if err == r.Context().Err() {
				return
			}
			w.WriteHeader(http.StatusBadGateway)
		},
	}

	ps := &ProxyServer{
		cacheManager: cacheManager,
	}
	ps.Server = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: newProxyHandler(proxy, cacheManager, cfg),
	}
	return ps, nil
}

func newProxyHandler(proxy *httputil.ReverseProxy, cacheManager *CacheManager, cfg *config.Config) http.Handler {
	mux := http.NewServeMux()

	logRequest := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logging.Logger.Info("Request", "method", r.Method, "path", r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}

	requireAuth := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !cfg.Auth.IsAuthenticated(r) {
				w.Header().Set("WWW-Authenticate", `Basic realm="OCI-Proxy"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next(w, r)
		}
	}

	mux.HandleFunc("/_/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	mux.HandleFunc("/_/stats", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		stats := cacheManager.GetStats()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(stats)
	}))

	webRoot, _ := fs.Sub(webFS, "web")
	fs := http.FileServer(http.FS(webRoot))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		if _, err := webRoot.Open(strings.TrimPrefix(path, "/")); err == nil {
			fs.ServeHTTP(w, r)
			return
		}

		requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if cfg.WhitelistMode && !isRegistryAllowed(r, cfg) {
				http.Error(w, "Registry not allowed", http.StatusForbidden)
				return
			}
			proxy.ServeHTTP(w, r)
		})(w, r)
	})

	return logRequest(mux)
}

func (ps *ProxyServer) PersistCache() {
	if ps.cacheManager != nil {
		ps.cacheManager.PersistAll()
	}
}

func newDirector(cfg *config.Config) func(*http.Request) {
	return func(req *http.Request) {
		remoteHost := cfg.DefaultRegistry

		path := req.URL.Path
		parts := strings.Split(strings.Trim(path, "/"), "/")

		if len(parts) >= 2 && parts[0] == "v2" {
			potentialRegistry := parts[1]
			if strings.Contains(potentialRegistry, ".") {
				remoteHost = potentialRegistry
				req.URL.Path = "/v2/" + strings.Join(parts[2:], "/")
			} else if !strings.Contains(potentialRegistry, "/") {
				req.URL.Path = "/v2/library/" + strings.Join(parts[1:], "/")
			}
		}

		settings := cfg.GetRegistrySettings(remoteHost)
		if settings.Insecure != nil && *settings.Insecure {
			req.URL.Scheme = "http"
		} else {
			req.URL.Scheme = "https"
		}

		req.URL.Host = remoteHost
		req.Host = remoteHost
		req.RequestURI = ""
		req.Header.Del("Authorization")
	}
}

func isRegistryAllowed(r *http.Request, cfg *config.Config) bool {
	path := r.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) >= 2 && parts[0] == "v2" {
		potentialRegistry := parts[1]
		if strings.Contains(potentialRegistry, ".") {
			return cfg.IsRegistryAllowed(potentialRegistry)
		}
	}
	return cfg.IsRegistryAllowed(cfg.DefaultRegistry)
}
