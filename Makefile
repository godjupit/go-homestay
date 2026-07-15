.PHONY: dev worker docker down logs fmt fmt-check vet test build compose-check migration-check verify demo benchmark-seckill clean

COMPOSE_ENV_FILES := --env-file config/.env.docker
ifneq ($(wildcard .env),)
COMPOSE_ENV_FILES += --env-file .env
endif

dev:
	@echo "Starting Go local development..."
	set -a; . config/.env.local; [ ! -f .env ] || . ./.env; set +a; go run ./cmd/api

worker:
	@echo "Starting worker..."
	set -a && . config/.env.local && set +a && go run ./cmd/worker

docker:
	@echo "Starting docker environment..."
	docker compose $(COMPOSE_ENV_FILES) up -d --build

down:
	docker compose down

logs:
	docker compose logs -f api worker

test:
	go test -race ./...

fmt:
	gofmt -w $$(find cmd internal -name '*.go' -type f)

fmt-check:
	@test -z "$$(gofmt -l $$(find cmd internal -name '*.go' -type f))" || \
		{ echo 'Go files need formatting. Run: make fmt'; gofmt -l $$(find cmd internal -name '*.go' -type f); exit 1; }

vet:
	go vet ./...

build:
	go build ./cmd/api ./cmd/worker

compose-check:
	docker compose $(COMPOSE_ENV_FILES) config --quiet

migration-check:
	./scripts/validate-migrations.sh

verify: fmt-check vet test build compose-check

demo:
	./scripts/demo.sh

benchmark-seckill:
	./scripts/benchmark-seckill.sh

clean:
	docker system prune -f
