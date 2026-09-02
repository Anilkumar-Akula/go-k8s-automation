package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dashboard-api/internal/api"
	"dashboard-api/internal/config"
	"dashboard-api/internal/events"
	"dashboard-api/internal/kclient"
	"dashboard-api/internal/poller"
)

func main() {
	cfg := config.Load()

	restConfig, err := kclient.Config()
	if err != nil {
		slog.Error("failed to build kubernetes config", "error", err)
		os.Exit(1)
	}
	clientset, err := kclient.New(restConfig)
	if err != nil {
		slog.Error("failed to build kubernetes client", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	cache := poller.NewCache()
	store := events.NewStore(cfg.EventBuffer)
	go poller.Run(ctx, cfg, clientset, cache, store)

	server := api.NewServer(cfg, cache, store)
	httpSrv := &http.Server{Addr: cfg.ListenAddr, Handler: server.Routes()}

	go func() {
		slog.Info("dashboard-api listening", "addr", cfg.ListenAddr, "poll_interval", cfg.PollInterval)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server failed", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
	slog.Info("shutdown complete")
}
