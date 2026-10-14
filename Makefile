# Подтягиваем переменные из .env и экспортируем их в окружение команд (go run и т.д.).
-include .env
export

# goose — утилита для миграций. go run скачает и запустит нужную версию.
GOOSE := go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 -dir migrations

.PHONY: run build test infra-up infra-down migrate-up migrate-down migrate-status migrate-create

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

test:
	go test ./...

infra-up:
	docker compose up -d

infra-down:
	docker compose down

migrate-up:
	$(GOOSE) postgres "$(POSTGRES_DSN)" up

migrate-down:
	$(GOOSE) postgres "$(POSTGRES_DSN)" down

migrate-status:
	$(GOOSE) postgres "$(POSTGRES_DSN)" status

# Создать новую миграцию: make migrate-create name=create_files
migrate-create:
	$(GOOSE) create $(name) sql
