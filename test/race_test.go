package integration

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

const (
	baseURL = "http://localhost:8080/api/v1"
)

func TestRaceCondition(t *testing.T) {
	if !isServerRunning() {
		t.Skip("Server is not running, start with 'make run' or 'make docker-up'")
	}

	orderID := createTestOrder(t, "STEAM-TOPUP-500")
	t.Logf("Created order: %s", orderID)

	var wg sync.WaitGroup
	eventID := "evt_race_" + uuid.New().String()

	webhookPayload := map[string]interface{}{
		"event_id": eventID,
		"order_id": orderID,
		"status":   "paid",
		"amount":   500,
		"currency": "RUB",
	}

	payloadBytes, _ := json.Marshal(webhookPayload)

	successCount := 0
	var mu sync.Mutex

	t.Logf("Sending 50 parallel webhooks with event_id: %s", eventID)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			resp, err := http.Post(
				baseURL+"/webhook/payment",
				"application/json",
				bytes.NewReader(payloadBytes),
			)
			if err != nil {
				t.Logf("Request #%d failed: %v", idx, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				mu.Lock()
				successCount++
				mu.Unlock()
			} else {
				t.Logf("Request #%d returned status: %d", idx, resp.StatusCode)
			}
		}(i)
	}

	wg.Wait()
	t.Logf("Successful responses: %d/50", successCount)

	time.Sleep(2 * time.Second)

	order := getTestOrder(t, orderID)
	t.Logf("Order status: %s", order.Status)

	if order.Status != "delivered" {
		t.Errorf("Expected status 'delivered', got '%s'", order.Status)
	} else {
		t.Log("Order is delivered")
	}

	issuedCount := getIssuedCodesCount(t, orderID)
	t.Logf("Issued codes count: %d", issuedCount)

	if issuedCount != 1 {
		t.Errorf("Expected exactly 1 issued code, got %d", issuedCount)
	} else {
		t.Log("Exactly one code issued")
	}

	t.Log("TestRaceCondition PASSED")
}

func TestIdempotency(t *testing.T) {
	if !isServerRunning() {
		t.Skip("Server is not running, start with 'make run' or 'make docker-up'")
	}

	orderID := createTestOrder(t, "STEAM-TOPUP-1000")
	t.Logf("Created order: %s", orderID)

	eventID := "evt_idempotent_" + uuid.New().String()
	webhookPayload := map[string]interface{}{
		"event_id": eventID,
		"order_id": orderID,
		"status":   "paid",
		"amount":   1000,
		"currency": "RUB",
	}

	payloadBytes, _ := json.Marshal(webhookPayload)

	t.Logf("Sending 3 identical webhooks with event_id: %s", eventID)

	for i := 0; i < 3; i++ {
		resp, err := http.Post(
			baseURL+"/webhook/payment",
			"application/json",
			bytes.NewReader(payloadBytes),
		)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	}

	time.Sleep(1 * time.Second)

	order := getTestOrder(t, orderID)
	t.Logf("Order status: %s", order.Status)

	if order.Status != "delivered" {
		t.Errorf("Expected status 'delivered', got '%s'", order.Status)
	} else {
		t.Log("Order is delivered")
	}

	issuedCount := getIssuedCodesCount(t, orderID)
	t.Logf("Issued codes count: %d", issuedCount)

	if issuedCount != 1 {
		t.Errorf("Expected exactly 1 issued code, got %d", issuedCount)
	} else {
		t.Log("Exactly one code issued")
	}

	t.Log("TestIdempotency PASSED")
}

func TestWebhookBeforeOrder(t *testing.T) {
	if !isServerRunning() {
		t.Skip("Server is not running, start with 'make run' or 'make docker-up'")
	}

	orderID := uuid.New().String()
	eventID := "evt_before_" + uuid.New().String()

	webhookPayload := map[string]interface{}{
		"event_id": eventID,
		"order_id": orderID,
		"status":   "paid",
		"amount":   500,
		"currency": "RUB",
	}

	payloadBytes, _ := json.Marshal(webhookPayload)

	t.Logf("Sending webhook for non-existent order: %s", orderID)

	resp, err := http.Post(
		baseURL+"/webhook/payment",
		"application/json",
		bytes.NewReader(payloadBytes),
	)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	} else {
		t.Log("Webhook accepted for non-existent order")
	}

	newOrderID := createTestOrder(t, "STEAM-TOPUP-500")
	t.Logf("Created new order: %s", newOrderID)

	time.Sleep(1 * time.Second)

	order := getTestOrder(t, newOrderID)
	t.Logf("New order status: %s", order.Status)

	if order.Status == "delivered" {
		t.Error("Webhook was incorrectly applied to a different order")
	} else {
		t.Log("Webhook not applied to wrong order")
	}

	t.Log("TestWebhookBeforeOrder PASSED")
}

func isServerRunning() bool {
	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func createTestOrder(t *testing.T, sku string) string {
	reqBody := map[string]string{"sku": sku}
	body, _ := json.Marshal(reqBody)

	resp, err := http.Post(
		baseURL+"/orders",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("Failed to create order: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Order struct {
			ID string `json:"id"`
		} `json:"order"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	return result.Order.ID
}

func getTestOrder(t *testing.T, orderID string) *struct {
	ID     string `json:"id"`
	Status string `json:"status"`
} {
	resp, err := http.Get(baseURL + "/orders/" + orderID)
	if err != nil {
		t.Fatalf("Failed to get order: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Order struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"order"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	return &result.Order
}

func getIssuedCodesCount(t *testing.T, orderID string) int {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "postgres"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "orders"
	}

	connStr := "host=" + host + " port=" + port + " user=" + user +
		" password=" + password + " dbname=" + dbname + " sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Logf("Failed to connect to DB: %v", err)
		return -1
	}
	defer db.Close()

	var count int
	query := "SELECT COUNT(*) FROM issued_codes WHERE order_id = $1"
	err = db.QueryRow(query, orderID).Scan(&count)
	if err != nil {
		t.Logf("Failed to query issued codes: %v", err)
		return -1
	}

	return count
}
