-- +goose Up
CREATE TABLE users (
    -- gen_random_uuid() встроена в Postgres начиная с 13 версии.
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- UNIQUE создаёт индекс: поиск по email при логине быстрый,
    -- а дубликаты отсекает сама база (ошибка 23505 unique_violation).
    email         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE users;
