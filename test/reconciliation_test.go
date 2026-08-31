package integration

import (
	"net/http"
	"testing"
	"time"
)

func TestReconciliation(t *testing.T) {
	if !isServerRunning() {
		t.Skip("Server is not running")
	}

	orderID := createTestOrder(t, "STEAM-TOPUP-500")
	sendWebhook(t, map[string]interface{}{
		"event_id": "evt_recon_" + orderID,
		"order_id": orderID,
		"status":   "paid",
		"amount":   500,
		"currency": "RUB",
	})

	time.Sleep(1 * time.Second)

	resp, err := http.Post(baseURL+"/admin/reconcile", "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to run reconciliation: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	t.Log("Reconciliation test passed")
}
