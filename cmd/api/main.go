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

	"gin-looklook/internal/admin"
	"gin-looklook/internal/bootstrap"
	"gin-looklook/internal/httpserver"
	"gin-looklook/internal/order"
	"gin-looklook/internal/payment"
	"gin-looklook/internal/search"
	"gin-looklook/internal/seckill"
	"gin-looklook/internal/shared"
	"gin-looklook/internal/travel"
	"gin-looklook/internal/user"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	cfg := shared.LoadConfig()
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	shutdownTrace, err := shared.InitTelemetry(cfg.JaegerEndpoint)
	if err != nil {
		slog.Error("init telemetry", "error", err)
		os.Exit(1)
	}
	app, err := bootstrap.New(ctx, cfg)
	if err != nil {
		slog.Error("init application", "error", err)
		os.Exit(1)
	}
	handlers := httpserver.Handlers{
		User:    user.NewHandler(app.UserSvc),
		Travel:  travel.NewHandler(app.TravelSvc, app.UserSvc),
		Order:   order.NewHandler(app.OrderSvc),
		Payment: payment.NewHandler(app.PaymentSvc),
		Seckill: seckill.NewHandler(app.SeckillSvc),
		Search:  search.NewHandler(app.SearchSvc),
		Admin:   admin.NewHandler(app.AdminSvc),
	}
	router := httpserver.NewRouter(handlers, cfg, app.AdminSvc)
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: router, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go serve("http", server)
	slog.Info("gin-looklook api started", "http", cfg.HTTPAddr)
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	_ = shutdownTrace(shutdownCtx)
	slog.Info("gin-looklook api stopped")
}

func serve(name string, server *http.Server) {
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error(name+" server failed", "error", err)
	}
}
