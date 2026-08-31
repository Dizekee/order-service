package repository

import (
	"database/sql"

	"github.com/google/uuid"
)

type CodeRepository struct {
	db *sql.DB
}

func NewCodeRepository(db *sql.DB) *CodeRepository {
	return &CodeRepository{db: db}
}

func (r *CodeRepository) GetAvailableCode() (string, error) {
	query := `SELECT code FROM key_pool WHERE used = false LIMIT 1 FOR UPDATE SKIP LOCKED`
	row := r.db.QueryRow(query)
	var code string
	err := row.Scan(&code)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return code, nil
}

func (r *CodeRepository) ReserveCodeTx(tx *sql.Tx, code string, orderID uuid.UUID) error {
	query := `UPDATE key_pool SET used = true, order_id = $1, reserved_at = NOW() WHERE code = $2 AND used = false`
	result, err := tx.Exec(query, orderID, code)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *CodeRepository) SaveIssuedCodeTx(tx *sql.Tx, orderID uuid.UUID, code, supplierID, requestID string) error {
	query := `
		INSERT INTO issued_codes (order_id, code, supplier_id, request_id, issued_at)
		VALUES ($1, $2, $3, $4, NOW())
	`
	_, err := tx.Exec(query, orderID, code, supplierID, requestID)
	return err
}

func (r *CodeRepository) GetIssuedCodeByOrderID(orderID uuid.UUID) (string, error) {
	query := `SELECT code FROM issued_codes WHERE order_id = $1`
	row := r.db.QueryRow(query, orderID)
	var code string
	err := row.Scan(&code)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return code, nil
}
