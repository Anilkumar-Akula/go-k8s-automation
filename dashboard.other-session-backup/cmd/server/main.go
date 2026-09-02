package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"k8s-automation-dashboard/internal/api"
	"k8s-automation-dashboard/internal/collector"
	"k8s-automation-dashboard/internal/kclient"
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

	// Try connecting to Kubernetes
	clients, err := kclient.New()
	initialDemo := false
	if err != nil {
		slog.Warn("Could not connect to live Kubernetes cluster, defaulting to Demo Mode", "error", err)
		initialDemo = true
	} else {
		slog.Info("Successfully connected to Kubernetes cluster", "live", clients.Live)
	}

	// Initialize manager & background simulation
	manager := collector.NewManager(clients, initialDemo)
	go manager.StartBackgroundSimulation(ctx)

	// Resolve web directory
	webDir := os.Getenv("WEB_DIR")
	if webDir == "" {
		// check relative to binary or current working dir
		candidates := []string{
			"./web",
			"../web",
			"../../web",
			"/app/web",
		}
		for _, c := range candidates {
			if _, err := os.Stat(filepath.Join(c, "index.html")); err == nil {
				webDir = c
				break
			}
		}
	}

	slog.Info("Serving web dashboard UI", "dir", webDir)

	server := api.NewServer(manager, webDir)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      server.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // 0 for Server-Sent Events support
	}

	go func() {
		slog.Info("Kubernetes Automation Control Plane running", "addr", fmt.Sprintf("http://localhost:%s", port))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server failed", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("Shutting down dashboard server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}
	slog.Info("Dashboard server exited")
}
