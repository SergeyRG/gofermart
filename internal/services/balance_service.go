package services

import (
	"context"

	"github.com/SergeyRG/gofermart/internal/model"
)

// Ошибки работы с репозиторием
// var ErrOrderAlreadyExist = errors.New(
// 	"заказ с данным идентификатором уже зарегистрирован в системе")
// var ErrOrderAlreadyAddedByAnotherUser = errors.New(
// 	"заказ с данным идентификатором уже зарегистрирован в системе другим пользователем")
// var ErrUndefinedRepositoryError = errors.New(
// 	"непредвиденная ошибка работы с БД")

type BalanceService interface {
	GetUserBalance(context.Context, model.UserID) (*model.Balance, error)
}

type BalanceRepo interface {
	GetByUserID(context.Context, model.UserID) (*model.Balance, error)
}

type BalanceServiceImpl struct {
	balanceRepo BalanceRepo
}

func NewBalanceService(repo BalanceRepo) BalanceServiceImpl {
	return BalanceServiceImpl{
		balanceRepo: repo,
	}
}

func (svc BalanceServiceImpl) GetUserBalance(ctx context.Context, userID model.UserID) (*model.Balance, error) {
	return svc.balanceRepo.GetByUserID(ctx, userID)
}
