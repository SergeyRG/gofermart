package services

import (
	"context"

	"github.com/SergeyRG/gofermart/internal/model"
)

type OrderService interface {
	Add(context.Context, model.UserID, model.OrderID) error
	Get(context.Context, model.UserID) ([]model.Order, error)
}

type OrderServiceImpl struct{}

func (svc OrderServiceImpl) Add(context.Context, model.UserID, model.OrderID) error {
	return nil
}

func (svc OrderServiceImpl) Get(context.Context, model.UserID) ([]model.Order, error) {
	return make([]model.Order, 0), nil
}
