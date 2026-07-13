# Gin LookLook

这是原 `go-zero-looklook` 的独立 Gin 单体实现。HTTP 路径、核心业务规则和 MySQL/Redis/Kafka/Asynq/Prometheus/Jaeger/ELK 中间件保持兼容；原项目不会被修改或替换。

## 运行

```bash
cp config/.env.example config/.env
docker compose up -d --build
curl http://localhost:8080/healthz
```

首次启动会自动建立四个数据库并导入兼容原结构的演示数据。可直接访问业务端口 `8080`，也可通过 Nginx 网关 `8888` 访问；指标端口是 `4000`。Jaeger、Prometheus、Grafana、Asynqmon、Kibana 分别位于 `16686`、`9090`、`3001`、`8980`、`5601`。

本地开发可通过环境变量覆盖配置后运行：

```bash
go test ./...
go run ./cmd/server
```

如果是在已有数据卷上升级，而不是首次创建数据库，需要依次执行新增迁移：

```bash
mysql -h 127.0.0.1 -P 33069 -u root -p < migrations/006_seckill.sql
mysql -h 127.0.0.1 -P 33069 -u root -p < migrations/y01_admin_rbac.sql
mysql -h 127.0.0.1 -P 33069 -u root -p < migrations/y02_search.sql
```

管理后台首次启动会创建 `admin / Admin@123` 超级管理员。它只用于本地演示，部署前必须通过 `ADMIN_INITIAL_USER`、`ADMIN_INITIAL_PASSWORD` 和 `ADMIN_JWT_SECRET` 更改。

微信登录和微信支付需要在 `.env` 中提供真实配置；未配置不会影响其他业务启动，支付接口会返回明确的“未配置”错误。

## 架构

```text
Gin Router / JWT / Recovery / Metrics / OpenTelemetry
                       |
 User | Travel | Order | Payment | Admin | Search services
                       |
 MySQL(4 schemas) | Redis | Kafka | Asynq | Elasticsearch
```

单体仅合并部署边界，不把业务揉进 Handler：路由负责协议，Service 负责用例和事务规则，Repository 负责数据访问，Worker 负责异步消费。详细设计见 [架构与业务实现](docs/架构与业务实现.md)，面试表达与所有已实现亮点见 [技术亮点与面试讲解](docs/技术亮点与面试讲解.md)。

## API

保留原项目 17 个业务接口：用户 4 个、旅行 8 个、订单 3 个、支付 2 个；新增秒杀 3 个接口、公开民宿搜索 1 个接口和 RBAC 管理后台 14 个接口，另提供 `/healthz` 和 `/metrics`。请求与响应仍使用统一 JSON 结构。

秒杀的 Lua、Redis Stream、双层防超卖和补偿设计见 [秒杀机制设计与实现](docs/秒杀机制设计与实现.md)；RBAC、四种数据范围、审计、地理搜索和搜索 Outbox 见 [RBAC 与 Elasticsearch 搜索设计与实现](docs/RBAC与Elasticsearch搜索设计与实现.md)。
