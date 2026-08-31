package repository

import (
	"database/sql"
	"time"

	"github.com/Dizekee/order-service/internal/models"
)

type WebhookRepository struct {
	db *sql.DB
}

func NewWebhookRepository(db *sql.DB) *WebhookRepository {
	return &WebhookRepository{db: db}
}

func (r *WebhookRepository) CreateWebhookEvent(event *models.WebhookEvent) error {
	query := `
		INSERT INTO webhook_events (event_id, order_id, status, amount, currency, processed_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	event.ProcessedAt = time.Now()
	_, err := r.db.Exec(query, event.EventID, event.OrderID, event.Status, event.Amount, event.Currency, event.ProcessedAt)
	return err
}

func (r *WebhookRepository) WebhookExists(eventID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM webhook_events WHERE event_id = $1)`
	var exists bool
	err := r.db.QueryRow(query, eventID).Scan(&exists)
	return exists, err
}

func (r *WebhookRepository) CreateWebhookEventTx(tx *sql.Tx, event *models.WebhookEvent) error {
	query := `
		INSERT INTO webhook_events (event_id, order_id, status, amount, currency, processed_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	event.ProcessedAt = time.Now()
	_, err := tx.Exec(query, event.EventID, event.OrderID, event.Status, event.Amount, event.Currency, event.ProcessedAt)
	return err
}
