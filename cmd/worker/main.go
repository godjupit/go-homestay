package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"gin-looklook/internal/bootstrap"
	"gin-looklook/internal/shared"

	"gin-looklook/internal/worker"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	cfg := shared.LoadConfig()
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	_, err := shared.InitTelemetry(cfg.JaegerEndpoint)
	if err != nil {
		slog.Error("init telemetry", "error", err)
		os.Exit(1)
	}
	app, err := bootstrap.New(ctx, cfg)
	if err != nil {
		slog.Error("init application", "error", err)
		os.Exit(1)
	}
	rt := worker.New(cfg, app.OrderSvc, app.SeckillSvc, app.SearchSvc, app.SearchRepo, app.PaymentRepo, app.Kafka, app.Redis)
	if err = rt.Start(ctx); err != nil {
		slog.Error("start workers", "error", err)
		os.Exit(1)
	}
	slog.Info("gin-looklook worker started")
	<-ctx.Done()
	rt.Stop()
	slog.Info("gin-looklook worker stopped")
}
