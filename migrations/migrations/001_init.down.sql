DROP INDEX IF EXISTS idx_issued_codes_order_id;
DROP INDEX IF EXISTS idx_webhook_events_order_id;
DROP INDEX IF EXISTS idx_orders_created_at;
DROP INDEX IF EXISTS idx_orders_status;

DROP TABLE IF EXISTS issued_codes;
DROP TABLE IF EXISTS webhook_events;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS key_pool;
DROP TABLE IF EXISTS products;