# Gin LookLook Homestay

一个基于 Go + Gin 构建、强调业务一致性和工程实践的民宿预订系统后端。

项目最初来源于 `go-zero-looklook`，后经过独立 Gin 架构重构，保留原有业务模型，同时引入现代互联网后端常见技术方案：

* 分层架构设计
* JWT 用户认证
* RBAC 权限管理
* Redis 缓存
* Kafka 异步消息
* Asynq 后台任务
* Elasticsearch 搜索
* 秒杀库存控制
* Eino AI 智能客服与民宿推荐
* Prometheus 指标监控
* OpenTelemetry 链路追踪
* Docker Compose 一键部署

## 项目定位

`gin-looklook` 是一个模拟真实在线民宿平台的模块化单体后端服务。

核心业务包括：

* 用户系统
* 民宿浏览
* 旅行内容管理
* 订单系统
* 支付流程
* 管理后台
* 权限控制
* 民宿搜索
* 秒杀活动
* AI 民宿推荐与订单查询客服

项目目标不是简单实现业务接口，而是实践企业后端开发中常见的工程能力：

* 如何组织 Go 项目结构
* 如何进行业务分层
* 如何处理高并发场景
* 如何设计异步任务
* 如何接入监控体系
* 如何进行服务治理

## 当前工程质量

仓库提供统一质量门禁、真实业务演示和空库迁移校验：

```bash
make verify           # 格式、静态检查、竞态测试、构建、Compose 配置
make migration-check  # 在临时 MySQL 中顺序执行全部迁移
make demo             # 注册、登录、读取民宿、下单、取消的完整链路
make benchmark-seckill # 隔离活动上的秒杀并发与库存一致性验证
```

`main` 分支应始终保持这些检查通过；带有故意失败测试和 TODO 的教学内容放在独立练习分支，不合并到主分支。

详细设计文档：

* [架构与业务实现](docs/架构与业务实现.md)
* [秒杀机制设计与实现](docs/秒杀机制设计与实现.md)
* [RBAC 与 Elasticsearch](docs/RBAC与Elasticsearch搜索设计与实现.md)
* [技术亮点与面试讲解](docs/技术亮点与面试讲解.md)
* [秒杀压测方法与报告](docs/秒杀压测报告.md)
* [AI Agent 设计与实现](docs/AI-Agent设计与实现.md)

---

# 技术栈

## 后端

| 技术            | 用途                 |
| ------------- | ------------------ |
| Go 1.22       | 后端开发语言             |
| Gin           | HTTP Web Framework |
| JWT           | 用户认证               |
| MySQL         | 核心业务数据库            |
| Redis         | 缓存、分布式锁、秒杀库存       |
| Kafka         | 消息队列               |
| Asynq         | 异步任务系统             |
| CloudWeGo Eino | AI Agent 与工具调用      |
| Elasticsearch | 全文搜索               |
| Prometheus    | 指标采集               |
| Grafana       | 数据可视化              |
| Jaeger        | 链路追踪               |
| OpenTelemetry | 可观测性标准             |

## 基础设施

Docker Compose 提供：

* MySQL
* Redis
* Kafka
* Elasticsearch
* Prometheus
* Grafana
* Jaeger
* Kibana
* Nginx
* Asynq Monitor

---

# 项目架构

项目采用经典 Go 后端分层架构：

```
                 Client
                   |
                   |
              Gin Router
                   |
        Middleware / JWT / Recovery
                   |
        -------------------------
        |           |           |
     User       Travel       Order
     Service    Service      Service

                   |
              Repository

                   |
        ------------------------
        |          |           |
      MySQL      Redis       Kafka

                   |
              Async Worker

                   |
              Asynq Tasks
```

核心设计思想：

> Handler 负责协议处理，Service 负责业务逻辑，Repository 负责数据访问。

---

# 目录结构

