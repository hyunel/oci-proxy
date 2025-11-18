package proxy

import (
	"net/http"

	"oci-proxy/internal/pkg/proxy/middleware"
)

type Middleware interface {
	Name() string
	Process(req *http.Request, next middleware.Handler) (*http.Response, error)
}

type Pipeline struct {
	middlewares  []Middleware
	finalHandler middleware.Handler
}

func NewPipeline() *Pipeline {
	return &Pipeline{
		middlewares: make([]Middleware, 0),
	}
}

func (p *Pipeline) Use(m Middleware) *Pipeline {
	p.middlewares = append(p.middlewares, m)
	return p
}

func (p *Pipeline) SetFinalHandler(h middleware.Handler) *Pipeline {
	p.finalHandler = h
	return p
}

func (p *Pipeline) Execute(req *http.Request) (*http.Response, error) {
	if len(p.middlewares) == 0 {
		if p.finalHandler != nil {
			return p.finalHandler(req)
		}
		return nil, http.ErrNotSupported
	}

	var chain middleware.Handler
	chain = p.finalHandler

	for i := len(p.middlewares) - 1; i >= 0; i-- {
		m := p.middlewares[i]
		next := chain
		chain = func(r *http.Request) (*http.Response, error) {
			return m.Process(r, next)
		}
	}

	return chain(req)
}

type Transport struct {
	pipeline *Pipeline
}

func NewTransport(pipeline *Pipeline) *Transport {
	return &Transport{
		pipeline: pipeline,
	}
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.pipeline.Execute(req)
}
