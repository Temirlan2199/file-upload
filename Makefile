# Подтягиваем переменные из .env и экспортируем их в окружение команд (go run и т.д.).
-include .env
export

GOOSE_VERSION := v3.26.0

.PHONY: run build test lint tidy infra-up infra-down infra-logs \
	migrate-up migrate-down migrate-status migrate-create

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api
	go build -o bin/migrator ./cmd/migrator

test:
	go test -race ./...

lint:
	golangci-lint run

tidy:
	go mod tidy

infra-up:
	docker compose up -d

infra-down:
	docker compose down

infra-logs:
	docker compose logs -f

# --- Миграции ---
migrate-up:
	go run ./cmd/migrator up

migrate-down:
	go run ./cmd/migrator down

migrate-status:
	go run ./cmd/migrator status

# Создать новую миграцию: make migrate-create name=create_files
migrate-create:
	go run github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION) -dir migrations create $(name) sql