```
.
├── cmd
│   ├── api
│   │   └── main.go              # HTTP API 启动入口
│   └── worker
│       └── main.go              # 异步 Worker 启动入口
│
├── internal
│   ├── bootstrap                  # 应用初始化
│   │
│   ├── httpserver                 # Gin 路由与中间件
│   │
│   ├── shared                     # 配置、数据库、可观测性等共享能力
│   │
│   ├── user                       # 用户领域
│   ├── assistant                  # Eino AI 智能客服
│   ├── travel                     # 民宿领域
│   ├── order                      # 订单领域
│   ├── payment                    # 支付领域
│   ├── seckill                    # 秒杀领域
│   ├── search                     # 搜索领域
│   ├── admin                      # 管理后台与 RBAC
│   └── worker                     # 异步任务
│
├── migrations                   # 数据库迁移
│
├── deploy                       # 部署配置
│
├── docker-compose.yml
│
└── Dockerfile
```

---

# 核心业务设计

## 用户系统

功能：

* 手机号登录
* JWT Token认证
* 用户信息管理
* 权限验证

认证流程：

```
用户登录

      |
      v

验证账号密码

      |
      v

生成 JWT

      |
      v

请求携带 Token

      |
      v

Middleware解析用户身份
```

---

# 民宿业务

主要模块：

* 房源展示
* 旅行内容
* 民宿详情
* 订单创建

业务流程：

```
用户浏览民宿

       |

创建订单

       |

库存检查

       |

订单生成

       |

支付流程
```

---

# 秒杀系统设计

项目实现了完整秒杀流程：

```
用户请求秒杀

        |

Redis预扣库存

        |

发送Kafka消息

        |

异步创建订单

        |

最终扣减数据库库存
```

解决问题：

## 超卖问题

采用：

* Redis库存预扣
* 数据库库存校验
* 事务控制

## 高并发问题

采用：

* Redis挡流量
* MQ削峰
* Worker异步消费

---

# 搜索系统

使用 Elasticsearch 实现：

* 民宿关键词搜索
* 条件过滤
* 搜索索引同步

数据流程：

```
MySQL

 |

Outbox事件

 |

Kafka

 |

消费者

 |

Elasticsearch
```

避免：

* 数据库直接承担搜索压力
* 双写数据不一致

---

# RBAC权限系统

管理后台支持：

* 用户管理
* 角色管理
* 权限管理
* 数据范围控制

权限模型：

```
User

 |

Role

 |

Permission

 |

Resource
```

支持：

* RBAC权限控制
* 数据权限隔离
* 管理操作审计

---

# 可观测性设计

## Metrics

Prometheus采集：

* HTTP请求数量
* 请求耗时
* 错误率

访问：

```
http://localhost:8080/metrics
```

## Trace

使用 OpenTelemetry + Jaeger：

```
HTTP Request

      |

Gin Middleware

      |

Service

      |

Database / Redis
```

方便定位：

* 慢请求
* 服务瓶颈
* 调用链问题

---

# 配置说明

项目通过环境变量读取配置，**Go 程序不会自动加载 `.env` 文件**。请根据运行方式选择配置：

| 文件 | 用途 |
| --- | --- |
| `config/.env.example` | 完整的配置项模板，用于查看或创建自定义配置 |
| `config/.env.local` | Go 程序在宿主机运行时使用，中间件地址指向 `localhost` 的映射端口 |
| `config/.env.docker` | `docker compose --env-file` 使用的变量文件 |

如果根目录存在被 Git 忽略的 `.env`，`make dev` 和 `make docker` 会在上述基础配置之后加载它，适合保存本机的 AI 密钥等私有配置。

修改密码或 JWT 密钥时，应保证 API、Worker 和对应中间件使用相同的值。生产环境必须替换示例密码、`JWT_SECRET` 和 `ADMIN_JWT_SECRET`，不要将真实密钥提交到仓库。

当 `APP_ENV=production` 时，API 和 Worker 会在启动阶段拒绝短 JWT 密钥、相同的用户/管理员密钥以及默认管理员密码。开发环境允许示例值，便于本地一键启动。

AI 助手是可选能力。配置 `AI_API_KEY` 后启用，`AI_BASE_URL` 可指定 OpenAI 兼容接口，`AI_MODEL` 选择模型，`AI_TIMEOUT_SECONDS` 控制一次 Agent 请求的最长时间。项目也兼容常用的 `OPENAI_API_KEY`、`OPENAI_BASE_URL` 和 `OPENAI_MODEL` 变量名；同一项同时存在时优先使用 `AI_*`。未配置密钥时 API 仍可启动，助手接口返回 `503`。

常用地址：

