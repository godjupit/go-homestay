#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/.."

base_url="${BASE_URL:-http://localhost:8080}"
last_body=""
last_status=""

fail() {
  printf 'DEMO FAILED: %s\n' "$*" >&2
  [[ -z "$last_body" ]] || printf 'HTTP %s: %s\n' "$last_status" "$last_body" >&2
  exit 1
}

for tool in curl jq; do
  command -v "$tool" >/dev/null || fail "$tool is required"
done

if [[ "${DEMO_STACK_READY:-0}" != "1" ]]; then
  command -v docker >/dev/null || fail 'docker is required to start the demo stack'
  printf '%s\n' '[setup] starting Docker Compose'
  docker compose --env-file config/.env.docker up -d --build api worker mysql redis kafka elasticsearch
fi

for _ in $(seq 1 60); do
  curl -fsS "$base_url/healthz" >/dev/null 2>&1 && break
  sleep 1
done
curl -fsS "$base_url/healthz" >/dev/null || fail 'API did not become healthy'

request() {
  local path="$1"
  local body="$2"
  local token="${3:-}"
  local raw
  local -a args=(-sS -X POST "$base_url$path" -H 'Content-Type: application/json' -d "$body" -w $'\n%{http_code}')
  [[ -z "$token" ]] || args+=(-H "Authorization: Bearer $token")
  raw=$(curl "${args[@]}")
  last_status="${raw##*$'\n'}"
  last_body="${raw%$'\n'*}"
}

expect_ok() {
  [[ "$last_status" == 200 ]] || fail "expected HTTP 200, got $last_status"
  [[ "$(jq -r '.code // empty' <<<"$last_body")" == 200 ]] || fail 'business request failed'
}

mobile=$(printf '135%08d' "$(( (10#$(date +%s) + $$ * 97) % 100000000 ))")
password='Demo@123456'

printf '%s\n' '[1/6] registering a fictional user'
request '/usercenter/v1/user/register' "{\"mobile\":\"$mobile\",\"password\":\"$password\",\"nickname\":\"repo-demo\"}"
expect_ok

printf '%s\n' '[2/6] explicitly logging in to obtain a JWT'
request '/usercenter/v1/user/login' "{\"mobile\":\"$mobile\",\"password\":\"$password\"}"
expect_ok
token=$(jq -r '.data.accessToken // empty' <<<"$last_body")
[[ "$token" == *.*.* ]] || fail 'login did not return a JWT'

printf '%s\n' '[3/6] resolving the current user from the JWT'
request '/usercenter/v1/user/detail' '{}' "$token"
expect_ok
[[ "$(jq -r '.data.userInfo.mobile' <<<"$last_body")" == "$mobile" ]] || fail 'JWT resolved to another user'

printf '%s\n' '[4/6] reading the seed homestay'
request '/travel/v1/homestay/homestayDetail' '{"id":11}' "$token"
expect_ok
jq -e '.data.homestay.id == 11 and .data.homestay.homestayPrice == 299' <<<"$last_body" >/dev/null || fail 'seed homestay is inconsistent'

printf '%s\n' '[5/6] creating a two-night order'
start=$(( $(date +%s) + 3*24*3600 ))
end=$(( start + 2*24*3600 ))
request '/order/v1/homestayOrder/createHomestayOrder' "{\"homestayId\":11,\"isFood\":true,\"liveStartTime\":$start,\"liveEndTime\":$end,\"livePeopleNum\":2,\"remark\":\"repository-demo\"}" "$token"
expect_ok
sn=$(jq -r '.data.orderSn // empty' <<<"$last_body")
[[ "$sn" == HSO* ]] || fail 'order number is invalid'

printf '%s\n' '[6/6] canceling the order and checking the final state'
request '/order/v1/homestayOrder/userHomestayOrderCancel' "{\"sn\":\"$sn\"}" "$token"
expect_ok
jq -e '.data.tradeState == -1' <<<"$last_body" >/dev/null || fail 'order was not canceled'

jq -n --arg mobile "$mobile" --arg orderSn "$sn" '{result:"PASS",user:$mobile,orderSn:$orderSn,finalTradeState:-1}'
