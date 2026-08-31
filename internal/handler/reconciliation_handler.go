package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Dizekee/order-service/internal/service"
)

type ReconciliationHandler struct {
	reconService *service.ReconciliationService
}

func NewReconciliationHandler(reconService *service.ReconciliationService) *ReconciliationHandler {
	return &ReconciliationHandler{
		reconService: reconService,
	}
}

func (h *ReconciliationHandler) RunReconciliation(w http.ResponseWriter, r *http.Request) {
	if err := h.reconService.RunReconciliation(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Reconciliation completed successfully",
	})
}

func (h *ReconciliationHandler) RecoverStuck(w http.ResponseWriter, r *http.Request) {
	if err := h.reconService.RecoverStuckOrders(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Stuck orders recovery completed",
	})
}
