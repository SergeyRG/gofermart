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
var ErrRecoverableError = errors.New("ошибка, которая может не возникнуть при повторной попытке")
var ErrUnRecoverableError = errors.New("ошибка, повторная папытка не требуется")

type ErrTooManyRequests struct {
	FreezeDuration int
}

func (e *ErrTooManyRequests) Is(target error) bool {
	_, ok := target.(*ErrTooManyRequests)
	return ok
}

func (e ErrTooManyRequests) Error() string {
	return "слишком много запросов к сервису"
}

type AccrualSystemOrderInfo struct {
	Order   model.OrderID       `json:"order"`
	Status  AccrualSystemStatus `json:"status"`
	Accrual model.MoneyQty      `json:"accrual"`
}

type AccrualSystemStatus string

const (
	AccrualSystemStatusRegistered = "REGISTERED"
	AccrualSystemStatusInvalid    = "INVALID"
	AccrualSystemStatusProcessing = "PROCESSING"
	AccrualSystemStatusProcessed  = "PROCESSED"
)

//go:generate mockgen -destination=../mocks/mock_accrual_service.go -package=mocks . AccrualService
type AccrualService interface {
	SucceedOrder(context.Context, model.OrderID) error
	ProcessOrder(context.Context, model.OrderID) error
	ChangeProcessingOrderStatus(context.Context, model.OrderID, model.OrderStatus) error
}

//go:generate mockgen -destination=../mocks/mock_accrual_client.go -package=mocks . AccrualClient
type AccrualClient interface {
	RequestAccrualSystemOrderStatus(ctx context.Context, orderID model.OrderID) (*AccrualSystemOrderInfo, error)
}

type AccrualServiceImpl struct {
	Txm            TransactionManager
	OrderService   OrderService
	BalanceService BalanceService
	AccrualClient  AccrualClient
	FreezeManager  FreezeManager
	WorkersCount   int
}

