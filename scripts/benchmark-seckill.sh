#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/.."

base_url="${BASE_URL:-http://localhost:8080}"
mysql_password="${MYSQL_PASSWORD:-PXDN93VRKUm8TeE7}"
redis_password="${REDIS_PASSWORD:-G62m50oigInC30sf}"
activity_id="${ACTIVITY_ID:-9001}"
stock="${SECKILL_STOCK:-5}"

for tool in curl jq docker; do
  command -v "$tool" >/dev/null || { printf '%s is required\n' "$tool" >&2; exit 1; }
done

if [[ "${BENCHMARK_STACK_READY:-0}" != "1" ]]; then
  docker compose --env-file config/.env.docker up -d --build api worker mysql redis kafka elasticsearch
fi

for _ in $(seq 1 60); do
  curl -fsS "$base_url/healthz" >/dev/null 2>&1 && break
  sleep 1
done
curl -fsS "$base_url/healthz" >/dev/null

worker_id=$(docker compose ps -q worker)
worker_path=$(docker inspect "$worker_id" --format '{{.Path}}')
[[ "$worker_path" == worker || "$worker_path" == /usr/local/bin/worker ]] || {
  printf 'worker container runs %q, want worker\n' "$worker_path" >&2
  exit 1
}

mysql() {
  docker compose exec -T mysql mysql -uroot -p"$mysql_password" "$@" 2>/dev/null
}

redis() {
  docker compose exec -T redis redis-cli -a "$redis_password" --no-auth-warning "$@"
}

printf '[setup] resetting isolated activity %s with stock %s\n' "$activity_id" "$stock"
mysql -e "
USE looklook_order;
DELETE h FROM homestay_order h
JOIN seckill_order s ON s.order_sn=h.sn
WHERE s.activity_id=$activity_id;
DELETE FROM seckill_order WHERE activity_id=$activity_id;
INSERT INTO seckill_activity(id,homestay_id,title,price,stock,sold_count,start_time,end_time,status)
VALUES($activity_id,11,'Reproducible Load Test',9900,$stock,0,DATE_SUB(NOW(),INTERVAL 1 HOUR),DATE_ADD(NOW(),INTERVAL 1 DAY),1)
ON DUPLICATE KEY UPDATE stock=$stock,sold_count=0,start_time=DATE_SUB(NOW(),INTERVAL 1 HOUR),end_time=DATE_ADD(NOW(),INTERVAL 1 DAY),status=1;
"

now=$(date +%s)
redis DEL \
  "gin:looklook:{seckill}:v1:activity:$activity_id" \
  "gin:looklook:{seckill}:v1:stock:$activity_id" \
  "gin:looklook:{seckill}:v1:users:$activity_id" >/dev/null
redis HSET "gin:looklook:{seckill}:v1:activity:$activity_id" startAt $((now-3600)) endAt $((now+86400)) status 1 >/dev/null
redis SET "gin:looklook:{seckill}:v1:stock:$activity_id" "$stock" EX 604800 >/dev/null

register_and_login() {
  local mobile="$1"
  local nickname="$2"
  local response
  response=$(curl -sS -X POST "$base_url/usercenter/v1/user/register" \
    -H 'Content-Type: application/json' \
    -d "{\"mobile\":\"$mobile\",\"password\":\"Benchmark@123\",\"nickname\":\"$nickname\"}")
  [[ "$(jq -r '.code // 0' <<<"$response")" == 200 ]] || { printf 'registration failed: %s\n' "$response" >&2; return 1; }
  response=$(curl -sS -X POST "$base_url/usercenter/v1/user/login" \
    -H 'Content-Type: application/json' \
    -d "{\"mobile\":\"$mobile\",\"password\":\"Benchmark@123\"}")
  jq -er '.data.accessToken' <<<"$response"
}

live_start=$((now+172800))
live_end=$((live_start+172800))
same_mobile=$(printf '133%08d' $((70000000 + (now + $$) % 10000000)))
same_token=$(register_and_login "$same_mobile" 'benchmark-same')

