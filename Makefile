.PHONY: dev docker down logs test clean

dev:
	@echo "Starting Go local development..."
	set -a && . config/.env.local && set +a && go run ./cmd/server

docker:
	@echo "Starting docker environment..."
	docker compose --env-file config/.env.docker up -d --build

down:
	docker compose down

logs:
	docker compose logs -f app

test:
	go test -race ./...

clean:
	docker system prune -f