func NewAccrualService(txm TransactionManager, os OrderService, bs BalanceService, ac AccrualClient, wc int) *AccrualServiceImpl {
	return &AccrualServiceImpl{Txm: txm, OrderService: os, BalanceService: bs, AccrualClient: ac, WorkersCount: wc}
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

func (as *AccrualServiceImpl) handleAccrualSystemResponse(
	ctx context.Context,
	oi AccrualSystemOrderInfo) error {

	var targetStatus model.OrderStatus

	switch oi.Status {
	case AccrualSystemStatusRegistered, AccrualSystemStatusProcessing:
		targetStatus = model.StatusProcessing
	case AccrualSystemStatusInvalid:
		targetStatus = model.StatusInvalid
	case AccrualSystemStatusProcessed:
		txErr := as.Txm.WithinTransaction(ctx, func(txCtx context.Context) error {
			return as.succeedOrder(txCtx, oi)
		})
		if txErr != nil {
			logging.Logger.Error("ошибка БД выполнения начисления", zap.Error(txErr))
			return fmt.Errorf("%w: %v", ErrRecoverableError, txErr)
		}
		return nil
	default:
		err := fmt.Errorf("неизвестный статус ответа: %s", oi.Status)
		return fmt.Errorf("%w: %v", ErrUnRecoverableError, err)
	}

	txErr := as.Txm.WithinTransaction(ctx, func(txCtx context.Context) error {
		_, err := as.ChangeProcessingOrderStatus(txCtx, oi.Order, targetStatus)
		return err
	})
	if txErr != nil {
		logging.Logger.Error("ошибка обновления статуса в БД", zap.Error(txErr), zap.String("status", string(targetStatus)))
		return fmt.Errorf("%w: %v", ErrRecoverableError, txErr)
	}
	return nil
}

func (as *AccrualServiceImpl) handleAccrualSystemError(
	ctx context.Context,
	err error, oID model.OrderID) error {

	var errMaxRequests *ErrTooManyRequests
	if errors.As(err, &errMaxRequests) {
		txErr := as.Txm.WithinTransaction(ctx, func(txCtx context.Context) error {
			_, err := as.ChangeProcessingOrderStatus(txCtx, oID, model.StatusNew)
			return err
		})
		//Данную ошибку просто логируем, попытка обработки заказа повторно
		//будет выполнена с задержкой
		if txErr != nil {
			logging.Logger.Error("ошибка возвращения заказу статуса NEW",
				zap.Error(txErr))
		}
		return errMaxRequests
	}

	if errors.Is(err, ErrOrderNotRegistered) {
		txErr := as.Txm.WithinTransaction(ctx, func(txCtx context.Context) error {
			order, err := as.OrderService.GetByIDForUpdate(txCtx, oID)
			if err != nil {
				return err
			}
			// Если информация больше часа не загружается в систему расчета
			// то заказу присвоить статус INVALID
			if order.AddedAt.Add(time.Hour).Before(time.Now()) {
				_, err = as.ChangeProcessingOrderStatus(txCtx, oID, model.StatusInvalid)
				if err != nil {
					return err
				}
			} else {
				// Чтобы обновить поле updated_at
				_, err = as.ChangeProcessingOrderStatus(txCtx, oID, model.StatusProcessing)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if txErr != nil {
			return fmt.Errorf("%w:%v", ErrRecoverableError, txErr)
		} else {
			return nil
		}
	}
	logging.Logger.Error("непредвиденная ошибка при запросе к системе начислений", zap.Error(err))
	return fmt.Errorf("%w:%v", ErrRecoverableError, err)
}

func (as *AccrualServiceImpl) ProcessOrder(
	ctx context.Context,
	oID model.OrderID,
) error {

	resp, err := as.AccrualClient.RequestAccrualSystemOrderStatus(ctx, oID)

	if err != nil {
		logging.Logger.Debug("ошибка получения информации от системы начислений",
			zap.Error(err))
		return as.handleAccrualSystemError(ctx, err, oID)
	}
	logging.Logger.Debug("получен ответ системы начисления",
		zap.String("order ID", string(resp.Order)),
		zap.String("status", string(resp.Status)))

	err = as.handleAccrualSystemResponse(ctx, *resp)
	return err
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

func (as *AccrualServiceImpl) workerSleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

func (as *AccrualServiceImpl) RunWorker(ctx context.Context, id int) error {
	logging.Logger.Debug("запуск воркера", zap.Int("Worker ID", id))
	for {
		if ctx.Err() != nil {
			return nil // Мгновенно выходим, если контекст закрыт
		}

		if as.FreezeManager.isFrozen() {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(as.FreezeManager.DurationRemaining()):
			}
			continue
		}

		order, err := as.OrderService.GetNextOrderIDForProcessing(ctx)
		if err != nil {
			if errors.Is(err, ErrNoOrdersForProcessing) {
				logging.Logger.Debug(
					"нет заказов для обработки, засыпаем на 10 секунд")
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(time.Second * 10):
				}
				continue
			} else {
				logging.Logger.Error(
					"ошибка получения очередного заказа для обработки", zap.Error(err))
				as.workerSleep(ctx, time.Second*2)
				continue
			}
		}

		logging.Logger.Debug(
			"воркер обрабатывает заказ", zap.Int("Worker ID", id), zap.String("Order ID", string(order.ID)))

		maxAttempts := 3
		baseSleep := 1 * time.Second

		for attempt := range maxAttempts {
			err = as.ProcessOrder(ctx, order.ID)
			if err == nil || errors.Is(err, ErrUnRecoverableError) {
				break
			}

			var errMaxRequests *ErrTooManyRequests
			if errors.As(err, &errMaxRequests) {
				as.FreezeManager.Freeze(errMaxRequests.FreezeDuration)
				break
			}
			if errors.Is(err, ErrRecoverableError) {
				as.workerSleep(ctx, baseSleep*time.Duration(attempt+1))
				continue
			}
		}
	}
}
