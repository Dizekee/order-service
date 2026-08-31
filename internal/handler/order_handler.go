package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Dizekee/order-service/internal/models"
	"github.com/Dizekee/order-service/internal/service"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type OrderHandler struct {
	orderService *service.OrderService
}

func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
	}
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req models.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SKU == "" {
		http.Error(w, "SKU is required", http.StatusBadRequest)
		return
	}

	order, err := h.orderService.CreateOrder(req.SKU)
	if err != nil {
		log.Printf("Failed to create order: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.OrderResponse{Order: *order})
}

func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	order, err := h.orderService.GetOrder(id)
	if err != nil {
		log.Printf("Failed to get order: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.OrderResponse{Order: *order})
}

func (h *OrderHandler) PaymentWebhook(w http.ResponseWriter, r *http.Request) {
	var req models.PaymentWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode webhook request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Received webhook: event_id=%s, order_id=%s, status=%s", req.EventID, req.OrderID, req.Status)

	if req.EventID == "" || req.OrderID == "" || req.Status == "" {
		http.Error(w, "event_id, order_id and status are required", http.StatusBadRequest)
		return
	}

	err := h.orderService.ProcessPaymentWebhook(req.EventID, req.OrderID, req.Status, req.Amount, req.Currency)
	if err != nil {
		log.Printf("Failed to process webhook: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
