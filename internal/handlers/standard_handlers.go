package handlers

import (
	"github.com/SergeyRG/gofermart/internal/services"
)

type StandardHandlers struct {
	orderSvc   services.OrderService
	balanceSvc services.BalanceService
}

func NewStandardHandlers(orderSvc services.OrderService, balanceSvc services.BalanceService) StandardHandlers {
	return StandardHandlers{orderSvc: orderSvc, balanceSvc: balanceSvc}
}
