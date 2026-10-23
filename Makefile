# Подтягиваем переменные из .env и экспортируем их в окружение команд (go run и т.д.).
-include .env
export

# golang-migrate — утилита для миграций. go run скачает и запустит нужную версию.
# -tags 'postgres' включает поддержку Postgres (без него утилита не знает про эту базу).
MIGRATE := go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.20.1

.PHONY: run worker build test infra-up infra-down migrate-up migrate-down migrate-version migrate-create

run:
	go run ./cmd/api

# Воркер Kafka — запускать в отдельном терминале параллельно с make run
worker:
	go run ./cmd/worker

build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

test:
	go test ./...

infra-up:
	docker compose up -d

infra-down:
	docker compose down

# Применить все новые миграции (*.up.sql)
migrate-up:
	$(MIGRATE) -path migrations -database "$(POSTGRES_DSN)" up

# Откатить одну последнюю миграцию (*.down.sql)
migrate-down:
	$(MIGRATE) -path migrations -database "$(POSTGRES_DSN)" down 1

# Показать текущую версию базы
migrate-version:
	$(MIGRATE) -path migrations -database "$(POSTGRES_DSN)" version

# Создать новую миграцию: make migrate-create name=create_files
# -seq — номера по порядку (000002, 000003...) вместо даты и времени.
migrate-create:
	$(MIGRATE) create -ext sql -dir migrations -seq $(name)
