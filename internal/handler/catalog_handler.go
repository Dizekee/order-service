package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/Dizekee/order-service/internal/models"
)

type CatalogHandler struct {
	db *sql.DB
}

func NewCatalogHandler(db *sql.DB) *CatalogHandler {
	return &CatalogHandler{db: db}
}

func (h *CatalogHandler) GetCatalog(w http.ResponseWriter, r *http.Request) {

	productType := r.URL.Query().Get("type")
	limit := 50
	offset := 0

	query := `
        SELECT sku, name, type, price, currency 
        FROM products 
        WHERE 1=1
    `
	args := []interface{}{}

	if productType != "" {
		query += " AND type = $" + string(len(args)+1)
		args = append(args, productType)
	}

	query += " ORDER BY price ASC LIMIT $" + string(len(args)+1) + " OFFSET $" + string(len(args)+2)
	args = append(args, limit, offset)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.SKU, &p.Name, &p.Type, &p.Price, &p.Currency); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		products = append(products, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"products": products,
		"count":    len(products),
	})
}
