CREATE TABLE files (
    -- id генерируем в Go (uuid.New()), чтобы знать его ещё до вставки:
    -- он же будет ключом объекта в MinIO.
    id           UUID PRIMARY KEY,
    -- REFERENCES — внешний ключ: файл не может ссылаться на несуществующего пользователя.
    -- ON DELETE CASCADE — при удалении пользователя его файлы удалятся из таблицы автоматически.
    owner_id     UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name         TEXT        NOT NULL,
    size         BIGINT      NOT NULL,
    content_type TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Индекс под запрос «файлы пользователя, новые сверху» (список файлов).
CREATE INDEX files_owner_id_created_at_idx ON files (owner_id, created_at DESC);
