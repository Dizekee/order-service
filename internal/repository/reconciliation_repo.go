package repository

import (
	"database/sql"
	"time"

	"github.com/Dizekee/order-service/internal/models"
	"github.com/google/uuid"
)

type ReconciliationRepository struct {
	db *sql.DB
}

func NewReconciliationRepository(db *sql.DB) *ReconciliationRepository {
	return &ReconciliationRepository{db: db}
}

func (r *ReconciliationRepository) GetOrdersPaidNotDelivered() ([]uuid.UUID, error) {
	query := `
        SELECT id FROM orders 
        WHERE status IN ('paid', 'delivering', 'out_of_stock', 'delivery_failed')
        AND NOT EXISTS (SELECT 1 FROM issued_codes WHERE order_id = orders.id)
    `
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *ReconciliationRepository) GetOrdersDeliveredNotPaid() ([]uuid.UUID, error) {
	query := `
        SELECT o.id FROM orders o
        WHERE EXISTS (SELECT 1 FROM issued_codes WHERE order_id = o.id)
        AND NOT EXISTS (
            SELECT 1 FROM webhook_events we 
            WHERE we.order_id = o.id AND we.status = 'paid'
        )
    `
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *ReconciliationRepository) SaveReconciliationResult(result *models.ReconciliationResult) error {
	query := `
        INSERT INTO reconciliation_results (order_id, issue_type, description, created_at)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (order_id, issue_type) DO NOTHING
    `
	_, err := r.db.Exec(query, result.OrderID, result.IssueType, result.Description, time.Now())
	return err
}

func (r *ReconciliationRepository) GetStuckOrders(olderThan time.Duration) ([]uuid.UUID, error) {
	query := `
        SELECT id FROM orders 
        WHERE status IN ('delivering', 'out_of_stock', 'delivery_failed')
        AND updated_at < $1
        AND retry_count < 5
    `
	cutoff := time.Now().Add(-olderThan)
	rows, err := r.db.Query(query, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *ReconciliationRepository) IncrementRetryCount(orderID uuid.UUID) error {
	query := `UPDATE orders SET retry_count = retry_count + 1, last_retry_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(query, orderID)
	return err
}

func (r *ReconciliationRepository) SaveLedgerEntry(entry *models.LedgerEntry) error {
	query := `
        INSERT INTO ledger_entries (order_id, event_id, entry_type, amount, currency, balance_before, balance_after)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        ON CONFLICT (order_id, event_id) DO NOTHING
    `
	_, err := r.db.Exec(
		query,
		entry.OrderID,
		entry.EventID,
		entry.EntryType,
		entry.Amount,
		entry.Currency,
		entry.BalanceBefore,
		entry.BalanceAfter,
	)
	return err
}
