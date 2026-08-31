DROP INDEX IF EXISTS idx_reconciliation_resolved;
DROP INDEX IF EXISTS idx_reconciliation_order_id;
DROP INDEX IF EXISTS idx_ledger_created_at;
DROP INDEX IF EXISTS idx_ledger_order_id;

DROP TABLE IF EXISTS reconciliation_results;
DROP TABLE IF EXISTS ledger_entries;

ALTER TABLE orders DROP COLUMN IF EXISTS error_message;
ALTER TABLE orders DROP COLUMN IF EXISTS last_retry_at;
ALTER TABLE orders DROP COLUMN IF EXISTS retry_count;