printf '%s\n' '[test] 10 concurrent requests from one user'
same_responses=$(seq 1 10 | xargs -P10 -I{} curl -sS -X POST "$base_url/order/v1/seckill/reserve" \
  -H "Authorization: Bearer $same_token" -H 'Content-Type: application/json' \
  -d "{\"activityId\":$activity_id,\"liveStartTime\":$live_start,\"liveEndTime\":$live_end,\"livePeopleNum\":1}")
same_unique=$(jq -s '[.[].data.reservationSn] | map(select(. != null)) | unique | length' <<<"$same_responses")
[[ "$same_unique" == 1 ]] || { printf 'same user produced %s reservations\n' "$same_unique" >&2; exit 1; }
[[ "$(redis GET "gin:looklook:{seckill}:v1:stock:$activity_id")" == $((stock-1)) ]] || {
  printf '%s\n' 'same user deducted inventory more than once' >&2
  exit 1
}

printf '%s\n' '[test] 8 users racing for the remaining inventory'
declare -a tokens=()
for i in $(seq 1 8); do
  mobile=$(printf '132%08d' $((70000000 + (now + $$ + i) % 10000000)))
  tokens+=("$(register_and_login "$mobile" "benchmark-$i")")
done

responses=$(printf '%s\n' "${tokens[@]}" | xargs -P8 -I{} curl -sS -X POST "$base_url/order/v1/seckill/reserve" \
  -H 'Authorization: Bearer {}' -H 'Content-Type: application/json' \
  -d "{\"activityId\":$activity_id,\"liveStartTime\":$live_start,\"liveEndTime\":$live_end,\"livePeopleNum\":1}")
accepted=$(jq -s '[.[] | select(.code==200)] | length' <<<"$responses")
sold_out=$(jq -s '[.[] | select(.msg=="秒杀商品已售罄")] | length' <<<"$responses")
want_accepted=$((stock-1))
want_sold_out=$((8-want_accepted))
[[ "$accepted" == "$want_accepted" && "$sold_out" == "$want_sold_out" ]] || {
  printf 'accepted=%s sold_out=%s, want %s/%s\n' "$accepted" "$sold_out" "$want_accepted" "$want_sold_out" >&2
  exit 1
}

for _ in $(seq 1 30); do
  orders=$(mysql -N -e "SELECT COUNT(*) FROM looklook_order.seckill_order WHERE activity_id=$activity_id AND status=1")
  [[ "$orders" == "$stock" ]] && break
  sleep 1
done

read -r db_stock sold orders unique_users <<<"$(mysql -N -e "
SELECT a.stock,a.sold_count,COUNT(s.id),COUNT(DISTINCT s.user_id)
FROM looklook_order.seckill_activity a
LEFT JOIN looklook_order.seckill_order s ON s.activity_id=a.id
WHERE a.id=$activity_id GROUP BY a.id")"
redis_stock=$(redis GET "gin:looklook:{seckill}:v1:stock:$activity_id")
[[ "$db_stock" == "$stock" && "$sold" == "$stock" && "$orders" == "$stock" && "$unique_users" == "$stock" && "$redis_stock" == 0 ]] || {
  printf 'invariant failed: db_stock=%s sold=%s orders=%s users=%s redis=%s\n' "$db_stock" "$sold" "$orders" "$unique_users" "$redis_stock" >&2
  exit 1
}

jq -n \
  --argjson same_user_requests 10 \
  --argjson unique_same_reservations "$same_unique" \
  --argjson competing_requests 8 \
  --argjson accepted "$accepted" \
  --argjson sold_out "$sold_out" \
  --argjson final_orders "$orders" \
  '{result:"PASS",same_user_requests:$same_user_requests,unique_same_reservations:$unique_same_reservations,competing_requests:$competing_requests,accepted:$accepted,sold_out:$sold_out,final_orders:$final_orders,oversold:false}'
