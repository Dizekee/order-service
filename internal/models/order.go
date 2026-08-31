package models

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	StatusCreated        OrderStatus = "created"
	StatusPaid           OrderStatus = "paid"
	StatusDelivering     OrderStatus = "delivering"
	StatusDelivered      OrderStatus = "delivered"
	StatusPaymentFailed  OrderStatus = "payment_failed"
	StatusOutOfStock     OrderStatus = "out_of_stock"
	StatusDeliveryFailed OrderStatus = "delivery_failed"
)

type Order struct {
	ID        uuid.UUID   `json:"id"`
	SKU       string      `json:"sku"`
	Amount    int         `json:"amount"`
	Status    OrderStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type WebhookEvent struct {
	EventID     string    `json:"event_id"`
	OrderID     uuid.UUID `json:"order_id"`
	Status      string    `json:"status"`
	Amount      int       `json:"amount"`
	Currency    string    `json:"currency"`
	ProcessedAt time.Time `json:"processed_at"`
}

type IssuedCode struct {
	ID         int       `json:"id"`
	OrderID    uuid.UUID `json:"order_id"`
	Code       string    `json:"code"`
	SupplierID string    `json:"supplier_id"`
	RequestID  string    `json:"request_id"`
	IssuedAt   time.Time `json:"issued_at"`
}

type Product struct {
	SKU      string `json:"sku"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Price    int    `json:"price"`
	Currency string `json:"currency"`
}

type CreateOrderRequest struct {
	SKU string `json:"sku"`
}

type OrderResponse struct {
	Order Order `json:"order"`
}

type PaymentWebhookRequest struct {
	EventID   string `json:"event_id"`
	OrderID   string `json:"order_id"`
	Status    string `json:"status"`
	Amount    int    `json:"amount"`
	Currency  string `json:"currency"`
	CreatedAt string `json:"created_at"`
}

type LedgerEntry struct {
	ID            int       `json:"id"`
	OrderID       uuid.UUID `json:"order_id"`
	EventID       string    `json:"event_id"`
	EntryType     string    `json:"entry_type"`
	Amount        int       `json:"amount"`
	Currency      string    `json:"currency"`
	BalanceBefore int       `json:"balance_before"`
	BalanceAfter  int       `json:"balance_after"`
	CreatedAt     time.Time `json:"created_at"`
}

type ReconciliationResult struct {
	ID          int        `json:"id"`
	OrderID     uuid.UUID  `json:"order_id"`
	IssueType   string     `json:"issue_type"`
	Description string     `json:"description"`
	Resolved    bool       `json:"resolved"`
	CreatedAt   time.Time  `json:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// Для расширения Order
type OrderExtended struct {
	Order
	ErrorMessage string     `json:"error_message,omitempty"`
	LastRetryAt  *time.Time `json:"last_retry_at,omitempty"`
	RetryCount   int        `json:"retry_count"`
}
