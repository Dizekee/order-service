BINARY_NAME=order
GO=go
GOFLAGS=-v
MAIN_PATH=./cmd/order-service
MIGRATE=migrate
MIGRATIONS_PATH=./migrations
DB_URL=postgres://postgres:postgres@localhost:5432/orders?sslmode=disable

GREEN=\033[0;32m
YELLOW=\033[0;33m
RESET=\033[0m

.PHONY: help run build test lint clean migrate-up migrate-down docker-up docker-down deps coverage

help:
	@echo "${GREEN}Доступные команды:${RESET}"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "${YELLOW}%-20s${RESET} %s\n", $$1, $$2}'

run:
	$(GO) run $(MAIN_PATH)

build:
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)

test:
	$(GO) test -race -cover ./...

test-race:
	$(GO) test -race -run TestRaceCondition ./...

test-timeout:
	$(GO) test -race -run TestSupplierTimeout ./...

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out

migrate-up:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DB_URL)" up

migrate-down:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DB_URL)" down

migrate-force:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DB_URL)" force $(VERSION)

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f app

docker-build:
	docker-compose build

deps:
	$(GO) mod download
	$(GO) mod tidy

coverage:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out
	
.PHONY: test-integration test-race test-idempotency test-all

test-integration:
	$(GO) test -race -v ./test/integration/...

test-race:
	$(GO) test -race -v -run TestRaceCondition ./test/integration/...

test-idempotency:
	$(GO) test -race -v -run TestIdempotency ./test/integration/...

test-webhook-before:
	$(GO) test -race -v -run TestWebhookBeforeOrder ./test/integration/...

test-all:
	$(GO) test -race -v ./...

supplier-build:
	docker-compose build supplier-a supplier-b

supplier-up:
	docker-compose up -d supplier-a supplier-b

supplier-down:
	docker-compose stop supplier-a supplier-b

supplier-logs:
	docker-compose logs -f supplier-a supplier-b