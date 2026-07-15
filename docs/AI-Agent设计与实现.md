# AI Agent 设计与实现

## 业务目标

项目使用 CloudWeGo Eino 实现了一个统一的民宿智能客服。登录用户可以用自然语言完成：

* 按城市、每晚预算、人数、标签、评分和关键词推荐真实民宿
* 最近有哪些订单、订单处于什么状态
* 某个订单的入住时间、金额和民宿信息
* 某个公开民宿的城市、容量和价格

第一版刻意限定为只读助手。取消订单、支付、退款等操作仍由确定性的业务接口完成，避免模型误调用产生真实副作用。

## 请求链路

```text
POST /agent/v1/chat
        |
        v
JWT 认证 -> 按用户限流 -> Assistant Handler
                              |
                              v
                    Eino ChatModelAgent
                    /        |         \
                   v         v          v
        recommend_homestays  订单工具   get_homestay
                   |         |          |
                   v         v          v
            Search Service  Order     Travel
                   |
                   v
             Elasticsearch
```

Eino 的 `ChatModelAgent` 负责 ReAct 循环：模型判断是否需要工具、生成结构化工具参数、接收工具结果，再组织最终回答。项目没有让模型直接访问数据库，而是复用现有 Service，保留原来的业务边界和查询规则。

## 四个只读工具

| 工具 | 输入 | 数据来源 | 安全边界 |
| --- | --- | --- | --- |
| `list_my_orders` | `page_size` | Order Service | 用户 ID 由 JWT 上下文绑定 |
| `get_my_order` | `order_sn` | Order Service | Repository 继续校验订单归属 |
| `get_homestay` | `homestay_id` | Travel Service | 只返回公开民宿信息 |
| `recommend_homestays` | 城市、每晚预算、人数、偏好等 | Search Service | 只查询已上架房源，最多返回 5 个候选 |

工具参数里没有 `user_id`。它由服务端在创建工具闭包时注入，因此即使用户提示“查询其他用户的订单”，模型也无法越权指定目标用户。这比只在系统提示词中要求模型守规矩更可靠：提示词负责行为引导，代码负责权限边界。

返回给模型的数据也经过裁剪，只包含回答所需字段，金额从分转换成元，不暴露手机号、密码或内部数据库字段。

推荐工具复用现有 Elasticsearch 搜索，先按城市、价格、标签、评分和关键词查询，再按民宿容量过滤入住人数。模型只负责从真实候选中解释“为什么适合”，不能生成索引中不存在的民宿。推荐不检查指定日期库存，因此工具输出和系统提示都会要求客服提醒用户在下单时确认房态及最终价格。

## 配置与调用

```dotenv
AI_API_KEY=
AI_BASE_URL=
AI_MODEL=gpt-4.1-mini
AI_TIMEOUT_SECONDS=20
```

`AI_API_KEY` 为空时 Agent 不初始化，但主服务正常启动，调用接口会收到 `503`。`AI_BASE_URL` 为空时使用模型组件默认地址；也可以配置兼容 OpenAI 协议的服务。

为了兼容常见的模型配置，代码也接受 `OPENAI_API_KEY`、`OPENAI_BASE_URL` 和 `OPENAI_MODEL`。若两套变量同时存在，`AI_*` 的优先级更高。Go 程序本身不会自动读取根目录 `.env`；项目的 `make dev` 和 `make docker` 会在文件存在时加载它，直接执行 `go run` 则需要先手动导出变量。

取得用户 Token 后调用：

```bash
curl -X POST http://localhost:8080/agent/v1/chat \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"question":"推荐杭州适合 4 人入住、每晚 500 元以内的亲子民宿"}'
```

接口按登录用户独立限流，单次运行有超时，并限制 Agent 最大迭代次数，避免循环工具调用导致延迟和费用失控。当前限流器位于进程内；多实例部署时应改为 Redis 或网关限流。

## 测试重点

单元测试使用假的 Tool Calling 模型，不依赖外部 API，也不产生模型费用。测试会让模型先发出工具调用，再检查工具结果是否回到模型，最后生成回答，从而覆盖完整的 ReAct 工具循环。此外还验证：

* 四个工具的结构化输入输出
* 订单工具始终使用已认证用户 ID
* 推荐条件正确转换为搜索条件，并过滤容量不足的民宿
* 金额单位转换
* 未配置密钥时返回 `503`
* 限流按用户隔离，突发额度耗尽后返回 `429`

## 面试讲解要点

可以按下面的顺序介绍：

1. 为什么做 Agent：同一入口既能理解推荐需求，也能处理订单查询，避免为每种问法设计接口。
2. 为什么用工具调用：LLM 负责意图理解，真实业务数据仍由确定性的 Go 服务提供。
3. 推荐为什么可信：候选来自 Elasticsearch 真实索引，模型只做筛选解释，并明确不承诺实时库存。
4. 如何防越权：用户 ID 不交给模型，由 JWT 上下文在服务端绑定；订单详情仍执行归属校验。
5. 如何控制风险：第一版只读、限制输入长度、候选数量、按用户限流、超时和最大迭代次数。
6. 如何测试：用假模型固定“工具调用 -> 工具结果 -> 最终回答”的过程，不让单元测试依赖网络和随机输出。

后续可以增加对话历史、Redis 分布式限流、模型调用指标和离线评测集。若增加写工具，应设计“模型生成操作草案 -> 用户明确确认 -> 服务端重新鉴权和幂等执行”的两阶段流程，不能允许模型一次调用直接修改订单。

## 参考资料

* [CloudWeGo Eino](https://github.com/cloudwego/eino)
* [Eino ChatModelAgent 官方文档](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_implementation/chat_model/)
