-- +goose Up
-- +goose StatementBegin
-- Pre-launch смена флоу модерации транспорта: авто-постановка на модерацию
-- (status=pending при add/update) заменена на ручную отправку владельцем
-- (unconfirmed → submit → pending). Висящие pending без явной отправки больше
-- не валидны: переводим в unconfirmed и очищаем очередь модерации — владельцы
-- отправят машины на проверку заново кнопкой «Подтвердить».
UPDATE vehicles_lookup SET status = 'unconfirmed' WHERE status = 'pending';
DELETE FROM pending_vehicles;

-- Снапшоты агрегата организации хранят VehicleSnapshot.Status дословно: pending
-- из старого флоу пережил бы миграцию и при следующем rebuild-from-aggregate
-- (LoadWithSnapshot → FromSnapshot) откатил бы vehicles_lookup в pending и
-- вернул машину в pending_vehicles. Удаляем снапшоты — полный replay по новой
-- apply()-семантике даёт unconfirmed (pending только через явный submit);
-- снапшот пересоздастся при следующем SaveWithState на пороге версий.
DELETE FROM snapshots WHERE aggregate_type = 'organization';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE vehicles_lookup SET status = 'pending' WHERE status = 'unconfirmed';

-- Up очистил pending_vehicles; старый код читает очередь модерации из этой
-- таблицы, а не из vehicles_lookup.status — восстанавливаем связность проекций
-- (submitted_at ← updated_at, как в rebuild хендлера).
INSERT INTO pending_vehicles (id, org_id, registration_number, brand, model, vehicle_type, vehicle_subtype, submitted_at)
SELECT id, org_id, registration_number, brand, model, vehicle_type, vehicle_subtype, updated_at
FROM vehicles_lookup
WHERE status = 'pending'
ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd
