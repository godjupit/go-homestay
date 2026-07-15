#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/.."

base_url="${BASE_URL:-http://localhost:8080}"
mysql_password="${MYSQL_PASSWORD:-PXDN93VRKUm8TeE7}"
redis_password="${REDIS_PASSWORD:-G62m50oigInC30sf}"
activity_id=9001
stock=5

for command in curl jq docker; do
  command -v "$command" >/dev/null || { printf '%s is required\n' "$command" >&2; exit 1; }
done

printf '%s\n' '[1/6] Building and starting practice services'
docker compose --env-file config/.env.docker up -d --build api worker mysql redis kafka elasticsearch

worker_id=$(docker compose ps -q worker)
worker_process=$(docker inspect "$worker_id" --format '{{.Path}} {{join .Args " "}}')
if [[ "$worker_process" == api\ worker* ]]; then
  printf 'worker service still runs %q; finish practice-05 first\n' "$worker_process" >&2
  exit 1
fi

for _ in $(seq 1 30); do
  curl -fsS "$base_url/healthz" >/dev/null && break
  sleep 1
done
curl -fsS "$base_url/healthz" >/dev/null

printf '%s\n' '[2/6] Resetting isolated activity 9001'
docker compose exec -T mysql mysql -uroot -p"$mysql_password" -e "
DELETE h FROM looklook_order.homestay_order h
JOIN looklook_order.seckill_order s ON s.order_sn=h.sn
WHERE s.activity_id=$activity_id;
DELETE FROM looklook_order.seckill_order WHERE activity_id=$activity_id;
INSERT INTO looklook_order.seckill_activity(id,homestay_id,title,price,stock,sold_count,start_time,end_time,status)
VALUES($activity_id,11,'Practice Load Test',9900,$stock,0,DATE_SUB(NOW(),INTERVAL 1 HOUR),DATE_ADD(NOW(),INTERVAL 1 DAY),1)
ON DUPLICATE KEY UPDATE stock=$stock,sold_count=0,start_time=DATE_SUB(NOW(),INTERVAL 1 HOUR),end_time=DATE_ADD(NOW(),INTERVAL 1 DAY),status=1;
"

now=$(date +%s)
expire=$((now + 7*24*3600))
redis() { docker compose exec -T redis redis-cli -a "$redis_password" --no-auth-warning "$@"; }
redis DEL "gin:looklook:{seckill}:v1:activity:$activity_id" "gin:looklook:{seckill}:v1:stock:$activity_id" "gin:looklook:{seckill}:v1:users:$activity_id" >/dev/null
redis HSET "gin:looklook:{seckill}:v1:activity:$activity_id" startAt $((now-3600)) endAt $((now+86400)) status 1 >/dev/null
redis SET "gin:looklook:{seckill}:v1:stock:$activity_id" "$stock" EX $((7*24*3600)) >/dev/null
redis EXPIREAT "gin:looklook:{seckill}:v1:activity:$activity_id" "$expire" >/dev/null

printf '%s\n' '[3/6] Registering one idempotency user'
mobile=$(printf '133%08d' $((80000000 + now % 10000000)))
register=$(curl -sS -X POST "$base_url/usercenter/v1/user/register" -H 'Content-Type: application/json' -d "{\"mobile\":\"$mobile\",\"password\":\"123456\",\"nickname\":\"same-user\"}")
[[ "$(jq -r '.code // 0' <<<"$register")" -eq 200 ]] || { printf 'registration failed: %s\n' "$register" >&2; exit 1; }
login=$(curl -sS -X POST "$base_url/usercenter/v1/user/login" -H 'Content-Type: application/json' -d "{\"mobile\":\"$mobile\",\"password\":\"123456\"}")
token=$(jq -r '.data.accessToken // empty' <<<"$login")
[[ -n "$token" ]] || { printf 'login failed: %s\n' "$login" >&2; exit 1; }
live_start=$((now+172800))
live_end=$((live_start+172800))

same=$(seq 1 10 | xargs -P10 -I{} curl -sS -X POST "$base_url/order/v1/seckill/reserve" -H "Authorization: Bearer $token" -H 'Content-Type: application/json' -d "{\"activityId\":$activity_id,\"liveStartTime\":$live_start,\"liveEndTime\":$live_end,\"livePeopleNum\":1,\"remark\":\"practice-same\"}")
same_unique=$(jq -s '[.[].data.reservationSn] | unique | length' <<<"$same")
same_reservation=$(jq -sr '.[0].data.reservationSn // empty' <<<"$same")
[[ "$same_unique" -eq 1 ]] || { printf 'same user produced %s reservation numbers\n' "$same_unique" >&2; exit 1; }
[[ "$(redis GET "gin:looklook:{seckill}:v1:stock:$activity_id")" -eq 4 ]] || { printf 'same user deducted stock more than once\n' >&2; exit 1; }

unauthorized_status=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$base_url/order/v1/seckill/result" -H 'Content-Type: application/json' -d "{\"reservationSn\":\"$same_reservation\"}")
[[ "$unauthorized_status" -eq 401 ]] || { printf 'result endpoint accepted a request without token\n' >&2; exit 1; }

