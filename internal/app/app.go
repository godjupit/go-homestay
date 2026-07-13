package app

import (
	"context"

	"gin-looklook/internal/config"
	"gin-looklook/internal/repository"
	"gin-looklook/internal/service"
	"gin-looklook/internal/worker"

	"github.com/hibiken/asynq"
	"github.com/segmentio/kafka-go"
)

type App struct {
	Config   config.Config
	Repo     *repository.Repository
	Users    *service.UserService
	Travel   *service.TravelService
	Orders   *service.OrderService
	Payments *service.PaymentService
	Seckill  *service.SeckillService
	Admin    *service.AdminService
	Search   *service.SearchService
	Workers  *worker.Runtime
	Asynq    *asynq.Client
	Kafka    *kafka.Writer
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	repo, err := repository.Open(ctx, cfg)
	if err != nil {
		return nil, err
	}
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisAddr, Password: cfg.RedisPassword})
	writer := &kafka.Writer{Addr: kafka.TCP(cfg.KafkaBrokers...), Balancer: &kafka.Hash{}, RequiredAcks: kafka.RequireOne}
	users := service.NewUserService(repo, cfg)
	travel := service.NewTravelService(repo, users)
	orders := service.NewOrderService(repo, travel, asynqClient)
	seckill := service.NewSeckillService(repo, orders)
	if err = seckill.Warmup(ctx); err != nil {
		_ = writer.Close()
		_ = asynqClient.Close()
		repo.Close()
		return nil, err
	}
	payments := service.NewPaymentService(repo, users, orders, cfg)
	admin := service.NewAdminService(repo, cfg)
	if err = admin.Bootstrap(ctx); err != nil {
		_ = writer.Close()
		_ = asynqClient.Close()
		repo.Close()
		return nil, err
	}
	search := service.NewSearchService(repo, cfg)
	if err = search.EnsureIndex(ctx); err != nil {
		_ = writer.Close()
		_ = asynqClient.Close()
		repo.Close()
		return nil, err
	}
	if err = repo.BootstrapSearchOutbox(ctx); err != nil {
		_ = writer.Close()
		_ = asynqClient.Close()
		repo.Close()
		return nil, err
	}
	workers := worker.New(cfg, repo, orders, seckill, search, writer)
	return &App{Config: cfg, Repo: repo, Users: users, Travel: travel, Orders: orders, Payments: payments, Seckill: seckill, Admin: admin, Search: search, Workers: workers, Asynq: asynqClient, Kafka: writer}, nil
}
func (a *App) Close() { a.Workers.Stop(); _ = a.Kafka.Close(); _ = a.Asynq.Close(); a.Repo.Close() }
