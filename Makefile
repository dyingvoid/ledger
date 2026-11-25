-include .env
export

SHARED_DIR := shared
ACCOUNTING_DIR := services/accounting
BFF_DIR := services/bff
GO_PACKAGES := ./$(SHARED_DIR)/... ./$(ACCOUNTING_DIR)/... ./$(BFF_DIR)/...
MIGRATE_LOCK_TIMEOUT ?= 5
RATE ?= 6
DURATION ?= 1m

.PHONY: run run-accounting run-bff build build-accounting build-bff \
	test fmt vet lint lint-fix tidy seed \
	docker-up docker-down docker-logs \
	postgres-up postgres-down postgres-logs \
	tigerbeetle-up tigerbeetle-down tigerbeetle-logs \
	temporal-up temporal-down temporal-logs \
	migrate-up migrate-down migrate-action migrate-create \
	proto-gen proto-lint \
	loadtest-accounts loadtest-deposit loadtest-matrix

run: run-accounting

run-accounting:
	@go run ./$(ACCOUNTING_DIR)/cmd/ledger

run-bff:
	@go run ./$(BFF_DIR)/cmd/bff

build: build-accounting build-bff

build-accounting:
	@mkdir -p bin
	@go build -o bin/accounting ./$(ACCOUNTING_DIR)/cmd/ledger

build-bff:
	@mkdir -p bin
	@go build -o bin/bff ./$(BFF_DIR)/cmd/bff

seed:
	@go run ./$(ACCOUNTING_DIR)/cmd/seed -addr localhost:$(or $(GRPC_PORT),50051)

test:
	@go test -race $(GO_PACKAGES)

fmt:
	@gofmt -s -w shared services

vet:
	@go vet $(GO_PACKAGES)

lint:
	@docker compose run --rm lint

lint-fix:
	@docker compose run --rm lint run --fix ./services/accounting/... ./services/bff/...

tidy:
	@go -C $(SHARED_DIR) mod tidy
	@go -C $(ACCOUNTING_DIR) mod tidy
	@go -C $(BFF_DIR) mod tidy
	@go work sync

docker-up:
	@docker compose up -d --build --wait

docker-down:
	@docker compose down

docker-logs:
	@docker compose logs -f accounting bff temporal

postgres-up:
	@docker compose up -d --wait postgres

postgres-down:
	@docker compose down

postgres-logs:
	@docker compose logs -f postgres

tigerbeetle-up:
	@docker compose up -d --wait tigerbeetle

tigerbeetle-down:
	@docker compose down

tigerbeetle-logs:
	@docker compose logs -f tigerbeetle

temporal-up:
	@docker compose up -d --wait temporal

temporal-down:
	@docker compose down

temporal-logs:
	@docker compose logs -f temporal

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
		-lock-timeout $(MIGRATE_LOCK_TIMEOUT) \
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

proto-gen:
	@cd $(SHARED_DIR)/proto && buf generate

proto-lint:
	@cd $(SHARED_DIR)/proto && buf lint

loadtest-accounts:
	@set -a; . ./.env; [ -f loadtest/seed.env ] && . ./loadtest/seed.env; set +a; \
		k6 run -e DURATION=$(DURATION) -e VUS=$${VUS:-2000} \
			-e RUN_ID=$${RUN_ID:-$$(date +%Y%m%d-%H%M%S)} \
			-e ACCOUNT_MIN_ID=$${ACCOUNT_MIN_ID:-1} \
			-e CURRENCY=$${SEED_CURRENCY:-USD} \
			loadtest/accounts.js

loadtest-deposit:
	@set -a; . ./.env; [ -f loadtest/seed.env ] && . ./loadtest/seed.env; set +a; \
		k6 run -e MODE=$${MODE:-deadline} -e RATE=$(RATE) -e DURATION=$(DURATION) \
			-e VUS=$${VUS:-2000} \
			-e RUN_ID=$${RUN_ID:-$$(date +%Y%m%d-%H%M%S)} \
			-e ACCOUNT_MIN_ID=$${ACCOUNT_MIN_ID:-1} \
			-e ACCOUNT_COUNT=$${ACCOUNT_COUNT:-15000000} \
			-e CURRENCY=$${SEED_CURRENCY:-USD} \
			loadtest/deposit.js

loadtest-matrix:
	@./loadtest/run-matrix.sh
