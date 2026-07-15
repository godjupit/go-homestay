#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/.."

stage="${1:-list}"

list() {
  printf '%s\n' \
    '01  用户登录与 JWT' \
    '02  订单时间、快照与计价' \
    '03  订单状态机' \
    '04  支付状态与支付前置校验' \
    '05  Worker 启动与支付消息映射' \
    '06  秒杀预约号（集成压测需另跑 verify-seckill.sh）' \
    '07  Elasticsearch 标签与排序 DSL' \
    '08  RBAC 数据范围' \
    '09  Prometheus 可靠性指标' \
    'e2e-auth     真实注册、登录、JWT 与用户资料' \
    'e2e-travel   民宿详情、列表、金额与异常资源' \
    'e2e-order    真实下单、详情、越权与取消' \
    'e2e-payment  支付鉴权、订单归属与失败无副作用' \
    'e2e-search   真实 ES 查询与参数校验' \
    'e2e-admin    管理员 JWT、RBAC 与数据范围' \
    'e2e-metrics  真实状态迁移与 Prometheus 指标' \
    'e2e          执行全部 HTTP E2E' \
    'all 全部单元测试与 TODO 检查'
}

verify_worker_entrypoint() {
  local actual
  actual=$(docker compose --env-file config/.env.docker config --format json | jq -r '.services.worker.entrypoint[0] // ""')
  if [[ "$actual" != "worker" && "$actual" != "/usr/local/bin/worker" ]]; then
    printf 'Worker entrypoint is %q; want worker.\n' "$actual" >&2
    return 1
  fi
}

case "$stage" in
  list) list ;;
  01) go test ./internal/user -run '^TestPracticeLogin' -v ;;
  02) go test ./internal/order -run '^TestPractice(StayNights|BuildOrder)' -v ;;
  03) go test ./internal/order -run '^TestPracticeOrderStateMachine$' -v ;;
  04) go test ./internal/payment -run '^TestPractice' -v ;;
  05) go test ./internal/worker -run '^TestPracticeOrderStateForPayment$' -v; verify_worker_entrypoint ;;
  06) go test ./internal/seckill -run '^TestPracticeReservationSN$' -v ;;
  07) go test ./internal/search -run '^TestPractice' -v ;;
  08) go test ./internal/admin -run '^TestPractice' -v ;;
  09) go test ./internal/shared -run '^TestPracticeReliabilityMetrics$' -v ;;
  e2e-auth) ./practice/verify-e2e.sh auth ;;
  e2e-travel) ./practice/verify-e2e.sh travel ;;
  e2e-order) ./practice/verify-e2e.sh order ;;
  e2e-payment) ./practice/verify-e2e.sh payment ;;
  e2e-search) ./practice/verify-e2e.sh search ;;
  e2e-admin) ./practice/verify-e2e.sh admin ;;
  e2e-metrics) ./practice/verify-e2e.sh metrics ;;
  e2e) ./practice/verify-e2e.sh all ;;
  all)
    go test -race ./...
    if rg -n 'TODO\(practice-' internal docker-compose.yml; then
      printf 'Practice TODOs remain.\n' >&2
      exit 1
    fi
    ;;
  *) printf 'Unknown stage: %s\n\n' "$stage" >&2; list >&2; exit 2 ;;
esac
