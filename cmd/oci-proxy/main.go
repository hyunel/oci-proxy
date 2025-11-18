package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"oci-proxy/internal/pkg/config"
	"oci-proxy/internal/pkg/logging"
	"oci-proxy/internal/pkg/proxy"
)

func main() {
	configFile := flag.String("c", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		logging.Logger.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	logging.Init(cfg.LogLevel)

	logging.Logger.Info("Starting OCI proxy", "port", cfg.Port)

	server, err := proxy.NewProxy(cfg)
	if err != nil {
		logging.Logger.Error("Failed to create proxy", "error", err)
		os.Exit(1)
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Logger.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-shutdown

	logging.Logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logging.Logger.Error("Server shutdown failed", "error", err)
	}

	server.PersistCache()

	logging.Logger.Info("Server gracefully stopped")
}
