-- Статус обработки файла воркером:
--   pending — загружен, ждёт обработки;
--   ready   — воркер обработал (посчитал checksum);
--   failed  — обработка не удалась.
-- CHECK не даст записать в колонку другое значение.
ALTER TABLE files
    ADD COLUMN status   TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'ready', 'failed')),
    -- SHA-256 содержимого файла в hex. Пустая строка — ещё не посчитан.
    ADD COLUMN checksum TEXT NOT NULL DEFAULT '';

-- Файлы, загруженные до этой миграции, считаем уже обработанными,
-- иначе они навсегда остались бы в pending (событий для них в Kafka нет).
UPDATE files SET status = 'ready';
