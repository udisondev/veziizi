-- +goose NO TRANSACTION
-- CONCURRENTLY нельзя выполнять внутри транзакции, поэтому используем
-- NO TRANSACTION. Каждый CREATE INDEX выполняется отдельным statement-ом и
-- не берёт ACCESS EXCLUSIVE на freight_requests_lookup (только short lock в
-- начале/конце). На большой таблице это критично — иначе три последовательных
-- CREATE INDEX заморозят запись в проекцию на всё время построения индексов.

-- +goose Up

-- Покрываем range-фильтры WithMinPrice/WithMaxPrice/WithMinWeight/WithMaxWeight
-- из FreightRequestsProjection. На большом public marketplace без этих индексов
-- любой "ищу заявки от 10000₽" приводит к full table scan.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_freight_requests_price_amount
    ON freight_requests_lookup(price_amount);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_freight_requests_cargo_weight
    ON freight_requests_lookup(cargo_weight);

-- WithFreightCarrierOrgID используется для "мои подтверждённые перевозки".
-- Без индекса — full scan по всему marketplace на каждой загрузке дашборда
-- перевозчика. carrier_org_id NULLable, частично нагружен → btree уместен.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_freight_requests_carrier_org
    ON freight_requests_lookup(carrier_org_id)
    WHERE carrier_org_id IS NOT NULL;

-- +goose Down

DROP INDEX CONCURRENTLY IF EXISTS idx_freight_requests_carrier_org;
DROP INDEX CONCURRENTLY IF EXISTS idx_freight_requests_cargo_weight;
DROP INDEX CONCURRENTLY IF EXISTS idx_freight_requests_price_amount;