| 服务 | 宿主机地址 |
| --- | --- |
| API | `http://localhost:8080` |
| Nginx 网关 | `http://localhost:8888` |
| Metrics | `http://localhost:8080/metrics` |
| MySQL | `localhost:33069` |
| Redis | `localhost:36379` |
| Kafka | `localhost:9094` |
| Elasticsearch | `http://localhost:9200` |
| Kibana | `http://localhost:5601` |
| Jaeger | `http://localhost:16686` |
| Prometheus | `http://localhost:9091` |
| Grafana | `http://localhost:3001` |
| Asynq Monitor | `http://localhost:8980` |

---

# 本地启动

## 方式一：Docker 全家桶

启动：

```bash
make docker
```

等价命令：

```bash
docker compose --env-file config/.env.docker up -d --build
```

检查：

```bash
curl http://localhost:8080/healthz
docker compose ps
```

执行完整业务演示：

```bash
make demo
```

如果服务已经启动，可避免重复构建：

```bash
DEMO_STACK_READY=1 make demo
```

> 不要依赖 `cp config/.env.example config/.env`：Docker Compose 默认只会自动读取项目根目录下的 `.env`，不会自动读取 `config/.env`。

---

## 方式二：本地运行 Go

启动基础设施：

```bash
docker compose --env-file config/.env.docker up -d mysql redis kafka elasticsearch jaeger
```

运行 API（`make dev` 会先加载 `config/.env.local`）：

```bash
make dev
```

如需处理 Kafka 消息和 Asynq 任务，在另一个终端启动 Worker：

```bash
make worker
```

不使用 Makefile 时，需要手动导出环境变量：

```bash
set -a
. config/.env.local
set +a
go run ./cmd/api
```

测试：

```bash
make verify
```

测试分为三层：包内单元测试验证状态与安全边界，`make demo` 验证真实 HTTP 主链路，`make migration-check` 验证空数据库初始化。GitHub Actions 会在每个 Pull Request 上执行同一套质量门禁。

---

# API 示例

## 用户登录

```http
POST /usercenter/v1/user/login
```

请求：

```json
{
 "mobile":"18888888888",
 "password":"123456"
}
```

---

## 健康检查

```http
GET /healthz
```

返回：

```json
{
  "code": 200,
  "msg": "OK",
  "data": {
    "status": "ok"
  }
}
```

## AI 智能客服

先登录并取得 Token。客服可以根据城市、人数、预算和偏好推荐真实民宿：

```bash
curl -X POST http://localhost:8080/agent/v1/chat \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"question":"推荐杭州适合 4 人入住、每晚 500 元以内的亲子民宿"}'
```

同一个接口也可以询问“我最近的订单是什么状态？”。客服将民宿推荐、公开民宿详情和本人订单查询集成在一起，但不执行取消、支付、退款等写操作。推荐候选来自 Elasticsearch 真实房源索引，指定日期库存和最终价格仍需在下单时确认。

---

# 开发规范

## 新增业务流程

推荐：

```
Router

 ↓

Handler

 ↓

Service

 ↓

Repository

 ↓

Database
```

不要：

```
Handler

直接操作数据库
```

---

# 后续优化方向

当前项目已经具备生产级项目雏形，可以继续优化：

## 架构

* 在模块数量继续增长时评估 Wire 依赖注入
* 明确模块边界，优先保持模块化单体
* 根据真实容量和团队规模决定是否拆分服务

## 性能

* Redis Cluster
* Kafka分区优化
* 数据库读写分离

## 工程化

* 增加关键 Handler、Worker 和支付回调的集成测试
* 将演示环境扩展为可选的预发布环境
* 根据实际部署需求评估 Kubernetes，而不是为技术栈而引入

---

# 学习路线建议

如果你刚开始学习 Go 后端，可以按照：

```
1. cmd/api/main.go
        |
        v
2. router.go
        |
        v
3. middleware.go
        |
        v
4. service层
        |
        v
5. repository层
        |
        v
6. worker异步任务
        |
        v
7. 秒杀和搜索设计
```

不要一开始研究所有 Docker、中间件。

先理解：

> 一个请求如何进入系统 → 如何经过业务层 → 如何访问数据库 → 如何返回结果。

---

# License

MIT
