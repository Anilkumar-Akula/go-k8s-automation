package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s-automation-dashboard-backend/internal/api"
	"k8s-automation-dashboard-backend/internal/events"
	"k8s-automation-dashboard-backend/internal/kubernetes"
	"k8s-automation-dashboard-backend/internal/prometheus"
	"k8s-automation-dashboard-backend/internal/store"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	slog.Info("Initializing Kubernetes Control Plane & Observability API...")

	// Initialize core singletons
	kubeClient := kubernetes.NewClusterClient()
	auditStore := store.NewMemoryAuditStore()
	eventBus := events.NewEventBus()
	metrics := prometheus.NewMetricsCollector()

	slog.Info("Cluster client status", "live", kubeClient.Live, "cluster", kubeClient.ClusterName)

	server := api.NewServer(kubeClient, auditStore, eventBus, metrics)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      server.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // 0 required for Server-Sent Events (SSE)
	}

	go func() {
		slog.Info("Go Dashboard API Server listening", "addr", fmt.Sprintf("http://localhost:%s", port))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("Gracefully shutting down Go Dashboard API...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced shutdown", "error", err)
	}
	slog.Info("Go Dashboard API exited cleanly")
}
