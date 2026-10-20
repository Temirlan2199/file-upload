# file-upload

Пет-проект: файловое хранилище («mini-Dropbox») на Go.

Стек: Go (net/http), PostgreSQL (pgx), Redis, Kafka, MinIO (S3), Docker.

## Быстрый старт

```bash
cp .env.example .env   # локальные настройки
make infra-up          # поднять Postgres, Redis, Kafka, MinIO
make migrate-up        # применить миграции БД
make run               # запустить API на :8000
curl localhost:8000/health
```

## API

```bash
# регистрация
curl -X POST localhost:8000/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"me@mail.com","password":"supersecret"}'

# логин → token
curl -X POST localhost:8000/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"me@mail.com","password":"supersecret"}'

# текущий пользователь
curl localhost:8000/api/v1/users/me -H "Authorization: Bearer <token>"

# загрузить файл
curl -X POST localhost:8000/api/v1/files -H "Authorization: Bearer <token>" -F "file=@./README.md"

# список файлов
curl "localhost:8000/api/v1/files?limit=20&offset=0" -H "Authorization: Bearer <token>"

# метаданные / скачать / удалить
curl localhost:8000/api/v1/files/<id> -H "Authorization: Bearer <token>"
curl -OJ localhost:8000/api/v1/files/<id>/download -H "Authorization: Bearer <token>"
curl -X DELETE localhost:8000/api/v1/files/<id> -H "Authorization: Bearer <token>"
```

Все запросы также есть в `api.http` — его можно запускать прямо из GoLand.

## Локальные UI

| Сервис        | Адрес                 |
|---------------|-----------------------|
| API           | http://localhost:8000 |
| Kafka UI      | http://localhost:8080 |
| MinIO console | http://localhost:9001 |
