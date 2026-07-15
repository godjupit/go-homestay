#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/.."

scenario="${1:-all}"
base_url="${BASE_URL:-http://localhost:8080}"
mysql_password="${MYSQL_PASSWORD:-PXDN93VRKUm8TeE7}"
admin_user="${ADMIN_INITIAL_USER:-admin}"
admin_password="${ADMIN_INITIAL_PASSWORD:-Admin@123}"
last_body=""
last_status=""
test_mobile=""
test_token=""

fail() {
  printf 'E2E FAILED: %s\n' "$*" >&2
  if [[ -n "$last_body" ]]; then
    printf 'last HTTP status: %s\nlast response: %s\n' "$last_status" "$last_body" >&2
  fi
  exit 1
}

require_tools() {
  for command in curl jq docker; do
    command -v "$command" >/dev/null || fail "$command is required"
  done
}

start_stack() {
  if [[ "${E2E_STACK_READY:-0}" != "1" ]]; then
    printf '%s\n' '[setup] building and starting API dependencies'
    docker compose --env-file config/.env.docker up -d --build api worker mysql redis kafka elasticsearch
  fi
  for _ in $(seq 1 60); do
    if curl -fsS "$base_url/healthz" >/dev/null 2>&1; then
      return
    fi
    sleep 1
  done
  fail "API did not become healthy"
}

request() {
  local path="$1"
  local body="$2"
  local token="${3:-}"
  local raw
  local -a args=(-sS -X POST "$base_url$path" -H 'Content-Type: application/json' -d "$body" -w $'\n%{http_code}')
  if [[ -n "$token" ]]; then
    args+=(-H "Authorization: Bearer $token")
  fi
  raw=$(curl "${args[@]}")
  last_status="${raw##*$'\n'}"
  last_body="${raw%$'\n'*}"
}

assert_http() {
  [[ "$last_status" == "$1" ]] || fail "HTTP status $last_status, want $1"
}

assert_code() {
  local got
  got=$(jq -r '.code // empty' <<<"$last_body")
  [[ "$got" == "$1" ]] || fail "business code $got, want $1"
}

assert_jq() {
  local expression="$1"
  local message="$2"
  jq -e "$expression" <<<"$last_body" >/dev/null || fail "$message"
}

mysql_value() {
  docker compose exec -T mysql mysql -uroot -p"$mysql_password" -N -e "$1" 2>/dev/null
}

