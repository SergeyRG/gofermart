package handlers

import (
	"github.com/SergeyRG/gofermart/internal/services"
)

type StandartHandlers struct {
	orderSvc   services.OrderService
	balanceSvc services.BalanceService
}

func NewStandartHandlers(orderSvc services.OrderService, balanceSvc services.BalanceService) StandartHandlers {
	return StandartHandlers{orderSvc: orderSvc, balanceSvc: balanceSvc}
}
