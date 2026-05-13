package services

import (
	"context"
	"errors"

	"github.com/SergeyRG/gofermart/internal/model"
)

// Ошибки работы с репозиторием
var ErrOrderAlreadyExist = errors.New(
	"заказ с данным идентификатором уже зарегистрирован в системе")
var ErrOrderAlreadyAddedByAnotherUser = errors.New(
	"заказ с данным идентификатором уже зарегистрирован в системе другим пользователем")
var ErrUndefinedRepositoryError = errors.New(
	"непредвиденная ошибка работы с БД")

type OrderService interface {
	AddOrder(context.Context, model.UserID, model.OrderID) error
	GetUserOrders(context.Context, model.UserID) ([]model.Order, error)
}

type OrderRepo interface {
	GetByUserID(context.Context, model.UserID) ([]model.Order, error)
	Add(context.Context, model.Order) error
}

type AccrualClient interface {
	GetAccrual(ctx context.Context, orderID string)
}

type OrderServiceImpl struct {
	orderRepo OrderRepo
}

func (svc OrderServiceImpl) AddOrder(ctx context.Context, userID model.UserID, orderID model.OrderID) error {
	order := model.NewOrder(orderID, userID)

	return svc.orderRepo.Add(ctx, order)
}

func (svc OrderServiceImpl) GetUserOrders(ctx context.Context, userID model.UserID) ([]model.Order, error) {
	return svc.orderRepo.GetByUserID(ctx, userID)
}

func NewOrderService(repo OrderRepo) OrderServiceImpl {
	return OrderServiceImpl{
		orderRepo: repo,
	}
}
