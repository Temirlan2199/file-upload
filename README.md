# file-upload

Пет-проект: файловое хранилище («mini-Dropbox») на Go.

Стек: Go, gin, PostgreSQL (pgx), Redis, Kafka, MinIO (S3), Docker.

## Быстрый старт

```bash
cp .env.example .env   # локальные настройки
make infra-up          # поднять Postgres, Redis, Kafka, MinIO
make run               # запустить API на :8000
curl localhost:8000/health
```

## Локальные UI

| Сервис        | Адрес                 |
|---------------|-----------------------|
| API           | http://localhost:8000 |
| Kafka UI      | http://localhost:8080 |
| MinIO console | http://localhost:9001 |
