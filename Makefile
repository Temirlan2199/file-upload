# Подтягиваем переменные из .env и экспортируем их в окружение команд (go run и т.д.).
-include .env
export

.PHONY: run build test lint tidy infra-up infra-down infra-logs

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

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
