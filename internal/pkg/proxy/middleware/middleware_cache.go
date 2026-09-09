package middleware

import (
	"io"
	"net/http"
	"strings"
	"sync"

	"oci-proxy/internal/pkg/logging"
	"oci-proxy/internal/pkg/proxy/cache"
)

type CacheMiddleware struct {
	cacheManager CacheManager
}

type CacheManager interface {
	GetCache(registryHost string) *cache.Cache
}

func NewCacheMiddleware(cm CacheManager) *CacheMiddleware {
	return &CacheMiddleware{
		cacheManager: cm,
	}
}

func (m *CacheMiddleware) Name() string {
	return "cache"
}

func (m *CacheMiddleware) Process(req *http.Request, next Handler) (*http.Response, error) {
	if resp, ok := m.tryServeFromCache(req); ok {
		return resp, nil
	}

	resp, err := next(req)
	if err != nil {
		return nil, err
	}

	resp = m.cacheResponse(req, resp)
	return resp, nil
}

func (m *CacheMiddleware) tryServeFromCache(req *http.Request) (*http.Response, bool) {
	if !isBlobRequest(req) {
		return nil, false
	}

	digest := extractDigestFromPath(req.URL.Path)
	if digest == "" {
		return nil, false
	}

	cache := m.cacheManager.GetCache(req.URL.Host)
	reader, size, ok := cache.GetReader(digest)
	if !ok {
		return nil, false
	}

	logging.Logger.Debug("serving blob from cache", "digest", digest)
	return &http.Response{
		StatusCode:    http.StatusOK,
		Body:          reader,
		Header:        make(http.Header),
		ContentLength: size,
		Request:       req,
	}, true
}

func (m *CacheMiddleware) cacheResponse(req *http.Request, resp *http.Response) *http.Response {
	if !isBlobRequest(req) || resp.StatusCode != http.StatusOK {
		return resp
	}

	digest := extractDigestFromPath(req.URL.Path)
	if digest == "" {
		return resp
	}

	writer, err := m.cacheManager.GetCache(req.URL.Host).NewWriter(digest, digest)
	if err != nil {
		logging.Logger.Debug("not caching blob", "digest", digest, "error", err)
		return resp
	}

	resp.Body = &cacheWriter{body: resp.Body, writer: writer, digest: digest}
	return resp
}

func isBlobRequest(req *http.Request) bool {
	if req.Method != http.MethodGet {
		return false
	}
	parts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	return len(parts) >= 4 && parts[len(parts)-2] == "blobs"
}

func extractDigestFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 && parts[len(parts)-2] == "blobs" {
		return parts[len(parts)-1]
	}
	return ""
}

type cacheWriter struct {
	body   io.ReadCloser
	writer *cache.Writer
	digest string
	once   sync.Once
}

func (cw *cacheWriter) Read(p []byte) (int, error) {
	n, err := cw.body.Read(p)
	if n > 0 && cw.writer != nil {
		if _, writeErr := cw.writer.Write(p[:n]); writeErr != nil {
			// caching gives up here, but the client keeps its download
			logging.Logger.Error("failed to write blob to cache", "digest", cw.digest, "error", writeErr)
			cw.writer.Close()
			cw.writer = nil
		}
	}
	if err == io.EOF {
		cw.finish(true)
	}
	return n, err
}

func (cw *cacheWriter) Close() error {
	cw.finish(false)
	return cw.body.Close()
}

func (cw *cacheWriter) finish(complete bool) {
	cw.once.Do(func() {
		if cw.writer == nil {
			return
		}
		if !complete {
			cw.writer.Close()
			return
		}
		if err := cw.writer.Commit(); err != nil {
			logging.Logger.Error("failed to cache blob", "digest", cw.digest, "error", err)
			return
		}
		logging.Logger.Info("successfully cached blob", "digest", cw.digest)
	})
}