new_mobile() {
  local prefix="$1"
  local salt="$2"
  local suffix=$(( (10#$(date +%s) + $$ * 97 + salt * 7919) % 100000000 ))
  printf '%s%08d' "$prefix" "$suffix"
}

register_and_login() {
  local prefix="$1"
  local salt="$2"
  local nickname="$3"
  test_mobile=$(new_mobile "$prefix" "$salt")

  request '/usercenter/v1/user/register' "{\"mobile\":\"$test_mobile\",\"password\":\"Practice@123\",\"nickname\":\"$nickname\"}"
  assert_http 200
  assert_code 200
  assert_jq '.data.accessToken | type == "string" and length > 20' 'registration did not issue a token'

  # Do not reuse the registration response: explicitly exercise the login endpoint to obtain the token.
  request '/usercenter/v1/user/login' "{\"mobile\":\"$test_mobile\",\"password\":\"Practice@123\"}"
  assert_http 200
  assert_code 200
  test_token=$(jq -r '.data.accessToken // empty' <<<"$last_body")
  [[ "$test_token" == *.*.* ]] || fail 'login token is not a three-part JWT'

  request '/usercenter/v1/user/detail' '{}' "$test_token"
  assert_http 200
  assert_code 200
  [[ "$(jq -r '.data.userInfo.mobile // empty' <<<"$last_body")" == "$test_mobile" ]] || fail 'login token belongs to the wrong user'
  [[ "$(mysql_value "SELECT COUNT(*) FROM looklook_usercenter.user WHERE mobile='$test_mobile' AND del_state=0")" == 1 ]] || fail 'registered user was not persisted exactly once'
}

create_order() {
  local token="$1"
  local remark="$2"
  local start=$(( $(date +%s) + 3*24*3600 ))
  local end=$(( start + 2*24*3600 ))
  request '/order/v1/homestayOrder/createHomestayOrder' "{\"homestayId\":11,\"isFood\":true,\"liveStartTime\":$start,\"liveEndTime\":$end,\"livePeopleNum\":3,\"remark\":\"$remark\"}" "$token"
  assert_http 200
  assert_code 200
  local sn
  sn=$(jq -r '.data.orderSn // empty' <<<"$last_body")
  [[ "$sn" == HSO* && ${#sn} -eq 25 ]] || fail "invalid order SN $sn"
  printf '%s' "$sn"
}

verify_auth() {
  printf '%s\n' '[auth] registering a fictional user and explicitly logging in'
  register_and_login 131 1 'e2e-auth'
  local mobile="$test_mobile"
  local token="$test_token"

  request '/usercenter/v1/user/detail' '{}' "$token"
  assert_http 200
  assert_code 200
  [[ "$(jq -r '.data.userInfo.mobile' <<<"$last_body")" == "$mobile" ]] || fail 'JWT resolved to the wrong user'

  request '/usercenter/v1/user/detail' '{}'
  assert_http 401
  assert_code 100003

  request '/usercenter/v1/user/detail' '{}' 'forged.token.value'
  assert_http 401
  assert_code 100003

  request '/usercenter/v1/user/login' "{\"mobile\":\"$mobile\",\"password\":\"wrong-password\"}"
  [[ "$last_status" != 200 ]] || fail 'wrong password was accepted'

  local missing_mobile
  missing_mobile=$(new_mobile 130 99)
  request '/usercenter/v1/user/login' "{\"mobile\":\"$missing_mobile\",\"password\":\"Practice@123\"}"
  [[ "$last_status" != 200 ]] || fail 'nonexistent user was accepted'

  request '/usercenter/v1/user/register' "{\"mobile\":\"$mobile\",\"password\":\"Practice@123\",\"nickname\":\"duplicate\"}"
  [[ "$last_status" != 200 ]] || fail 'duplicate registration was accepted'

  request '/usercenter/v1/user/profile' '{"nickname":"e2e-updated","sex":1,"info":"practice profile"}' "$token"
  assert_http 200
  assert_code 200
  request '/usercenter/v1/user/detail' '{}' "$token"
  assert_jq '.data.userInfo.nickname == "e2e-updated" and .data.userInfo.sex == 1' 'profile update was not persisted'
  printf '%s\n' '[auth] PASS'
}

verify_order() {
  printf '%s\n' '[order] creating two users and obtaining both tokens'
  register_and_login 129 2 'e2e-owner'
  local owner_mobile="$test_mobile"
  local owner_token="$test_token"
  register_and_login 128 3 'e2e-stranger'
  local stranger_token="$test_token"

  local sn
  sn=$(create_order "$owner_token" "practice-order-$$")

  request '/order/v1/homestayOrder/createHomestayOrder' '{"homestayId":11,"isFood":false,"liveStartTime":1,"liveEndTime":2,"livePeopleNum":1}'
  assert_http 401

  local invalid_start=$(( $(date +%s) + 4*24*3600 ))
  request '/order/v1/homestayOrder/createHomestayOrder' "{\"homestayId\":11,\"isFood\":false,\"liveStartTime\":$invalid_start,\"liveEndTime\":$invalid_start,\"livePeopleNum\":1}" "$owner_token"
  [[ "$last_status" != 200 ]] || fail 'zero-night order was accepted'

  request '/order/v1/homestayOrder/userHomestayOrderDetail' "{\"sn\":\"$sn\"}" "$owner_token"
  assert_http 200
  assert_code 200
  assert_jq '.data.tradeState == 0 and .data.homestayTotalPrice == 598 and .data.foodTotalPrice == 180 and .data.orderTotalPrice == 778' 'order snapshot or pricing is wrong'

  request '/order/v1/homestayOrder/userHomestayOrderList' '{"lastId":0,"pageSize":20,"tradeState":0}' "$owner_token"
  assert_http 200
  jq -e --arg sn "$sn" '.data.list | any(.sn == $sn)' <<<"$last_body" >/dev/null || fail 'new order missing from owner list'

  request '/order/v1/homestayOrder/userHomestayOrderDetail' "{\"sn\":\"$sn\"}" "$stranger_token"
  [[ "$last_status" != 200 ]] || fail 'another user could read the order'
  [[ "$(jq -r '.msg // empty' <<<"$last_body")" == 'order no exists' ]] || fail 'ownership failure leaks order existence'

  request '/order/v1/homestayOrder/userHomestayOrderCancel' "{\"sn\":\"$sn\"}" "$stranger_token"
  [[ "$last_status" != 200 ]] || fail 'another user could cancel the order'
  [[ "$(mysql_value "SELECT trade_state FROM looklook_order.homestay_order WHERE sn='$sn'")" == 0 ]] || fail 'unauthorized cancel changed the order'

  request '/order/v1/homestayOrder/userHomestayOrderCancel' "{\"sn\":\"$sn\"}" "$owner_token"
  assert_http 200
  assert_jq '.data.tradeState == -1' 'cancel did not return canceled state'

  request '/order/v1/homestayOrder/userHomestayOrderCancel' "{\"sn\":\"$sn\"}" "$owner_token"
  assert_http 200
  assert_jq '.data.tradeState == -1' 'repeated cancel is not idempotent'

  read -r state version db_mobile <<<"$(mysql_value "SELECT o.trade_state,o.version,u.mobile FROM looklook_order.homestay_order o JOIN looklook_usercenter.user u ON u.id=o.user_id WHERE o.sn='$sn'")"
  [[ "$state" == -1 && "$version" == 1 && "$db_mobile" == "$owner_mobile" ]] || fail "DB invariant failed: state=$state version=$version owner=$db_mobile"
  printf '%s\n' '[order] PASS'
}

verify_travel() {
  printf '%s\n' '[travel] registering/login first, then checking the public homestay read model'
  register_and_login 123 8 'e2e-travel'
  local token="$test_token"

  request '/travel/v1/homestay/homestayDetail' '{"id":11}' "$token"
  assert_http 200
  assert_code 200
  assert_jq '.data.homestay | .id == 11 and .title == "Interview Demo Homestay" and .city == "杭州" and .homestayPrice == 299 and .foodPrice == 30 and .peopleNum == 4' 'homestay detail does not match the seed read model'

  request '/travel/v1/homestay/homestayList' '{"page":1,"pageSize":20}' "$token"
  assert_http 200
  assert_code 200
  assert_jq '.data.list | any(.id == 11 and .homestayPrice == 299)' 'homestay list and detail are inconsistent'

  request '/travel/v1/homestay/businessList' '{"homestayBusinessId":1,"lastId":0,"pageSize":20}' "$token"
  assert_http 200
  assert_code 200
  assert_jq '.data.list | any(.id == 11)' 'business homestay cursor list missed seed homestay'

  request '/travel/v1/homestay/homestayDetail' '{}' "$token"
  [[ "$last_status" != 200 ]] || fail 'missing homestay id was accepted'
  assert_code 100002

  request '/travel/v1/homestay/homestayDetail' '{"id":999999999}' "$token"
  [[ "$last_status" != 200 ]] || fail 'nonexistent homestay returned success'
  printf '%s\n' '[travel] PASS'
}

verify_payment() {
  printf '%s\n' '[payment] creating owner/stranger users and a real pending order'
  register_and_login 127 4 'e2e-pay-owner'
  local owner_token="$test_token"
  register_and_login 126 5 'e2e-pay-stranger'
  local stranger_token="$test_token"
  local sn
  sn=$(create_order "$owner_token" "practice-payment-$$")

  request '/payment/v1/thirdPayment/thirdPaymentWxPay' "{\"orderSn\":\"$sn\",\"serviceType\":\"homestayOrder\"}"
  assert_http 401

  request '/payment/v1/thirdPayment/thirdPaymentWxPay' "{\"orderSn\":\"$sn\",\"serviceType\":\"homestayOrder\"}" "$stranger_token"
  [[ "$last_status" != 200 ]] || fail 'another user could initiate payment'
  [[ "$(mysql_value "SELECT COUNT(*) FROM looklook_payment.third_payment WHERE order_sn='$sn'")" == 0 ]] || fail 'unauthorized payment created a payment record'

  request '/payment/v1/thirdPayment/thirdPaymentWxPay' "{\"orderSn\":\"$sn\",\"serviceType\":\"unsupported\"}" "$owner_token"
  [[ "$last_status" != 200 ]] || fail 'unsupported payment service type was accepted'
  [[ "$(mysql_value "SELECT COUNT(*) FROM looklook_payment.third_payment WHERE order_sn='$sn'")" == 0 ]] || fail 'unsupported payment type created a payment record'

  request '/payment/v1/thirdPayment/thirdPaymentWxPay' "{\"orderSn\":\"$sn\",\"serviceType\":\"homestayOrder\"}" "$owner_token"
  [[ "$last_status" != 200 ]] || fail 'payment unexpectedly succeeded without WeChat authorization/configuration'
  [[ "$(mysql_value "SELECT COUNT(*) FROM looklook_payment.third_payment WHERE order_sn='$sn'")" == 0 ]] || fail 'failed prepay created a payment record'

  request '/order/v1/homestayOrder/userHomestayOrderCancel' "{\"sn\":\"$sn\"}" "$owner_token"
  assert_http 200
  request '/payment/v1/thirdPayment/thirdPaymentWxPay' "{\"orderSn\":\"$sn\",\"serviceType\":\"homestayOrder\"}" "$owner_token"
  [[ "$last_status" != 200 ]] || fail 'canceled order was allowed to pay'
  [[ "$(mysql_value "SELECT COUNT(*) FROM looklook_payment.third_payment WHERE order_sn='$sn'")" == 0 ]] || fail 'canceled order created a payment record'
  printf '%s\n' '[payment] PASS'
}

verify_search() {
  printf '%s\n' '[search] obtaining a user token before exercising search'
  register_and_login 125 6 'e2e-search'
  local token="$test_token"

  request '/travel/v1/search/homestays' '{"minPrice":500,"maxPrice":100,"page":1,"pageSize":10}' "$token"
  [[ "$last_status" != 200 ]] || fail 'invalid price range was accepted'
  assert_code 100002

  local found=0
  for _ in $(seq 1 30); do
    request '/travel/v1/search/homestays' '{"city":"杭州","tags":["西湖"],"sortBy":["price_asc"],"page":1,"pageSize":10}' "$token"
    if [[ "$last_status" == 200 ]] && jq -e '.data.list | any(.id == 11)' <<<"$last_body" >/dev/null; then
      found=1
      break
    fi
    sleep 1
  done
  [[ "$found" == 1 ]] || fail 'seed homestay 11 was not searchable'
  assert_jq '.data.total >= 1 and (.data.list | all(.city == "杭州"))' 'search filters or total are inconsistent'
  printf '%s\n' '[search] PASS'
}

verify_admin() {
  printf '%s\n' '[admin] obtaining a normal token and an independent admin token'
  register_and_login 124 7 'e2e-normal-user'
  local normal_token="$test_token"

  request '/admin/v1/auth/login' "{\"username\":\"$admin_user\",\"password\":\"wrong-password\"}"
  [[ "$last_status" != 200 ]] || fail 'wrong admin password was accepted'

  request '/admin/v1/auth/login' "{\"username\":\"$admin_user\",\"password\":\"$admin_password\"}"
  assert_http 200
  assert_code 200
  local admin_token
  admin_token=$(jq -r '.data.accessToken // empty' <<<"$last_body")
  [[ "$admin_token" == *.*.* ]] || fail 'admin login did not return a JWT'

  request '/admin/v1/user/list' '{"page":1,"pageSize":10}'
  assert_http 401
  request '/admin/v1/user/list' '{"page":1,"pageSize":10}' "$normal_token"
  assert_http 401
  request '/admin/v1/user/list' '{"page":1,"pageSize":10}' "$admin_token"
  assert_http 200
  assert_code 200
  assert_jq '.data.total >= 1' 'admin list is unexpectedly empty'

  request '/admin/v1/homestay/list' '{"page":1,"pageSize":20}' "$admin_token"
  assert_http 200
  assert_code 200
  assert_jq '.data.list | any(.id == 11)' 'super admin data scope did not include seed homestay'
  printf '%s\n' '[admin] PASS'
}

verify_metrics() {
  printf '%s\n' '[metrics] creating and canceling an order before inspecting exported metrics'
  register_and_login 122 9 'e2e-metrics'
  local token="$test_token"
  local sn
  sn=$(create_order "$token" "practice-metrics-$$")
  request '/order/v1/homestayOrder/userHomestayOrderCancel' "{\"sn\":\"$sn\"}" "$token"
  assert_http 200
  assert_jq '.data.tradeState == -1' 'metric setup cancellation failed'

  local metrics
  metrics=$(curl -fsS "$base_url/metrics") || fail '/metrics is unavailable'
  grep -q '^gin_looklook_http_requests_total{' <<<"$metrics" || fail 'HTTP request counter is missing'
  grep -q '^gin_looklook_order_transitions_total{from="0",result="success",to="-1"} ' <<<"$metrics" || fail 'successful pending-to-canceled transition was not observed'
  grep -q '^gin_looklook_payment_outbox_pending ' <<<"$metrics" || fail 'payment outbox pending gauge is missing'
  grep -q '^gin_looklook_payment_outbox_oldest_age_seconds ' <<<"$metrics" || fail 'payment outbox age gauge is missing'
  printf '%s\n' '[metrics] PASS'
}

require_tools
start_stack

case "$scenario" in
  auth) verify_auth ;;
  travel) verify_travel ;;
  order) verify_order ;;
  payment) verify_payment ;;
  search) verify_search ;;
  admin) verify_admin ;;
  metrics) verify_metrics ;;
  all)
    verify_auth
    verify_travel
    verify_order
    verify_payment
    verify_search
    verify_admin
    verify_metrics
    ;;
  *) fail "unknown scenario $scenario (use auth|travel|order|payment|search|admin|metrics|all)" ;;
esac

printf 'E2E scenario %s completed successfully.\n' "$scenario"
