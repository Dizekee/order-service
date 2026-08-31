package repository

import (
	"database/sql"
	"time"

	"github.com/Dizekee/order-service/internal/models"

	"github.com/google/uuid"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) CreateOrder(order *models.Order) error {
	query := `
		INSERT INTO orders (id, sku, amount, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	now := time.Now()
	order.ID = uuid.New()
	order.Status = models.StatusCreated
	order.CreatedAt = now
	order.UpdatedAt = now

	_, err := r.db.Exec(query, order.ID, order.SKU, order.Amount, order.Status, order.CreatedAt, order.UpdatedAt)
	return err
}

func (r *OrderRepository) GetOrderByID(id uuid.UUID) (*models.Order, error) {
	query := `SELECT id, sku, amount, status, created_at, updated_at FROM orders WHERE id = $1`
	row := r.db.QueryRow(query, id)

	var order models.Order
	err := row.Scan(&order.ID, &order.SKU, &order.Amount, &order.Status, &order.CreatedAt, &order.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *OrderRepository) GetOrderByIDForUpdate(id uuid.UUID) (*models.Order, error) {
	query := `SELECT id, sku, amount, status, created_at, updated_at FROM orders WHERE id = $1 FOR UPDATE`
	row := r.db.QueryRow(query, id)

	var order models.Order
	err := row.Scan(&order.ID, &order.SKU, &order.Amount, &order.Status, &order.CreatedAt, &order.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *OrderRepository) UpdateOrderStatus(id uuid.UUID, status models.OrderStatus) error {
	query := `UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.Exec(query, status, time.Now(), id)
	return err
}

func (r *OrderRepository) UpdateOrderStatusTx(tx *sql.Tx, id uuid.UUID, status models.OrderStatus) error {
	query := `UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := tx.Exec(query, status, time.Now(), id)
	return err
}

func (r *OrderRepository) GetProductBySKU(sku string) (*models.Product, error) {
	query := `SELECT sku, name, type, price, currency FROM products WHERE sku = $1`
	row := r.db.QueryRow(query, sku)

	var product models.Product
	err := row.Scan(&product.SKU, &product.Name, &product.Type, &product.Price, &product.Currency)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &product, nil
}
