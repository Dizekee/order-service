package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Dizekee/order-service/internal/config"
	"github.com/Dizekee/order-service/internal/models"
	"github.com/Dizekee/order-service/internal/repository"
	"github.com/Dizekee/order-service/internal/supplier"

	"github.com/google/uuid"
)

type OrderService struct {
	orderRepo      *repository.OrderRepository
	webhookRepo    *repository.WebhookRepository
	codeRepo       *repository.CodeRepository
	db             *sql.DB
	supplierClient *supplier.Client
	supplierConfig config.SupplierConfig
}

func NewOrderService(
	orderRepo *repository.OrderRepository,
	webhookRepo *repository.WebhookRepository,
	codeRepo *repository.CodeRepository,
	db *sql.DB,
	supplierClient *supplier.Client,
	supplierConfig config.SupplierConfig,
) *OrderService {
	return &OrderService{
		orderRepo:      orderRepo,
		webhookRepo:    webhookRepo,
		codeRepo:       codeRepo,
		db:             db,
		supplierClient: supplierClient,
		supplierConfig: supplierConfig,
	}
}

func (s *OrderService) CreateOrder(sku string) (*models.Order, error) {
	product, err := s.orderRepo.GetProductBySKU(sku)
	if err != nil {
		return nil, fmt.Errorf("failed to get product: %w", err)
	}
	if product == nil {
		return nil, errors.New("product not found")
	}

	order := &models.Order{
		SKU:    sku,
		Amount: product.Price,
	}

	if err := s.orderRepo.CreateOrder(order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	return order, nil
}

func (s *OrderService) GetOrder(id uuid.UUID) (*models.Order, error) {
	order, err := s.orderRepo.GetOrderByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	if order == nil {
		return nil, errors.New("order not found")
	}
	return order, nil
}

func (s *OrderService) ProcessPaymentWebhook(eventID, orderIDStr, status string, amount int, currency string) error {

	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		return fmt.Errorf("invalid order_id format: %w", err)
	}

	exists, err := s.webhookRepo.WebhookExists(eventID)
	if err != nil {
		return fmt.Errorf("failed to check webhook existence: %w", err)
	}
	if exists {
		log.Printf("Webhook %s already processed, skipping", eventID)
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	order, err := s.orderRepo.GetOrderByIDForUpdate(orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}
	if order == nil {
		return errors.New("order not found")
	}

	if order.Status == models.StatusDelivered ||
		order.Status == models.StatusPaymentFailed {
		log.Printf("Order %s already in final status %s, skipping", orderID, order.Status)
		if err := s.webhookRepo.CreateWebhookEventTx(tx, &models.WebhookEvent{
			EventID:  eventID,
			OrderID:  orderID,
			Status:   status,
			Amount:   amount,
			Currency: currency,
		}); err != nil {
			return fmt.Errorf("failed to save webhook event: %w", err)
		}
		return tx.Commit()
	}

	if status == "paid" {
		if order.Status == models.StatusPaid || order.Status == models.StatusDelivering {
			log.Printf("Order %s already paid/delivering, skipping", orderID)

			if err := s.webhookRepo.CreateWebhookEventTx(tx, &models.WebhookEvent{
				EventID:  eventID,
				OrderID:  orderID,
				Status:   status,
				Amount:   amount,
				Currency: currency,
			}); err != nil {
				return fmt.Errorf("failed to save webhook event: %w", err)
			}
			return tx.Commit()
		}

		if err := s.orderRepo.UpdateOrderStatusTx(tx, orderID, models.StatusPaid); err != nil {
			return fmt.Errorf("failed to update order status: %w", err)
		}

		if err := s.webhookRepo.CreateWebhookEventTx(tx, &models.WebhookEvent{
			EventID:  eventID,
			OrderID:  orderID,
			Status:   status,
			Amount:   amount,
			Currency: currency,
		}); err != nil {
			return fmt.Errorf("failed to save webhook event: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		if err := s.deliverProduct(orderID); err != nil {
			log.Printf("ERROR: failed to deliver product for order %s: %v", orderID, err)
			return err
		}

		return nil

	} else if status == "failed" {
		if order.Status == models.StatusCreated {
			if err := s.orderRepo.UpdateOrderStatusTx(tx, orderID, models.StatusPaymentFailed); err != nil {
				return fmt.Errorf("failed to update order status: %w", err)
			}
		}
		if err := s.webhookRepo.CreateWebhookEventTx(tx, &models.WebhookEvent{
			EventID:  eventID,
			OrderID:  orderID,
			Status:   status,
			Amount:   amount,
			Currency: currency,
		}); err != nil {
			return fmt.Errorf("failed to save webhook event: %w", err)
		}
		return tx.Commit()
	}

	return errors.New("unknown webhook status")
}

func (s *OrderService) deliverProduct(orderID uuid.UUID) error {

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	order, err := s.orderRepo.GetOrderByIDForUpdate(orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}
	if order == nil {
		return errors.New("order not found")
	}

	if order.Status == models.StatusDelivered {
		log.Printf("Order %s already delivered", orderID)
		return tx.Commit()
	}

	if err := s.orderRepo.UpdateOrderStatusTx(tx, orderID, models.StatusDelivering); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	code, err := s.codeRepo.GetAvailableCode()
	if err != nil {
		return fmt.Errorf("failed to get available code: %w", err)
	}
	if code == "" {
		if err := s.orderRepo.UpdateOrderStatusTx(tx, orderID, models.StatusOutOfStock); err != nil {
			return fmt.Errorf("failed to update order status: %w", err)
		}
		return tx.Commit()
	}

	if err := s.codeRepo.ReserveCodeTx(tx, code, orderID); err != nil {
		return fmt.Errorf("failed to reserve code: %w", err)
	}

	supplierID := "mock-supplier"
	requestID := fmt.Sprintf("req-%s-1", orderID.String())
	if err := s.codeRepo.SaveIssuedCodeTx(tx, orderID, code, supplierID, requestID); err != nil {
		return fmt.Errorf("failed to save issued code: %w", err)
	}

	if err := s.orderRepo.UpdateOrderStatusTx(tx, orderID, models.StatusDelivered); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	log.Printf("Order %s delivered with code: %s", orderID, code)

	return tx.Commit()
}

func (s *OrderService) ProcessPaymentWebhookEarly(eventID, orderIDStr, status string, amount int, currency string) error {
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		return fmt.Errorf("invalid order_id format: %w", err)
	}

	exists, err := s.webhookRepo.WebhookExists(eventID)
	if err != nil {
		return fmt.Errorf("failed to check webhook existence: %w", err)
	}
	if exists {
		log.Printf("Webhook %s already processed, skipping", eventID)
		return nil
	}

	order, err := s.orderRepo.GetOrderByID(orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}
	if order == nil {
		log.Printf("Order %s not found, saving webhook for later", orderID)

		if err := s.webhookRepo.CreateWebhookEvent(&models.WebhookEvent{
			EventID:  eventID,
			OrderID:  orderID,
			Status:   status,
			Amount:   amount,
			Currency: currency,
		}); err != nil {
			return fmt.Errorf("failed to save webhook event: %w", err)
		}
		return nil
	}

	return s.ProcessPaymentWebhook(eventID, orderIDStr, status, amount, currency)
}

func (s *OrderService) deliverProductWithSupplier(orderID uuid.UUID, sku string) error {
	log.Printf("Starting delivery for order %s, sku: %s", orderID, sku)

	requestID := fmt.Sprintf("req-%s-%d", orderID.String(), time.Now().UnixNano())

	code, err := s.trySupplier(orderID, sku, requestID, s.supplierConfig.SupplierAURL, "A")
	if err == nil && code != "" {
		return s.saveIssuedCode(orderID, code, "A", requestID)
	}

	log.Printf("Supplier A failed for order %s: %v, trying fallback B", orderID, err)

	code, err = s.trySupplier(orderID, sku, requestID, s.supplierConfig.SupplierBURL, "B")
	if err == nil && code != "" {
		return s.saveIssuedCode(orderID, code, "B", requestID)
	}

	log.Printf("Both suppliers failed for order %s", orderID)
	return s.updateOrderStatus(orderID, models.StatusDeliveryFailed)
}

func (s *OrderService) trySupplier(orderID uuid.UUID, sku, requestID, baseURL, supplierID string) (string, error) {
	maxAttempts := s.supplierConfig.RetryMaxAttempts
	backoff := s.supplierConfig.RetryBackoff

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			waitTime := backoff * time.Duration(1<<uint(attempt-1))
			log.Printf("Retry %d for supplier %s, order %s, waiting %v",
				attempt, supplierID, orderID, waitTime)
			time.Sleep(waitTime)
		}

		ctx, cancel := context.WithTimeout(context.Background(), s.supplierConfig.Timeout)
		defer cancel()

		req := &supplier.IssueRequest{
			RequestID: requestID,
			SKU:       sku,
			OrderID:   orderID.String(),
		}

		resp, err := s.supplierClient.IssueCode(ctx, req)
		if err != nil {
			log.Printf("Supplier %s attempt %d failed: %v", supplierID, attempt+1, err)
			continue
		}

		if resp.Status == "ok" && resp.Code != "" {
			log.Printf("Supplier %s returned code: %s for request_id: %s",
				supplierID, resp.Code, resp.RequestID)
			return resp.Code, nil
		}

		log.Printf("Supplier %s attempt %d returned error: %s",
			supplierID, attempt+1, resp.Reason)

		if resp.Reason == "out_of_stock" {
			return "", fmt.Errorf("supplier out of stock")
		}
	}

	return "", fmt.Errorf("all attempts failed for supplier %s", supplierID)
}

