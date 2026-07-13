# Gin Homestay

一个基于 Go + Gin 构建的高性能民宿预订系统后端。

项目最初来源于 `go-zero-looklook`，后经过独立 Gin 架构重构，保留原有业务模型，同时引入现代互联网后端常见技术方案：

* 分层架构设计
* JWT 用户认证
* RBAC 权限管理
* Redis 缓存
* Kafka 异步消息
* Asynq 后台任务
* Elasticsearch 搜索
* 秒杀库存控制
* Prometheus 指标监控
* OpenTelemetry 链路追踪
* Docker Compose 一键部署

## 项目定位

`gin-homestay` 是一个模拟真实在线民宿平台的后端服务。

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

项目目标不是简单实现业务接口，而是实践企业后端开发中常见的工程能力：

* 如何组织 Go 项目结构
* 如何进行业务分层
* 如何处理高并发场景
* 如何设计异步任务
* 如何接入监控体系
* 如何进行服务治理

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
│   └── server
│       └── main.go              # 服务启动入口
│
├── internal
│   ├── app
│   │   └── app.go               # 应用初始化
│   │
│   ├── config
│   │   └── config.go            # 配置管理
│   │
│   ├── httpapi
│   │   ├── router.go            # Gin路由
│   │   ├── middleware.go        # 中间件
│   │   └── admin.go             # 管理后台接口
│   │
│   ├── service                  # 业务层
│   │   ├── user.go
│   │   ├── travel.go
│   │   ├── order.go
│   │   ├── payment.go
│   │   ├── seckill.go
│   │   └── search.go
│   │
│   ├── repository               # 数据访问层
│   │   ├── user
│   │   ├── order
│   │   └── search
│   │
│   ├── model                    # 数据模型
│   │
│   ├── worker                   # 异步任务
│   │
│   └── platform                 # 基础能力
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
http://localhost:4000/metrics
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

# 本地启动

## 方式一：Docker 全家桶

启动：

```bash
cp config/.env.example config/.env

docker compose up -d --build
```

检查：

```bash
curl http://localhost:8080/healthz
```

---

## 方式二：本地运行 Go

启动基础设施：

```bash
docker compose up -d mysql redis kafka elasticsearch
```

运行：

```bash
go run ./cmd/server
```

测试：

```bash
go test -race ./...
```

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
 "status":"ok"
}
```

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

* 引入 Wire 依赖注入
* Repository 接入 GORM
* 服务拆分微服务

## 性能

* Redis Cluster
* Kafka分区优化
* 数据库读写分离

## 工程化

* CI/CD
* Kubernetes部署
* 自动化测试

---

# 学习路线建议

如果你刚开始学习 Go 后端，可以按照：

```
1. cmd/server/main.go
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
