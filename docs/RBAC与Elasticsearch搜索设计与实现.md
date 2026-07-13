# RBAC 与 Elasticsearch 搜索设计与实现

本文只描述项目中已经实现并运行验证的管理后台权限和民宿搜索能力。

## 一、整体链路

```text
管理端请求
  -> 独立 Admin JWT
  -> 权限码校验（Redis 缓存，MySQL 回源）
  -> 数据范围拼接
  -> 业务事务：更新 homestay + 写 search_event_outbox
  -> 审计中间件记录结果、耗时和脱敏请求体

后台 Search Outbox Worker
  -> 拉取待同步事件
  -> 读取民宿最新状态
  -> Elasticsearch 幂等 upsert/delete
  -> 成功标记完成；失败指数退避

公开搜索接口
  -> keyword/city/price/tags/star/geo filters
  -> distance/price/star/newest 多条件排序
  -> Elasticsearch 返回分页结果
```

## 二、RBAC 模型

核心关系是 `admin_user -> admin_user_role -> admin_role -> admin_role_permission -> admin_permission`。权限粒度采用稳定权限码，例如：

- `admin:user:create`
- `admin:role:configure`
- `travel:homestay:update`
- `search:index:rebuild`

路由显式声明所需权限码，不能只靠前端隐藏按钮。`RequirePermission` 从当前管理员的有效权限集合判断，不通过时返回 HTTP 403 和业务码 `100004`。

管理员使用独立的 Admin JWT，Claim 包含 `adminId`、`username` 和 `tokenType=admin`，签名密钥和有效期也与普通用户 Token 分开，避免业务用户 Token 被管理端误接受。管理员密码使用 bcrypt；首次启动时若管理表为空，会创建初始超级管理员并绑定 `super_admin` 角色。生产环境必须通过环境变量更改初始账号、密码和 JWT 密钥。

### 权限缓存与失效

有效权限和数据范围以 `gin:looklook:rbac:v1:admin:{id}` 缓存在 Redis，TTL 为 5 分钟。缓存只是一层加速，MySQL 是事实来源。以下变更会主动删除受影响管理员的缓存：

- 重新分配用户角色；
- 修改管理员状态、所属商家或关联业务用户；
- 修改角色状态、权限集合或数据范围。

因此权限收回不需要等待 TTL 自然过期。

## 三、四种数据范围

角色的 `scope_type` 支持：

| 值 | 范围 | 生成的条件 |
|---|---|---|
| 1 | 全部数据 | 不附加范围条件 |
| 2 | 所属商家 | `homestay_business_id = admin.business_id` |
| 3 | 自定义商家 | 商家 ID 来自 `admin_role_data_scope` |
| 4 | 本人数据 | `user_id = admin.linked_user_id` |

同一管理员可以有多个角色，范围按并集合并；任一角色拥有“全部数据”即直接放行。没有任何有效范围时使用 `AND 1=0` 默认拒绝，而不是因为条件为空意外查出全表。

数据权限不由 Handler 自己判断。Service 先加载 `AdminAuthorization`，Repository 再把范围条件应用到管理端民宿列表和更新 SQL。更新操作先在事务中以 `SELECT ... FOR UPDATE` 验证目标属于当前范围，再进行带 `version` 的乐观锁更新。

## 四、操作审计

管理端保护路由统一经过 `AdminAudit` 中间件。每条记录包含：

- 管理员 ID、用户名和权限码；
- HTTP 方法、路径、请求 ID、来源 IP；
- HTTP 状态、成功标志和耗时；
- 请求体和公开错误信息；
- 创建时间。

`password`、`token`、`secret` 等字段递归替换为 `***`。审计写入使用独立的短超时后台 Context，不受 HTTP 请求结束影响；写入失败会记录应用错误日志，但不会把已成功的业务请求改成失败。审计列表支持管理员、权限码、RFC3339 时间区间和分页过滤。

## 五、Elasticsearch 索引

应用启动时检查并创建 `gin-looklook-homestay-v1` 索引。主要字段映射为：

- `title/subTitle/info`：`text`，用于关键词检索；
- `city/tags`：`keyword`，用于精确过滤；
- `star`：`float`；
- `homestayPrice`：`long`，单位仍是分；
- `location`：`geo_point`；
- `rowState`：上架状态过滤；
- `id/version`：稳定文档 ID 和版本信息。

MySQL 的 `tags` 使用逗号分隔保存，写索引时转换成字符串数组；搜索结果再还原为兼容原接口的字符串。经纬度有效时才写 `geo_point`，避免把缺失坐标误认为真实的 `(0,0)`。

公开接口为：

```http
POST /travel/v1/search/homestays
Content-Type: application/json

{
  "keyword": "西湖",
  "city": "杭州",
  "minPrice": 200,
  "maxPrice": 500,
  "tags": ["亲子"],
  "minStar": 4.5,
  "latitude": 30.25,
  "longitude": 120.16,
  "distanceKm": 10,
  "sortBy": ["distance", "price_asc"],
  "page": 1,
  "pageSize": 10
}
```

