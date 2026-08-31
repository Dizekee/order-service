package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/Dizekee/order-service/internal/config"
	"github.com/Dizekee/order-service/internal/handler"
	"github.com/Dizekee/order-service/internal/repository"
	"github.com/Dizekee/order-service/internal/service"
	"github.com/Dizekee/order-service/internal/supplier"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.Load()

	db, err := sql.Open("postgres", getDBConnString(cfg))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Database connected successfully")

	supplierClient := supplier.NewClient("", 0)

	orderRepo := repository.NewOrderRepository(db)
	webhookRepo := repository.NewWebhookRepository(db)
	codeRepo := repository.NewCodeRepository(db)

	orderService := service.NewOrderService(
		orderRepo,
		webhookRepo,
		codeRepo,
		db,
		supplierClient,
		cfg.Supplier,
	)

	orderHandler := handler.NewOrderHandler(orderService)

	r := mux.NewRouter()

	api := r.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/orders", orderHandler.CreateOrder).Methods("POST")
	api.HandleFunc("/orders/{id}", orderHandler.GetOrder).Methods("GET")
	api.HandleFunc("/webhook/payment", orderHandler.PaymentWebhook).Methods("POST")

	reconRepo := repository.NewReconciliationRepository(db)
	reconciliationService := service.NewReconciliationService(
		orderRepo,
		codeRepo,
		reconRepo,
		orderService,
	)

	reconciliationService.StartReconciliationScheduler(10 * time.Minute)

	reconciliationHandler := handler.NewReconciliationHandler(reconciliationService)

	api.HandleFunc("/admin/reconcile", reconciliationHandler.RunReconciliation).Methods("POST")
	api.HandleFunc("/admin/recover", reconciliationHandler.RecoverStuck).Methods("POST")

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	port := cfg.Server.Port
	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

}

func getDBConnString(cfg *config.Config) string {
	return "host=" + cfg.Database.Host +
		" port=" + cfg.Database.Port +
		" user=" + cfg.Database.User +
		" password=" + cfg.Database.Password +
		" dbname=" + cfg.Database.Name +
		" sslmode=" + cfg.Database.SSLMode
}
