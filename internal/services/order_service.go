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
var ErrNotFound = errors.New("заказ не найден в БД")
var ErrNoOrdersForProcessing = errors.New("нет заказов для обработки")

type OrderService interface {
	AddOrder(context.Context, model.UserID, model.OrderID) error
	GetUserOrders(context.Context, model.UserID) ([]model.Order, error)
	ChangeOrderStatus(context.Context, model.OrderID, model.OrderStatus) error
	GetByIDForUpdate(ctx context.Context, oID model.OrderID) (*model.Order, error)
	GetNextOrderIDForProcessing(ctx context.Context) (*model.Order, error)
}

type OrderRepo interface {
	GetByUserID(context.Context, model.UserID) ([]model.Order, error)
	GetByIDForUpdate(context.Context, model.OrderID) (*model.Order, error)
	Add(context.Context, model.Order) error
	//	GetOrdersForProccessing(context.Context) ([]model.OrderID, error)
	ChangeOrderStatus(context.Context, model.OrderID, model.OrderStatus) error
	GetNextOrderIDForProcessing(ctx context.Context) (*model.Order, error)
}

type OrderServiceImpl struct {
	orderRepo    OrderRepo
	TxManager    TransactionManager
	WorkersCount int
}

func (svc OrderServiceImpl) AddOrder(ctx context.Context, userID model.UserID, orderID model.OrderID) error {
	order := model.NewOrder(orderID, userID)

	return svc.orderRepo.Add(ctx, order)
}

func (svc OrderServiceImpl) GetUserOrders(ctx context.Context, userID model.UserID) ([]model.Order, error) {
	return svc.orderRepo.GetByUserID(ctx, userID)
}

func (svc OrderServiceImpl) ChangeOrderStatus(ctx context.Context, oID model.OrderID, oStatus model.OrderStatus) error {
	return svc.orderRepo.ChangeOrderStatus(ctx, oID, oStatus)
}

func (svc OrderServiceImpl) GetByIDForUpdate(ctx context.Context, oID model.OrderID) (*model.Order, error) {
	return svc.orderRepo.GetByIDForUpdate(ctx, oID)
}

func (svc OrderServiceImpl) GetNextOrderIDForProcessing(ctx context.Context) (*model.Order, error) {
	return svc.orderRepo.GetNextOrderIDForProcessing(ctx)
}

// func (svc OrderServiceImpl) RunOrderExtractor(ctx context.Context, ordersChan chan<- model.OrderID) error {
// 	tk := time.NewTicker(time.Minute)
// 	defer tk.Stop()
// 	logging.Logger.Info("запуск выгрузки заказов")
// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return nil
// 		case <-tk.C:
// 			logging.Logger.Info("выгрузка заказов по таймеру")
// 			orders, err := svc.orderRepo.GetOrdersForProccessing(ctx)
// 			if err != nil {
// 				logging.Logger.Error("Ошибка выгрузки заказов для обработки", zap.Error(err))
// 				return err
// 			}

// 			for _, o := range orders {
// 				select {
// 				case ordersChan <- o:
// 					logging.Logger.Debug(
// 						"постановка в очередь заказа", zap.String("номер", string(o)))
// 				case <-ctx.Done():
// 					close(ordersChan)
// 					return nil
// 				}
// 			}
// 		}
// 	}
// }

func NewOrderService(repo OrderRepo, txManager TransactionManager, workersCnt int) OrderServiceImpl {
	return OrderServiceImpl{
		orderRepo:    repo,
		TxManager:    txManager,
		WorkersCount: workersCnt,
	}
}
