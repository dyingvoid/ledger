-include .env
export

APP := ledger

.PHONY: run build test fmt vet lint lint-fix tidy \
	docker-up docker-down docker-logs \
	postgres-up postgres-down postgres-logs \
	migrate-up migrate-down migrate-action migrate-create

run:
	@go run ./cmd/$(APP)

build:
	@go build -o bin/$(APP) ./cmd/$(APP)

test:
	@go test -race ./...

fmt:
	@gofmt -s -w .

vet:
	@go vet ./...

lint:
	@docker compose run --rm lint

lint-fix:
	@docker compose run --rm lint run --fix ./...

tidy:
	@go mod tidy

docker-up:
	@docker compose up -d --build --wait

docker-down:
	@docker compose down

docker-logs:
	@docker compose logs -f ledger

postgres-up:
	@docker compose up -d --wait postgres

postgres-down:
	@docker compose down

postgres-logs:
	@docker compose logs -f postgres

migrate-up:
	@$(MAKE) migrate-action action=up

migrate-down:
	@$(MAKE) migrate-action action=down

migrate-action:
	@if [ -z "$(action)" ]; then \
		echo "action is not specified. Example: make migrate-action action=up"; \
		exit 1; \
	fi
	@docker compose run --rm migrate \
		-path /migrations \
		-database "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable" \
		"$(action)"

migrate-create:
	@if [ -z "$(seq)" ]; then \
		echo "seq parameter is not specified. Example: make migrate-create seq=init"; \
		exit 1; \
	fi
	@docker compose run --rm migrate \
		create \
		-ext sql \
		-dir /migrations \
		-seq "$(seq)"
