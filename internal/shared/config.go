package shared

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Environment      string
	HTTPAddr         string
	MetricsAddr      string
	UserDSN          string
	TravelDSN        string
	OrderDSN         string
	PaymentDSN       string
	RedisAddr        string
	RedisPassword    string
	KafkaBrokers     []string
	PaymentTopic     string
	PaymentGroup     string
	JWTSecret        string
	JWTExpire        time.Duration
	JaegerEndpoint   string
	WxAppID          string
	WxAppSecret      string
	WxMchID          string
	WxMchCertSerial  string
	WxMchPrivateKey  string
	WxAPIv3Key       string
	WxNotifyURL      string
	ElasticsearchURL string
	SearchIndex      string
	AdminJWTSecret   string
	AdminJWTExpire   time.Duration
	AdminInitialUser string
	AdminInitialPass string
	AgentAPIKey      string
	AgentBaseURL     string
	AgentModel       string
	AgentTimeout     time.Duration
}

func LoadConfig() Config {
	mysqlHost := env("MYSQL_HOST", "mysql:3306")
	mysqlUser := env("MYSQL_USER", "root")
	mysqlPass := env("MYSQL_PASSWORD", "PXDN93VRKUm8TeE7")
	dsn := func(db string) string {
		return fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=true&loc=Asia%%2FShanghai", mysqlUser, mysqlPass, mysqlHost, db)
	}
	return Config{
		Environment:      env("APP_ENV", "development"),
		HTTPAddr:         env("HTTP_ADDR", ":8080"),
		MetricsAddr:      env("METRICS_ADDR", ":4000"),
		UserDSN:          env("USER_DSN", dsn("looklook_usercenter")),
		TravelDSN:        env("TRAVEL_DSN", dsn("looklook_travel")),
		OrderDSN:         env("ORDER_DSN", dsn("looklook_order")),
		PaymentDSN:       env("PAYMENT_DSN", dsn("looklook_payment")),
		RedisAddr:        env("REDIS_ADDR", "redis:6379"),
		RedisPassword:    env("REDIS_PASSWORD", "G62m50oigInC30sf"),
		KafkaBrokers:     []string{env("KAFKA_BROKER", "kafka:9092")},
		PaymentTopic:     env("PAYMENT_TOPIC", "payment-update-paystatus-topic"),
		PaymentGroup:     env("PAYMENT_GROUP", "gin-payment-update-paystatus-group"),
		JWTSecret:        env("JWT_SECRET", "ae0536f9-6450-4606-8e13-5a19ed505da0"),
		JWTExpire:        time.Duration(envInt("JWT_EXPIRE_SECONDS", 31536000)) * time.Second,
		JaegerEndpoint:   env("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
		WxAppID:          os.Getenv("WX_APP_ID"),
		WxAppSecret:      os.Getenv("WX_APP_SECRET"),
		WxMchID:          os.Getenv("WX_MCH_ID"),
		WxMchCertSerial:  os.Getenv("WX_MCH_CERT_SERIAL"),
		WxMchPrivateKey:  os.Getenv("WX_MCH_PRIVATE_KEY"),
		WxAPIv3Key:       os.Getenv("WX_API_V3_KEY"),
		WxNotifyURL:      os.Getenv("WX_NOTIFY_URL"),
		ElasticsearchURL: env("ELASTICSEARCH_URL", "http://elasticsearch:9200"),
		SearchIndex:      env("SEARCH_INDEX", "gin-looklook-homestay-v1"),
		AdminJWTSecret:   env("ADMIN_JWT_SECRET", env("JWT_SECRET", "ae0536f9-6450-4606-8e13-5a19ed505da0")+":admin"),
		AdminJWTExpire:   time.Duration(envInt("ADMIN_JWT_EXPIRE_SECONDS", 28800)) * time.Second,
		AdminInitialUser: env("ADMIN_INITIAL_USER", "admin"),
		AdminInitialPass: env("ADMIN_INITIAL_PASSWORD", "Admin@123"),
		AgentAPIKey:      env("AI_API_KEY", os.Getenv("OPENAI_API_KEY")),
		AgentBaseURL:     env("AI_BASE_URL", os.Getenv("OPENAI_BASE_URL")),
		AgentModel:       env("AI_MODEL", env("OPENAI_MODEL", "gpt-4.1-mini")),
		AgentTimeout:     time.Duration(envInt("AI_TIMEOUT_SECONDS", 20)) * time.Second,
	}
}

func (c Config) Validate() error {
	if c.JWTExpire <= 0 || c.AdminJWTExpire <= 0 {
		return errors.New("JWT expiration must be positive")
	}
	if c.AgentTimeout <= 0 {
		return errors.New("AI timeout must be positive")
	}
	if c.Environment != "production" {
		return nil
	}
	if len(c.JWTSecret) < 32 || c.JWTSecret == "replace-this-in-production" || c.JWTSecret == "change-me-in-production" {
		return errors.New("JWT_SECRET must be a non-placeholder value of at least 32 characters in production")
	}
	if len(c.AdminJWTSecret) < 32 || c.AdminJWTSecret == c.JWTSecret || c.AdminJWTSecret == "replace-admin-secret-in-production" || c.AdminJWTSecret == "change-admin-secret-in-production" {
		return errors.New("ADMIN_JWT_SECRET must be a distinct non-placeholder value of at least 32 characters in production")
	}
	if c.AdminInitialPass == "Admin@123" || len(c.AdminInitialPass) < 12 {
		return errors.New("ADMIN_INITIAL_PASSWORD must be changed and contain at least 12 characters in production")
	}
	return nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int64) int64 {
	value, err := strconv.ParseInt(os.Getenv(key), 10, 64)
	if err != nil {
		return fallback
	}
	return value
}
