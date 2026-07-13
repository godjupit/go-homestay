package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gin-looklook/internal/app"
	"gin-looklook/internal/config"
	"gin-looklook/internal/httpapi"
	"gin-looklook/internal/platform"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	cfg := config.Load()
	shutdownTrace, err := platform.InitTelemetry(cfg.JaegerEndpoint)
	if err != nil {
		slog.Error("init telemetry", "error", err)
		os.Exit(1)
	}
	application, err := app.New(ctx, cfg)
	if err != nil {
		slog.Error("init application", "error", err)
		os.Exit(1)
	}
	if err = application.Workers.Start(ctx); err != nil {
		slog.Error("start workers", "error", err)
		application.Close()
		os.Exit(1)
	}
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: httpapi.NewRouter(application), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	metrics := &http.Server{Addr: cfg.MetricsAddr, Handler: promhttp.Handler(), ReadHeaderTimeout: 3 * time.Second}
	go serve("metrics", metrics)
	go serve("http", server)
	slog.Info("gin-looklook started", "http", cfg.HTTPAddr, "metrics", cfg.MetricsAddr)
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	_ = metrics.Shutdown(shutdownCtx)
	application.Close()
	_ = shutdownTrace(shutdownCtx)
	slog.Info("gin-looklook stopped")
}

func serve(name string, server *http.Server) {
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error(name+" server failed", "error", err)
	}
}