same_status=""
for _ in $(seq 1 30); do
  result=$(curl -sS -X POST "$base_url/order/v1/seckill/result" -H "Authorization: Bearer $token" -H 'Content-Type: application/json' -d "{\"reservationSn\":\"$same_reservation\"}")
  same_status=$(jq -r '.data.status // empty' <<<"$result")
  [[ "$same_status" == success || "$same_status" == failed ]] && break
  sleep 1
done
[[ "$same_status" == success ]] || { printf 'same-user reservation ended in status %s\n' "$same_status" >&2; exit 1; }

printf '%s\n' '[4/6] Registering 8 competing users'
declare -a pairs=()
for i in $(seq 1 8); do
  mobile=$(printf '132%08d' $((80000000 + now % 10000000 + i)))
  response=$(curl -sS -X POST "$base_url/usercenter/v1/user/register" -H 'Content-Type: application/json' -d "{\"mobile\":\"$mobile\",\"password\":\"123456\",\"nickname\":\"race-$i\"}")
  [[ "$(jq -r '.code // 0' <<<"$response")" -eq 200 ]] || { printf 'registration failed for user %d: %s\n' "$i" "$response" >&2; exit 1; }
  login=$(curl -sS -X POST "$base_url/usercenter/v1/user/login" -H 'Content-Type: application/json' -d "{\"mobile\":\"$mobile\",\"password\":\"123456\"}")
  token=$(jq -r '.data.accessToken // empty' <<<"$login")
  [[ -n "$token" ]] || { printf 'login failed for user %d: %s\n' "$i" "$login" >&2; exit 1; }
  pairs+=("$mobile|$token")
done

printf '%s\n' '[5/6] Racing for the remaining 4 units'
responses=$(printf '%s\n' "${pairs[@]}" | xargs -P8 -I{} bash -c '
  pair="$1"; token=${pair#*|}
  response=$(curl -sS -X POST "'$base_url'/order/v1/seckill/reserve" \
    -H "Authorization: Bearer $token" -H "Content-Type: application/json" \
    -d "{\"activityId\":'$activity_id',\"liveStartTime\":'$live_start',\"liveEndTime\":'$live_end',\"livePeopleNum\":1}")
  if [[ $(jq -r ".code // 0" <<<"$response") -ne 200 ]]; then
    printf "%s" "$response"
    exit 0
  fi
  reservation=$(jq -r ".data.reservationSn" <<<"$response")
  for _ in $(seq 1 30); do
    result=$(curl -sS -X POST "'$base_url'/order/v1/seckill/result" \
      -H "Authorization: Bearer $token" -H "Content-Type: application/json" \
      -d "{\"reservationSn\":\"$reservation\"}")
    status=$(jq -r ".data.status // empty" <<<"$result")
    if [[ "$status" == success || "$status" == failed ]]; then
      printf "%s" "$result"
      exit 0
    fi
    sleep 1
  done
  printf "%s" "$result"
' _ '{}')
accepted=$(jq -s '[.[] | select(.code==200)] | length' <<<"$responses")
sold_out=$(jq -s '[.[] | select(.msg=="秒杀商品已售罄")] | length' <<<"$responses")
[[ "$accepted" -eq 4 && "$sold_out" -eq 4 ]] || { printf 'accepted=%s sold_out=%s, want 4/4\n' "$accepted" "$sold_out" >&2; exit 1; }
[[ "$(jq -s '[.[] | select(.code==200 and .data.status=="success")] | length' <<<"$responses")" -eq 4 ]] || { printf 'not every accepted reservation reached success\n' >&2; exit 1; }

for _ in $(seq 1 30); do
  orders=$(docker compose exec -T mysql mysql -uroot -p"$mysql_password" -N -e "SELECT COUNT(*) FROM looklook_order.seckill_order WHERE activity_id=$activity_id AND status=1" 2>/dev/null)
  [[ "$orders" -eq 5 ]] && break
  sleep 1
done

printf '%s\n' '[6/6] Checking invariants'
read -r db_stock sold orders unique_users <<<"$(docker compose exec -T mysql mysql -uroot -p"$mysql_password" -N -e "SELECT a.stock,a.sold_count,COUNT(s.id),COUNT(DISTINCT s.user_id) FROM looklook_order.seckill_activity a LEFT JOIN looklook_order.seckill_order s ON s.activity_id=a.id WHERE a.id=$activity_id GROUP BY a.id" 2>/dev/null)"
redis_stock=$(redis GET "gin:looklook:{seckill}:v1:stock:$activity_id")
if [[ "$db_stock" -ne 5 || "$sold" -ne 5 || "$orders" -ne 5 || "$unique_users" -ne 5 || "$redis_stock" -ne 0 ]]; then
  printf 'invariant failed: db_stock=%s sold=%s orders=%s users=%s redis=%s\n' "$db_stock" "$sold" "$orders" "$unique_users" "$redis_stock" >&2
  exit 1
fi

jq -n --argjson same_user_requests 10 --argjson unique_same_reservations "$same_unique" --argjson competing_requests 8 --argjson accepted "$accepted" --argjson sold_out "$sold_out" --argjson orders "$orders" '{same_user_requests:$same_user_requests,unique_same_reservations:$unique_same_reservations,competing_requests:$competing_requests,accepted:$accepted,sold_out:$sold_out,final_orders:$orders,oversold:false}'
