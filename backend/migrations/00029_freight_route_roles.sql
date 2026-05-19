-- +goose NO TRANSACTION
--
-- NO TRANSACTION нужен по двум независимым причинам:
--   1. CREATE INDEX CONCURRENTLY запрещён внутри транзакции — мы строим
--      GIN-индексы CONCURRENTLY, чтобы не брать ACCESS EXCLUSIVE на
--      freight_requests_lookup (горячий read-эндпоинт);
--   2. Батч-backfill использует COMMIT внутри DO-блока, чтобы реально
--      освобождать row-locks между батчами и давать autovacuum / другим
--      транзакциям окно. Без NO TRANSACTION goose оборачивает всё в одну
--      tx — COMMIT в DO упадёт, и батчи будут только косметикой.
--
-- Порядок «ALTER → backfill → CREATE INDEX» намеренный: индекс по пустой
-- колонке быстрее построить за один проход, чем поддерживать его при
-- per-row UPDATE в backfill (двойная запись WAL).
--
-- Семантика «пусто» — NULL (а не пустой массив). Совпадает с тем, как
-- extractRouteCityIDs/CountryIDs в Go-handler возвращает nil → pgx пишет
-- NULL. Это инвариант: `<column> IS NULL` означает «у точки нет id такого
-- типа», и не путается с пустым массивом из ARRAY[]::INTEGER[].

-- +goose Up

-- +goose StatementBegin
ALTER TABLE freight_requests_lookup
    ADD COLUMN IF NOT EXISTS origin_city_ids         INTEGER[],
    ADD COLUMN IF NOT EXISTS origin_country_ids      INTEGER[],
    ADD COLUMN IF NOT EXISTS destination_city_ids    INTEGER[],
    ADD COLUMN IF NOT EXISTS destination_country_ids INTEGER[];
-- +goose StatementEnd

-- Backfill из существующего route JSONB по батчам.
--
-- Каждый батч обновляет до 1000 строк и завершается COMMIT — это даёт:
--   - row-locks (UPDATE держит lock на каждую строку до конца своей tx)
--     отпускаются сразу после COMMIT, не дожидаясь конца миграции;
--   - autovacuum'у двинуть xmin horizon — иначе всё время backfill таблица
--     накапливает dead tuples без шанса их вычистить;
--   - другим UPDATE'ам на эти же строки попадать в окна между батчами.
--
-- Guard'ы:
--   1. route IS NOT NULL — иначе backfill не нужен;
--   2. route ? 'points'  — на случай битых данных без ключа;
--   3. jsonb_typeof(route->'points') = 'array' — защита от подмены типа;
--   4. id > last_id      — keyset-пагинация по PK; продвигаем через max(id)
--      батча независимо от того, обновились ли колонки в NULL (битый
--      маршрут без city/country: backfill оставит NULL, но мы не должны
--      зациклиться на этой же строке).
--
-- На повторный запуск миграция устойчива: уже backfill'нутые строки
-- по-прежнему попадают в батч (мы НЕ фильтруем по IS NULL — это могло бы
-- зациклиться на строках с битыми routes), но переписываются тем же
-- значением — идемпотентно.
--
-- +goose StatementBegin
DO $$
DECLARE
    last_id UUID := '00000000-0000-0000-0000-000000000000';
    max_id  UUID;
BEGIN
    LOOP
        max_id := NULL;

        WITH batch AS (
            SELECT id
            FROM freight_requests_lookup
            WHERE id > last_id
              AND route IS NOT NULL
              AND route ? 'points'
              AND jsonb_typeof(route->'points') = 'array'
              AND jsonb_array_length(route->'points') > 0
            ORDER BY id
            LIMIT 1000
        ),
        updated AS (
            UPDATE freight_requests_lookup l
            SET
                origin_city_ids = CASE
                    WHEN (l.route->'points'->0->>'city_id') IS NULL THEN NULL
                    ELSE ARRAY[(l.route->'points'->0->>'city_id')::INTEGER]
                END,
                origin_country_ids = CASE
                    WHEN (l.route->'points'->0->>'country_id') IS NULL THEN NULL
                    ELSE ARRAY[(l.route->'points'->0->>'country_id')::INTEGER]
                END,
                destination_city_ids = CASE
                    WHEN (l.route->'points'->-1->>'city_id') IS NULL THEN NULL
                    ELSE ARRAY[(l.route->'points'->-1->>'city_id')::INTEGER]
                END,
                destination_country_ids = CASE
                    WHEN (l.route->'points'->-1->>'country_id') IS NULL THEN NULL
                    ELSE ARRAY[(l.route->'points'->-1->>'country_id')::INTEGER]
                END
            FROM batch
            WHERE l.id = batch.id
            RETURNING l.id
        )
        SELECT id INTO max_id FROM updated ORDER BY id DESC LIMIT 1;

        EXIT WHEN max_id IS NULL;
        last_id := max_id;
        COMMIT;
    END LOOP;
END $$;
-- +goose StatementEnd

-- Индексы создаём CONCURRENTLY и ПОСЛЕ backfill: на горячей таблице
-- ACCESS EXCLUSIVE из обычного CREATE INDEX заморозил бы записи в
-- проекцию на всё время построения. Строим уже по заполненным колонкам —
-- один проход вместо инкрементальной поддержки во время UPDATE.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_freight_requests_origin_city_ids
    ON freight_requests_lookup USING GIN (origin_city_ids);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_freight_requests_origin_country_ids
    ON freight_requests_lookup USING GIN (origin_country_ids);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_freight_requests_destination_city_ids
    ON freight_requests_lookup USING GIN (destination_city_ids);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_freight_requests_destination_country_ids
    ON freight_requests_lookup USING GIN (destination_country_ids);

-- +goose Down

DROP INDEX CONCURRENTLY IF EXISTS idx_freight_requests_destination_country_ids;
DROP INDEX CONCURRENTLY IF EXISTS idx_freight_requests_destination_city_ids;
DROP INDEX CONCURRENTLY IF EXISTS idx_freight_requests_origin_country_ids;
DROP INDEX CONCURRENTLY IF EXISTS idx_freight_requests_origin_city_ids;

-- +goose StatementBegin
ALTER TABLE freight_requests_lookup
    DROP COLUMN IF EXISTS destination_country_ids,
    DROP COLUMN IF EXISTS destination_city_ids,
    DROP COLUMN IF EXISTS origin_country_ids,
    DROP COLUMN IF EXISTS origin_city_ids;
-- +goose StatementEnd
