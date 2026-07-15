# Gin Looklook 全项目核心实现练习

当前分支：`practice/full-project-core`

这个分支保留了路由、DTO、数据模型、Repository 和基础设施骨架，但将最能帮助理解项目的业务实现替换为 `TODO(practice-XX)`。每个阶段都有失败测试：你的目标是让当前阶段变绿，而不是一次修完全部。

## 使用方式

```bash
./practice/verify.sh list
./practice/verify.sh 01
```

完成一阶段后再进入下一阶段。先读 Handler -> Service -> Repository -> Model，再写代码。如果实在卡住，主分支保留了原实现，但建议至少先独立思考 30 分钟。

## 两层验收方式

每个核心实现都先由包内单元测试检查边界，再由真实 HTTP E2E 检查路由、中间件、MySQL、Redis、Kafka、Elasticsearch 和 Worker 的组合行为。E2E 不会直接伪造用户 ID：每个业务场景都会先注册一个或多个虚构用户，再明确调用登录接口申请 JWT，最后才携带 `Authorization: Bearer <token>` 操作订单等资源。

```bash
./practice/verify.sh e2e-auth
./practice/verify.sh e2e-travel
./practice/verify.sh e2e-order
./practice/verify.sh e2e-payment
./practice/verify.sh e2e-search
./practice/verify.sh e2e-admin
./practice/verify.sh e2e-metrics
./practice/verify.sh e2e
```

脚本默认构建并启动 Docker Compose；若服务已经由你启动，可用 `E2E_STACK_READY=1` 跳过构建。手机号按场景动态生成，订单备注带 `practice-` 前缀，方便在数据库中识别。为了不误删你自己的开发数据，普通 E2E 不做全表清理；秒杀压测只重置隔离活动 `9001`。

## 01：用户登录与 JWT

位置：`internal/user/service.go`

- 通过 Repository 查询手机号。
- 区分用户不存在与 DB 异常。
- 校验密码并签发带 `jwtUserId/iat/exp` 的 JWT。
- 错误信息不能帮助攻击者枚举账号。

面试要点：为什么鉴权不等于授权？现有 MD5 密码方案有什么问题，如何迁移到 bcrypt/Argon2？

E2E：`./practice/verify.sh e2e-auth`，覆盖注册后再次登录取 Token、受保护接口、伪造 Token、错误密码、重复注册和资料持久化。

## 02：订单时间、快照与计价

位置：`internal/order/service.go`

- 至少住一晚，23:59 必须失败。
- 民宿信息复制到订单快照，避免房源修改影响历史订单。
- 住宿价、餐费和总价全程使用“分”。
- 普通下单与秒杀下单复用同一时间规则。

面试要点：金额为什么不用 float？订单为什么要保存商品快照？

## 03：取消订单状态机

位置：`internal/order/service.go` 和 `internal/order/repository.go`

- 实现合法迁移白名单。
- 理解用户归属校验、幂等短路和 `version` 乐观锁。
- 额外任务：为 `UpdateState` 补充 mock Repository 测试，验证他人订单、重复取消和版本冲突。

面试要点：先查后改为什么仍有并发竞态？乐观锁与悲观锁如何选择？

E2E：`./practice/verify.sh e2e-order`，覆盖未登录下单、非法日期、订单快照计价、列表、详情越权、取消越权、幂等取消和数据库版本号。

## 04：支付前置校验与渠道状态

位置：`internal/payment/service.go`

- 只有待支付、金额大于 0 的订单可以支付。
- 将微信状态映射为内部状态，未知值 fail closed。
- 额外任务：识别“检查订单后、创建支付流水前”的竞态窗口。

面试要点：回调为什么会重复？验签、金额校验、幂等分别防什么？

E2E：`./practice/verify.sh e2e-payment`。因为真实支付成功回调必须由微信使用平台证书签名，练习脚本不会伪造它；这里验证未登录、他人订单、已取消订单均不可支付，并验证失败路径不会产生支付流水。

## 05：Worker 启动与支付消息

位置：`docker-compose.yml`、`Dockerfile`、`internal/worker/worker.go`

- 修复 `api worker` 错误进程，实际运行 `worker` 二进制。
- 将支付成功映射为待使用，退款映射为已退款，其他消息不更新订单。
- 理解“处理成功后再 commit offset”。

面试要点：至少一次投递为什么要求消费者幂等？

## 06：秒杀 Lua、Stream 与补偿

位置：`internal/seckill/service.go`、`internal/worker/worker.go`、`internal/order/repository.go`

- 预约号长度固定且可唯一。
- Reserve Lua 原子完成活动检查、一人一单、扣库存、初始化结果、写 Stream。
- Complete/Compensate 脚本必须幂等，不能重复恢复库存。
- MySQL 条件更新和唯一索引是最终防超卖边界。

验收：先运行单元测试，再运行 `./practice/verify-seckill.sh`。

压测会先注册并登录 9 个虚构用户：同一 Token 并发请求验证一人一单，随后 8 个不同 Token 抢剩余库存，并轮询异步结果，最后同时核对 Redis 库存、MySQL 售出量、订单数和唯一用户数。

## 07：Elasticsearch 文档和排序 DSL

位置：`internal/search/service.go`

- 清洗中英文逗号标签。
- 只允许排序白名单，防止客户端任意控制 DSL。
- 地理排序只在坐标合法时启用。
- 任何排序都加 `id` 作稳定次序，避免分页抖动。

面试要点：为什么搜索不直接查 MySQL？MySQL 与 ES 如何最终一致？

E2E：`./practice/verify.sh e2e-search`，先取得用户 Token，再检查非法价格范围、标签过滤、城市过滤和 ES 最终可见性。

## 08：RBAC 与数据权限

位置：`internal/admin/service.go`、`internal/admin/repository.go`

- 区分功能权限与数据范围。
- 根据商家 ID/关联用户构造参数化 SQL 条件。
- 无授权信息或空范围时 deny by default。
- 不得将 ID 拼接进 SQL。

面试要点：菜单权限为什么不等于数据权限？权限缓存如何失效？

E2E：`./practice/verify.sh e2e-admin`，分别取得普通用户 Token 和管理员 Token，验证两种身份不可混用，再检查权限与超级管理员数据范围。

## 09：可观测性与可靠性指标

位置：`internal/shared/practice_metrics.go`。调用点已接入 Order/Worker，你需要理解它们何时记录成功、拒绝、失败，以及 Outbox 统计为什么不能只看本批最多 100 条待发送记录。

- CounterVec：`gin_looklook_order_transitions_total{from,to,result}`。
- Gauge：`gin_looklook_payment_outbox_pending`。
- Gauge：`gin_looklook_payment_outbox_oldest_age_seconds`。
- label 只用有界枚举，不得使用 userID/orderSN/error message。
- 额外任务：将 trace context 注入 Kafka header，并在消费端提取。

面试要点：Metrics/Logs/Traces 如何分工？什么是高基数 label？

E2E：`./practice/verify.sh e2e-metrics`，真实创建并取消订单后检查 pending -> canceled 成功迁移以及 Outbox Gauge 是否暴露。

## 最终验收

```bash
./practice/verify.sh all
rg -n 'TODO\(practice-' internal docker-compose.yml
```

所有测试通过，产品代码中没有剩余的 `TODO(practice-XX)`，然后手动跑一次登录、普通下单、取消、秒杀与搜索链路。

最后还要将 `Dockerfile` 中为分阶段练习保留的 `go test -run '^$' ./...` 恢复为 `go test ./...`，保证正式镜像在构建时执行全部测试。
