#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/.."

container="gin-looklook-migration-check-$$"
password="migration-check-password"

cleanup() {
  docker rm -f "$container" >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker run -d --name "$container" \
  -e MYSQL_ROOT_PASSWORD="$password" \
  -e MYSQL_ROOT_HOST='%' \
  mysql/mysql-server:8.0.28 \
  --default-authentication-plugin=mysql_native_password \
  --character-set-server=utf8mb4 \
  --collation-server=utf8mb4_general_ci \
  --lower_case_table_names=1 >/dev/null

for _ in $(seq 1 60); do
  docker exec "$container" mysql -uroot -p"$password" -N -e 'SELECT 1' >/dev/null 2>&1 && break
  sleep 1
done
docker exec "$container" mysql -uroot -p"$password" -N -e 'SELECT 1' >/dev/null

for migration in migrations/*.sql; do
  printf 'applying %s\n' "$migration"
  docker exec -i "$container" mysql -uroot -p"$password" <"$migration"
done

table_count=$(docker exec "$container" mysql -uroot -p"$password" -N -e \
  "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema IN ('looklook_usercenter','looklook_travel','looklook_order','looklook_payment')")
[[ "$table_count" -ge 15 ]] || { printf 'only %s application tables were created\n' "$table_count" >&2; exit 1; }

index_count=$(docker exec "$container" mysql -uroot -p"$password" -N -e \
  "SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema='looklook_payment' AND table_name='third_payment' AND index_name='uk_order_service'")
[[ "$index_count" -ge 1 ]] || { printf 'payment idempotency index is missing\n' >&2; exit 1; }

printf 'migration validation passed with %s application tables\n' "$table_count"
