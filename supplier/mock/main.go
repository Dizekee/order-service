package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	issuedCodes = make(map[string]string)
	mu          sync.RWMutex
)

type IssueRequest struct {
	RequestID string `json:"request_id"`
	SKU       string `json:"sku"`
	OrderID   string `json:"order_id"`
}

type IssueResponse struct {
	Status    string `json:"status"`
	RequestID string `json:"request_id"`
	Code      string `json:"code,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	failRate := getFloatEnv("FAIL_RATE", 0.3)
	timeoutRate := getFloatEnv("TIMEOUT_RATE", 0.2)
	timeoutDuration := getDurationEnv("TIMEOUT_DURATION", 5*time.Second)
	supplierID := os.Getenv("SUPPLIER_ID")
	if supplierID == "" {
		supplierID = "A"
	}

	keys := []string{
		"SUPPLIER-" + supplierID + "-KEY-001",
		"SUPPLIER-" + supplierID + "-KEY-002",
		"SUPPLIER-" + supplierID + "-KEY-003",
		"SUPPLIER-" + supplierID + "-KEY-004",
		"SUPPLIER-" + supplierID + "-KEY-005",
		"SUPPLIER-" + supplierID + "-KEY-006",
		"SUPPLIER-" + supplierID + "-KEY-007",
		"SUPPLIER-" + supplierID + "-KEY-008",
		"SUPPLIER-" + supplierID + "-KEY-009",
		"SUPPLIER-" + supplierID + "-KEY-010",
	}

	mu.Lock()
	for i, key := range keys {
		requestID := "req-init-" + supplierID + "-" + strconv.Itoa(i)
		issuedCodes[requestID] = key
	}
	mu.Unlock()

	log.Printf("Supplier %s starting on port %s", supplierID, port)
	log.Printf("Fail rate: %.2f, Timeout rate: %.2f, Timeout: %v",
		failRate, timeoutRate, timeoutDuration)

	http.HandleFunc("/issue", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		delay := time.Duration(rand.Intn(1000)) * time.Millisecond
		time.Sleep(delay)

		if rand.Float64() < timeoutRate {
			log.Printf("[%s] Simulating timeout (request will hang)", supplierID)
			time.Sleep(timeoutDuration)
		}

		if rand.Float64() < failRate {
			log.Printf("[%s] Simulating error", supplierID)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(IssueResponse{
				Status: "error",
				Reason: "internal_server_error",
			})
			return
		}

		var req IssueRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if req.RequestID == "" {
			http.Error(w, "request_id required", http.StatusBadRequest)
			return
		}

		log.Printf("[%s] Received request: request_id=%s, sku=%s, order_id=%s",
			supplierID, req.RequestID, req.SKU, req.OrderID)

		mu.RLock()
		code, exists := issuedCodes[req.RequestID]
		mu.RUnlock()

		if exists {
			log.Printf("[%s] Returning existing code for request_id=%s: %s",
				supplierID, req.RequestID, code)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(IssueResponse{
				Status:    "ok",
				RequestID: req.RequestID,
				Code:      code,
			})
			return
		}

		if len(keys) > 0 {
			newCode := keys[0]
			keys = keys[1:]

			mu.Lock()
			issuedCodes[req.RequestID] = newCode
			mu.Unlock()

			log.Printf("[%s] Issued new code for request_id=%s: %s",
				supplierID, req.RequestID, newCode)

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(IssueResponse{
				Status:    "ok",
				RequestID: req.RequestID,
				Code:      newCode,
			})
			return
		}

		log.Printf("[%s] Out of stock for request_id=%s", supplierID, req.RequestID)
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(IssueResponse{
			Status: "error",
			Reason: "out_of_stock",
		})
	})

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func getFloatEnv(key string, defaultValue float64) float64 {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return defaultValue
	}
	return f
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return defaultValue
	}
	return d
}
