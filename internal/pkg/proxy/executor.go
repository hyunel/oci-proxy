package proxy

import (
	"net/http"

	"oci-proxy/internal/pkg/config"
	"oci-proxy/internal/pkg/logging"
)

type Executor struct {
	cfg *config.Config
}

func NewExecutor(cfg *config.Config) *Executor {
	return &Executor{cfg: cfg}
}

func (e *Executor) Execute(req *http.Request) (*http.Response, error) {
	client, err := e.cfg.GetRegistrySettings(req.URL.Host).Client()
	if err != nil {
		return nil, err
	}
	logging.Logger.Debug("executing request", "url", req.URL.String())
	return client.Do(req)
}
