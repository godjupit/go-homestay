package bootstrap

import (
	"context"

	"gin-looklook/internal/admin"
	"gin-looklook/internal/order"
	"gin-looklook/internal/payment"
	"gin-looklook/internal/search"
	"gin-looklook/internal/seckill"
	"gin-looklook/internal/shared"
	"gin-looklook/internal/travel"
	"gin-looklook/internal/user"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

type App struct {
	Config      shared.Config
	UserSvc     *user.Service
	TravelSvc   *travel.Service
	OrderSvc    *order.Service
	PaymentSvc  *payment.Service
	SeckillSvc  *seckill.Service
	SearchSvc   *search.Service
	AdminSvc    *admin.Service
	Redis       *redis.Client
	Asynq       *asynq.Client
	Kafka       *kafka.Writer
	OrderRepo   *order.Repository
	PaymentRepo *payment.Repository
	SearchRepo  *search.Repository
}

func New(ctx context.Context, cfg shared.Config) (*App, error) {
	// Open databases
	userDB, err := shared.OpenDB(cfg.UserDSN)
	if err != nil {
		return nil, err
	}
	travelDB, err := shared.OpenDB(cfg.TravelDSN)
	if err != nil {
		shared.CloseDB(userDB)
		return nil, err
	}
	orderDB, err := shared.OpenDB(cfg.OrderDSN)
	if err != nil {
		shared.CloseDB(userDB)
		shared.CloseDB(travelDB)
		return nil, err
	}
	paymentDB, err := shared.OpenDB(cfg.PaymentDSN)
	if err != nil {
		shared.CloseDB(userDB)
		shared.CloseDB(travelDB)
		shared.CloseDB(orderDB)
		return nil, err
	}

	// Redis
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword})
	if err := rdb.Ping(ctx).Err(); err != nil {
		shared.CloseDB(userDB)
		shared.CloseDB(travelDB)
		shared.CloseDB(orderDB)
		shared.CloseDB(paymentDB)
		return nil, err
	}

	// Repositories
	userRepo := user.NewRepository(userDB, rdb)
	travelRepo := travel.NewRepository(travelDB, rdb)
	orderRepo := order.NewRepository(orderDB)
	paymentRepo := payment.NewRepository(paymentDB)
	searchRepo := search.NewRepository(travelDB)
	adminRepo := admin.NewRepository(userDB, travelDB)

	// Services
	userSvc := user.NewService(userRepo, cfg)
	travelSvc := travel.NewService(travelRepo, userSvc)

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisAddr, Password: cfg.RedisPassword})
	orderSvc := order.NewService(orderRepo, travelSvc, asynqClient)
	paymentSvc := payment.NewService(paymentRepo, userSvc, orderSvc, cfg)
	seckillSvc := seckill.NewService(orderRepo, orderSvc, rdb)
	searchSvc := search.NewService(searchRepo, cfg)
	adminSvc := admin.NewService(adminRepo, searchRepo, rdb, cfg)

	// Workers deps
	kafkaWriter := &kafka.Writer{Addr: kafka.TCP(cfg.KafkaBrokers...), Balancer: &kafka.Hash{}, RequiredAcks: kafka.RequireOne}

	// Bootstrap admin
	if err := adminSvc.Bootstrap(ctx); err != nil {
		userDB, travelDB, orderDB, paymentDB := userDB, travelDB, orderDB, paymentDB
		shared.CloseDB(userDB)
		shared.CloseDB(travelDB)
		shared.CloseDB(orderDB)
		shared.CloseDB(paymentDB)
		_ = rdb.Close()
		return nil, err
	}

	// Ensure ES index
	if err := searchSvc.EnsureIndex(ctx); err != nil {
		shared.CloseDB(userDB)
		shared.CloseDB(travelDB)
		shared.CloseDB(orderDB)
		shared.CloseDB(paymentDB)
		_ = rdb.Close()
		return nil, err
	}

	// Bootstrap search outbox
	if err := searchRepo.BootstrapSearchOutbox(ctx); err != nil {
		shared.CloseDB(userDB)
		shared.CloseDB(travelDB)
		shared.CloseDB(orderDB)
		shared.CloseDB(paymentDB)
		_ = rdb.Close()
		return nil, err
	}

	// Warmup seckill
	if err := seckillSvc.Warmup(ctx); err != nil {
		shared.CloseDB(userDB)
		shared.CloseDB(travelDB)
		shared.CloseDB(orderDB)
		shared.CloseDB(paymentDB)
		_ = rdb.Close()
		return nil, err
	}

	return &App{
		Config: cfg, UserSvc: userSvc, TravelSvc: travelSvc,
		OrderSvc: orderSvc, PaymentSvc: paymentSvc, SeckillSvc: seckillSvc,
		SearchSvc: searchSvc, AdminSvc: adminSvc, Redis: rdb,
		Asynq: asynqClient, Kafka: kafkaWriter,
		OrderRepo: orderRepo, PaymentRepo: paymentRepo, SearchRepo: searchRepo,
	}, nil
}
