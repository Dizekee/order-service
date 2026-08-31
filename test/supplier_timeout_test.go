package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSupplierTimeout(t *testing.T) {
	if !isServerRunning() {
		t.Skip("Server is not running, start with 'make run' or 'make docker-up'")
	}

	orderID := createTestOrder(t, "STEAM-TOPUP-500")
	t.Logf("Created order: %s", orderID)

	eventID := "evt_timeout_" + uuid.New().String()
	webhookPayload := map[string]interface{}{
		"event_id": eventID,
		"order_id": orderID,
		"status":   "paid",
		"amount":   500,
		"currency": "RUB",
	}
	sendWebhook(t, webhookPayload)

	t.Log("Waiting for delivery (suppliers may timeout)...")
	time.Sleep(10 * time.Second)

	order := getTestOrder(t, orderID)
	t.Logf("Order status: %s", order.Status)

	if order.Status != "delivered" {
		t.Errorf("Expected status 'delivered', got '%s'", order.Status)
	} else {
		t.Log("Order delivered successfully")
	}

	count := getIssuedCodesCount(t, orderID)
	t.Logf("Issued codes count: %d", count)

	if count != 1 {
		t.Errorf("Expected exactly 1 issued code, got %d", count)
	} else {
		t.Log("Exactly one code issued")
	}

	t.Log("TestSupplierTimeout PASSED")
}

func TestSupplierFallback(t *testing.T) {
	if !isServerRunning() {
		t.Skip("Server is not running, start with 'make run' or 'make docker-up'")
	}

	if !isSupplierAvailable("http://localhost:8081") {
		t.Skip("Supplier A is not available, start with 'make supplier-up'")
	}

	orderID := createTestOrder(t, "STEAM-TOPUP-1000")
	t.Logf("Created order: %s", orderID)

	eventID := "evt_fallback_" + uuid.New().String()
	webhookPayload := map[string]interface{}{
		"event_id": eventID,
		"order_id": orderID,
		"status":   "paid",
		"amount":   1000,
		"currency": "RUB",
	}
	sendWebhook(t, webhookPayload)

	time.Sleep(5 * time.Second)

	order := getTestOrder(t, orderID)
	t.Logf("Order status: %s", order.Status)

	if order.Status != "delivered" {
		t.Errorf("Expected status 'delivered', got '%s'", order.Status)
	}

	count := getIssuedCodesCount(t, orderID)
	if count != 1 {
		t.Errorf("Expected exactly 1 issued code, got %d", count)
	}

	t.Log("TestSupplierFallback PASSED")
}

func TestExactlyOnceWithTimeout(t *testing.T) {
	if !isServerRunning() {
		t.Skip("Server is not running, start with 'make run' or 'make docker-up'")
	}

	orderID := createTestOrder(t, "KEY-CS2-PRIME")
	t.Logf("Created order: %s", orderID)

	eventID := "evt_exactly_once_" + uuid.New().String()
	webhookPayload := map[string]interface{}{
		"event_id": eventID,
		"order_id": orderID,
		"status":   "paid",
		"amount":   1290,
		"currency": "RUB",
	}
	sendWebhook(t, webhookPayload)

	time.Sleep(1 * time.Second)

	t.Log("Sending duplicate webhook with same event_id...")
	sendWebhook(t, webhookPayload)

	time.Sleep(10 * time.Second)

	count := getIssuedCodesCount(t, orderID)
	t.Logf("Issued codes count: %d", count)

	if count != 1 {
		t.Errorf("Expected exactly 1 issued code, got %d", count)
	} else {
		t.Log("Exactly one code issued despite duplicate webhook")
	}

	order := getTestOrder(t, orderID)
	if order.Status != "delivered" {
		t.Errorf("Expected status 'delivered', got '%s'", order.Status)
	}

	t.Log("TestExactlyOnceWithTimeout PASSED")
}

func sendWebhook(t *testing.T, payload map[string]interface{}) {
	body, _ := json.Marshal(payload)
	resp, err := http.Post(
		baseURL+"/webhook/payment",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("Failed to send webhook: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Webhook returned status: %d", resp.StatusCode)
	}
}

func isSupplierAvailable(url string) bool {
	resp, err := http.Get(url + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func getIssuedCodesCountWithRetry(t *testing.T, orderID string, maxAttempts int) int {
	for i := 0; i < maxAttempts; i++ {
		count := getIssuedCodesCount(t, orderID)
		if count >= 0 {
			return count
		}
		time.Sleep(1 * time.Second)
	}
	return -1
}
func getIssuedCodesCountFromDB(t *testing.T, orderID string) int {
	return getIssuedCodesCount(t, orderID)
}