价格在 API 使用元，进入领域模型后转换为分。搜索固定过滤 `rowState=1`，只返回上架房源。排序支持：

- `distance` / `distance_asc`
- `price_asc`
- `price_desc`
- `star_desc`
- `newest`

可以组合多个排序条件，最后追加 `id desc` 保证相同排序值下结果稳定。未指定排序时默认按评分和 ID 倒序。

## 六、为什么索引更新使用 Outbox

错误做法是在数据库更新成功后同步调用 Elasticsearch：如果进程此时退出，MySQL 已提交但索引永远还是旧值；如果先写 Elasticsearch，则数据库事务回滚时索引又会提前变化。

本项目在同一个 `looklook_travel` MySQL 事务内完成：

1. 校验数据范围并锁定民宿；
2. 用 `version` 乐观锁更新民宿；
3. 写入唯一事件键 `homestay:{id}:v{version}` 的 `search_event_outbox`；
4. 提交事务；
5. 删除民宿详情缓存。

Worker 每秒扫描到期事件，根据民宿最新状态执行 Elasticsearch upsert；若记录已删除则删除文档。ES 文档 ID 直接使用民宿 ID，因此重复投递是幂等的。写入成功后把事件标为完成；失败按指数退避，最长 5 分钟后重试。

这保证的是“至少一次 + 最终一致”，不是跨 MySQL/ES 的强一致。极短时间内搜索可能读到旧索引，这是为了不让搜索系统故障阻塞核心管理事务而做的明确取舍。

首次启动会为现有房源补写 `bootstrap:{id}:v{version}` 事件。管理端还提供 `/admin/v1/search/rebuild`，按一次性 token 为所有现存房源重新入箱，可用于索引重建和故障恢复。

## 七、管理接口

登录接口无需 Admin Token：

- `POST /admin/v1/auth/login`

其余接口均需要 `Authorization: Bearer <admin-token>`：

- 用户：`user/list`、`user/create`、`user/update`、`user/assignRoles`
- 角色：`role/list`、`role/create`、`role/configure`
- 权限：`permission/list`、`permission/create`
- 审计：`audit/list`
- 民宿：`homestay/list`、`homestay/update`
- 搜索恢复：`search/rebuild`

详细字段可以直接查看 `internal/httpapi/types.go`；每个接口的权限码见 `migrations/y01_admin_rbac.sql`。

## 八、面试时怎么讲

可以用下面这段作为主线：

> 我做的不是只有用户、角色三张表的演示 RBAC。路由层强制权限码，Redis 缓存有效授权并在配置变更时精确失效；数据层支持全部、所属商家、自定义商家和本人四种范围，多个角色按并集合并且默认拒绝。管理操作统一审计并对密码脱敏。民宿更新和搜索事件在一个 MySQL 事务提交，由 Outbox Worker 最终同步 Elasticsearch，所以 ES 故障不会造成业务更新丢失。搜索侧支持关键词、城市、价格、标签、评分、地理距离以及稳定的多条件排序。

常见追问：

### 为什么不直接从 MySQL 做全部搜索？

城市和价格可用普通索引，但关键词相关性、多标签组合、地理距离和多维排序会让 SQL 越来越复杂，且 `%LIKE%` 难以利用 B-Tree。ES 负责读模型和检索，MySQL 仍是事实来源。

### 为什么数据范围放在 Repository？

权限中间件只能回答“能否调用接口”，不能回答“能看哪些行”。把范围最终应用在 SQL 层，避免 Service 先查全量再在内存过滤造成越权和性能问题。

### Outbox 是否会重复？

会。Worker 可能已经写入 ES、但还没标记事件完成就宕机。重试仍写同一个文档 ID，所以结果幂等。设计目标是允许重复、不能丢失。

### 多副本 Worker 怎么继续加强？

当前重复认领仍是安全的，因为 ES upsert 幂等；更高并发下可增加 `processing` 状态、租约时间和 `SELECT ... FOR UPDATE SKIP LOCKED`，减少重复工作。

## 九、已完成的真实验证

- 初始超级管理员自动创建并成功登录；
- 无权限账号访问用户列表得到 HTTP 403；
- 商家范围为 `999` 时民宿列表为 0，改为 `1` 并主动失效缓存后立即查到对应房源；
- 失败和成功操作均写入审计，创建用户请求中的密码保存为 `***`；
- 管理端更新民宿后生成 `homestay:11:v1` Outbox 事件；
- Worker 将事件标记完成，ES 文档从 `version=0` 更新到 `version=1`；
- 城市、200～400 元、亲子标签、4.5 分以上、10km 内、距离+价格排序的组合搜索返回正确房源；
- `go test ./...`、`go vet ./...` 和 Docker 镜像内测试构建通过。
