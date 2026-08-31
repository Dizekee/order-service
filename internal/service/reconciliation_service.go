package service

import (
	"fmt"
	"log"
	"time"

	"github.com/Dizekee/order-service/internal/models"
	"github.com/Dizekee/order-service/internal/repository"
)

type ReconciliationService struct {
	orderRepo    *repository.OrderRepository
	codeRepo     *repository.CodeRepository
	reconRepo    *repository.ReconciliationRepository
	orderService *OrderService
}

func NewReconciliationService(
	orderRepo *repository.OrderRepository,
	codeRepo *repository.CodeRepository,
	reconRepo *repository.ReconciliationRepository,
	orderService *OrderService,
) *ReconciliationService {
	return &ReconciliationService{
		orderRepo:    orderRepo,
		codeRepo:     codeRepo,
		reconRepo:    reconRepo,
		orderService: orderService,
	}
}

func (s *ReconciliationService) RunReconciliation() error {
	log.Println("Starting reconciliation run...")

	paidNotDelivered, err := s.reconRepo.GetOrdersPaidNotDelivered()
	if err != nil {
		return fmt.Errorf("failed to get paid-not-delivered orders: %w", err)
	}

	for _, orderID := range paidNotDelivered {
		log.Printf("Issue: Order %s - paid but not delivered", orderID)
		result := &models.ReconciliationResult{
			OrderID:     orderID,
			IssueType:   "paid_not_delivered",
			Description: "Order is paid but no code was issued",
		}
		if err := s.reconRepo.SaveReconciliationResult(result); err != nil {
			log.Printf("Failed to save reconciliation result: %v", err)
		}
	}

	deliveredNotPaid, err := s.reconRepo.GetOrdersDeliveredNotPaid()
	if err != nil {
		return fmt.Errorf("failed to get delivered-not-paid orders: %w", err)
	}

	for _, orderID := range deliveredNotPaid {
		log.Printf("Issue: Order %s - delivered but not paid", orderID)
		result := &models.ReconciliationResult{
			OrderID:     orderID,
			IssueType:   "delivered_not_paid",
			Description: "Code was issued but no payment was received",
		}
		if err := s.reconRepo.SaveReconciliationResult(result); err != nil {
			log.Printf("Failed to save reconciliation result: %v", err)
		}
	}

	log.Printf("Reconciliation completed. Found %d paid-not-delivered, %d delivered-not-paid",
		len(paidNotDelivered), len(deliveredNotPaid))

	return nil
}

func (s *ReconciliationService) RecoverStuckOrders() error {
	log.Println("Starting recovery of stuck orders...")

	stuckOrders, err := s.reconRepo.GetStuckOrders(5 * time.Minute)
	if err != nil {
		return fmt.Errorf("failed to get stuck orders: %w", err)
	}

	if len(stuckOrders) == 0 {
		log.Println("No stuck orders found")
		return nil
	}

	log.Printf("Found %d stuck orders to recover", len(stuckOrders))

	for _, orderID := range stuckOrders {
		log.Printf("Attempting to recover order %s", orderID)

		if err := s.reconRepo.IncrementRetryCount(orderID); err != nil {
			log.Printf("Failed to increment retry count for order %s: %v", orderID, err)
			continue
		}

		order, err := s.orderRepo.GetOrderByID(orderID)
		if err != nil {
			log.Printf("Failed to get order %s: %v", orderID, err)
			continue
		}

		code, _ := s.codeRepo.GetIssuedCodeByOrderID(orderID)
		if code != "" {
			log.Printf("Order %s already has code, updating status to delivered", orderID)
			if err := s.orderRepo.UpdateOrderStatus(orderID, models.StatusDelivered); err != nil {
				log.Printf("Failed to update status for order %s: %v", orderID, err)
			}
			continue
		}

		if err := s.orderService.deliverProduct(orderID); err != nil {
			log.Printf("Failed to deliver product for order %s: %v", orderID, err)
		} else {
			log.Printf("Successfully recovered order %s", orderID)
		}
	}

	log.Printf("Recovery completed for %d orders", len(stuckOrders))
	return nil
}

func (s *ReconciliationService) StartReconciliationScheduler(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			log.Println("Running scheduled reconciliation...")
			if err := s.RunReconciliation(); err != nil {
				log.Printf("Error in scheduled reconciliation: %v", err)
			}
			if err := s.RecoverStuckOrders(); err != nil {
				log.Printf("Error in stuck orders recovery: %v", err)
			}
		}
	}()
	log.Printf("Reconciliation scheduler started, interval: %v", interval)
}
