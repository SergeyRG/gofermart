package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SergeyRG/gofermart/internal/logging"
	"github.com/SergeyRG/gofermart/internal/model"
	"go.uber.org/zap"
)

var ErrOrderNotRegistered = errors.New("заказ не зарегистрирован в системе расчета начислений")
var ErrTooManyRequests = errors.New("слишком много запросов к сервису")

type AccrualSystemOrderInfo struct {
	Order   model.OrderID       `json:"order"`
	Status  AccrualSystemStatus `json:"status"`
	Accrual model.MoneyQty      `json:"accrual"`
}

type AccrualSystemAnswer struct {
	OrderInfo     *AccrualSystemOrderInfo
	FreezeSeconds int `json:"-"`
}

type AccrualSystemStatus string

const (
	AccrualSystemStatusRegistered = "REGISTERED"
	AccrualSystemStatusInvalid    = "INVALID"
	AccrualSystemStatusProcessing = "PROCESSING"
	AccrualSystemStatusProcessed  = "PROCESSED"
)

type AccrualService interface {
	SucceedOrder(context.Context, model.OrderID) error
	ProcessOrder(context.Context, model.OrderID) error
	ChangeProcessingOrderStatus(context.Context, model.OrderID, model.OrderStatus) error
}

type AccrualClient interface {
	RequestAccrualSystemOrderStatus(ctx context.Context, orderID model.OrderID) (*AccrualSystemAnswer, error)
}

type AccrualServiceImpl struct {
	Txm            TransactionManager
	OrderService   OrderService
	BalanceService BalanceService
	AccrualClient  AccrualClient
	FreezeManager  FreezeManager
}

func NewAccrualService(txm TransactionManager, os OrderService, bs BalanceService, ac AccrualClient) *AccrualServiceImpl {
	return &AccrualServiceImpl{Txm: txm, OrderService: os, BalanceService: bs, AccrualClient: ac}
}

func (as *AccrualServiceImpl) ChangeProcessingOrderStatus(
	ctx context.Context,
	oID model.OrderID,
	status model.OrderStatus) (*model.Order, error) {

	order, err := as.OrderService.GetByIDForUpdate(ctx, oID)
	if err != nil {
		return nil, err
	}

	if order.Status != model.StatusProcessing {
		return nil, fmt.Errorf("у заказа статус отличный от PROCESSING: %v", order.Status)
	}

	err = as.OrderService.ChangeOrderStatus(ctx, order.ID, status)
	if err != nil {
		return nil, err
	}

	return order, nil
}

func (as *AccrualServiceImpl) ChangeProcessingOrder(
	ctx context.Context,
	oInfo AccrualSystemOrderInfo,
	status model.OrderStatus) (*model.Order, error) {

	order, err := as.OrderService.GetByIDForUpdate(ctx, oInfo.Order)
	if err != nil {
		return nil, err
	}

	if order.Status != model.StatusProcessing {
		return nil, fmt.Errorf("у заказа статус отличный от PROCESSING: %v", order.Status)
	}

	order.Status = status
	order.Accrual = oInfo.Accrual

	err = as.OrderService.UpdateOrder(ctx, *order)
	if err != nil {
		return nil, err
	}

	return order, nil
}

func (as *AccrualServiceImpl) ProcessOrder(
	ctx context.Context,
	oID model.OrderID,
) {
	answer, err := as.AccrualClient.RequestAccrualSystemOrderStatus(ctx, oID)
	if err != nil {
		logging.Logger.Debug("ошибка получения информации от системы начисления",
			zap.Error(err))

		if errors.Is(err, ErrTooManyRequests) {
			as.FreezeManager.Freeze(answer.FreezeSeconds)
			as.Txm.WithinTransaction(ctx, func(txCtx context.Context) error {
				_, err := as.ChangeProcessingOrderStatus(txCtx, oID, model.StatusNew)
				return err
			})
			return
		}
		if errors.Is(err, ErrOrderNotRegistered) {
			as.Txm.WithinTransaction(ctx, func(txCtx context.Context) error {
				order, err := as.OrderService.GetByIDForUpdate(txCtx, oID)
				if err != nil {
					return err
				}
				// Если информация больше часа не загружается в систему расчета
				// то заказу присвоить статус INVALID
				if order.AddedAt.Add(time.Hour).Before(time.Now()) {
					_, err := as.ChangeProcessingOrderStatus(txCtx, oID, model.StatusInvalid)
					if err != nil {
						return err
					}
				} else {
					// Чтобы обновить поле updated_at
					as.ChangeProcessingOrderStatus(txCtx, oID, model.StatusProcessing)
				}
				return nil
			})
			return
		}
		logging.Logger.Error("непредвиденная ошибка при запросе к системе начислений", zap.Error(err))
		return
	}
	logging.Logger.Debug("получен ответ системы начисления",
		zap.String("order ID", string(answer.OrderInfo.Order)),
		zap.String("status", string(answer.OrderInfo.Status)))

	if answer.OrderInfo.Status == AccrualSystemStatusRegistered ||
		answer.OrderInfo.Status == AccrualSystemStatusProcessing {
		as.Txm.WithinTransaction(ctx, func(txCtx context.Context) error {
			_, err := as.ChangeProcessingOrderStatus(txCtx, oID, model.StatusProcessing)
			return err
		})
		return
	}

	if answer.OrderInfo.Status == AccrualSystemStatusProcessed {
		err := as.Txm.WithinTransaction(ctx, func(txCtx context.Context) error {
			return as.succeedOrder(txCtx, *answer.OrderInfo)
		})
		if err != nil {
			logging.Logger.Error("ошибка БД выполнения начисления", zap.Error(err))
		}
		return
	}

	if answer.OrderInfo.Status == AccrualSystemStatusInvalid {
		as.Txm.WithinTransaction(ctx, func(txCtx context.Context) error {
			_, err := as.ChangeProcessingOrderStatus(txCtx, oID, model.StatusInvalid)
			return err
		})
		return
	}
}

func (as *AccrualServiceImpl) succeedOrder(ctx context.Context, oInfo AccrualSystemOrderInfo) error {
	order, err := as.ChangeProcessingOrder(ctx, oInfo, model.StatusProcessed)
	if err != nil {
		return err
	}

	oper := model.Operation{
		UserID:      order.UserID,
		OrderID:     order.ID,
		Sum:         oInfo.Accrual,
		OpType:      model.OperationDeposit,
		ProcessedAt: time.Now().UTC(),
	}

	_, err = as.BalanceService.ExecOper(ctx, oper)
	if err != nil {
		return err
	}
	return nil
}

func (as *AccrualServiceImpl) RunWorker(ctx context.Context, id int) error {
	logging.Logger.Debug("запуск воркера", zap.Int("Worker ID", id))
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			for as.FreezeManager.isFrozen() {
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(as.FreezeManager.DurationRemaining()):
				}
			}

			order, err := as.OrderService.GetNextOrderIDForProcessing(ctx)
			if err != nil {
				if errors.Is(err, ErrNoOrdersForProcessing) {
					logging.Logger.Debug(
						"нет заказов для обработки, засыпаем на секунду")
				} else {
					logging.Logger.Error(
						"ошибка получения очередного заказа для обработки", zap.Error(err))
				}

				select {
				case <-ctx.Done():
					return nil
				case <-time.After(time.Second * 10):
				}
				continue
			}
			logging.Logger.Debug(
				"воркер обрабатывает заказ", zap.Int("Worker ID", id), zap.String("Order ID", string(order.ID)))
			as.ProcessOrder(ctx, order.ID)
		}
	}
}
