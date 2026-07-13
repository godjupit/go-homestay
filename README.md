# go_homestay

这是原 `go-zero-looklook` 的独立 Gin 单体实现。HTTP 路径、核心业务规则和 MySQL/Redis/Kafka/Asynq/Prometheus/Jaeger/ELK 中间件保持兼容；原项目不会被修改或替换。

## 快速启动（Docker Compose 全家桶）

一条命令拉起所有服务——应用 + MySQL + Redis + Kafka + Elasticsearch + Jaeger + Prometheus + Grafana + Kibana：

```bash
cp config/.env.example config/.env
docker compose up -d --build
curl http://localhost:8080/healthz
```

首次启动会自动建立四个数据库并导入演示数据。可直接访问业务端口 `8080`，也可通过 Nginx 网关 `8888` 访问；指标端口是 `4000`。

| 服务 | 地址 |
|---|---|
| 业务 API | `http://localhost:8080` |
| Nginx 网关 | `http://localhost:8888` |
| Metrics | `http://localhost:4000/metrics` |
| Jaeger | `http://localhost:16686` |
| Prometheus | `http://localhost:9091` |
| Grafana | `http://localhost:3001` |
| Asynqmon | `http://localhost:8980` |
| Kibana | `http://localhost:5601` |

## 本地开发（Go 本地跑，Docker 只跑基础设施）

日常开发时只把 MySQL、Redis、Kafka、ES 等服务放在 Docker 里，Go 代码本地 `go run`，方便断点调试和快速迭代。

### 1. 启动基础设施

```bash
# 只启动数据库和中间件，不启动 app 容器
docker compose up -d mysql redis kafka elasticsearch jaeger prometheus grafana asynqmon kibana filebeat go-stash
```

### 2. 本地运行应用

```bash
# 设置环境变量指向 localhost，然后启动
MYSQL_HOST=localhost:33069 \
REDIS_ADDR=localhost:36379 \
REDIS_PASSWORD=G62m50oigInC30sf \
KAFKA_BROKER=localhost:9094 \
JAEGER_ENDPOINT=http://localhost:14268/api/traces \
ELASTICSEARCH_URL=http://localhost:9200 \
go run ./cmd/server
```

> 一行版：`env MYSQL_HOST=localhost:33069 REDIS_ADDR=localhost:36379 REDIS_PASSWORD=G62m50oigInC30sf KAFKA_BROKER=localhost:9094 JAEGER_ENDPOINT=http://localhost:14268/api/traces ELASTICSEARCH_URL=http://localhost:9200 go run ./cmd/server`

### 3. 验证

```bash
# 健康检查
curl http://localhost:8080/healthz

# 用户登录（演示账号 18888888888 / 123456）
curl -X POST http://localhost:8080/usercenter/v1/user/login \
  -H "Content-Type: application/json" \
  -d '{"mobile":"18888888888","password":"123456"}'

# 管理后台登录（admin / Admin@123）
curl -X POST http://localhost:8080/admin/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"Admin@123"}'

# 运行测试
go test -race ./...
```

首次启动会自动建立四个数据库并导入演示数据。如果是在已有数据卷上升级，需要手动执行新增迁移：

```bash
mysql -h 127.0.0.1 -P 33069 -u root -p < migrations/006_seckill.sql
mysql -h 127.0.0.1 -P 33069 -u root -p < migrations/y01_admin_rbac.sql
mysql -h 127.0.0.1 -P 33069 -u root -p < migrations/y02_search.sql
```

### 环境变量说明

本地开发运行时，以下环境变量需要从容器地址改为 `localhost`：

| 变量 | Docker Compose 值 | 本地开发值 |
|---|---|---|
| `MYSQL_HOST` | `mysql:3306` | `localhost:33069` |
| `REDIS_ADDR` | `redis:6379` | `localhost:36379` |
| `KAFKA_BROKER` | `kafka:9092` | `localhost:9094` |
| `JAEGER_ENDPOINT` | `http://jaeger:14268/...` | `http://localhost:14268/...` |
| `ELASTICSEARCH_URL` | `http://elasticsearch:9200` | `http://localhost:9200` |

管理后台首次启动会创建 `admin / Admin@123` 超级管理员。它只用于本地演示，部署前必须通过 `ADMIN_INITIAL_USER`、`ADMIN_INITIAL_PASSWORD` 和 `ADMIN_JWT_SECRET` 更改。

微信登录和微信支付需要在 `.env` 中提供真实配置；未配置不会影响其他业务启动，支付接口会返回明确的"未配置"错误。

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