func (s *OrderService) saveIssuedCode(orderID uuid.UUID, code, supplierID, requestID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	existingCode, err := s.codeRepo.GetIssuedCodeByOrderID(orderID)
	if err != nil {
		return fmt.Errorf("failed to check existing code: %w", err)
	}
	if existingCode != "" {
		log.Printf("Order %s already has code: %s, skipping", orderID, existingCode)
		return tx.Commit()
	}

	if err := s.codeRepo.ReserveCodeTx(tx, code, orderID); err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Code %s already used, checking if it's for this order", code)
			existing, _ := s.codeRepo.GetIssuedCodeByOrderID(orderID)
			if existing != "" {
				return tx.Commit()
			}
			return fmt.Errorf("code %s already reserved by another order", code)
		}
		return fmt.Errorf("failed to reserve code: %w", err)
	}

	if err := s.codeRepo.SaveIssuedCodeTx(tx, orderID, code, supplierID, requestID); err != nil {
		return fmt.Errorf("failed to save issued code: %w", err)
	}

	if err := s.orderRepo.UpdateOrderStatusTx(tx, orderID, models.StatusDelivered); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	log.Printf("Order %s delivered with code: %s from supplier %s",
		orderID, code, supplierID)

	return tx.Commit()
}

func (s *OrderService) updateOrderStatus(orderID uuid.UUID, status models.OrderStatus) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.orderRepo.UpdateOrderStatusTx(tx, orderID, status); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return tx.Commit()
}
