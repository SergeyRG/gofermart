package services

import (
	"context"
	"errors"
	"fmt"
	"iter"

	"github.com/SergeyRG/gofermart/internal/model"
)

var ErrIncorrectOperSum = errors.New("некорректная сумма списания")
var ErrInsufficientBalance = errors.New("недостаточно средств на бонусном счете")

//go:generate mockgen -destination=../mocks/mock_balance_service.go -package=mocks . BalanceService
type BalanceService interface {
	AddUserBalance(context.Context, model.UserID) error
	GetUserBalance(context.Context, model.UserID) (*model.Balance, error)
	ExecOper(context.Context, model.Operation) (*model.Balance, error)
	GetUserWithdrawals(ctx context.Context, userID model.UserID) iter.Seq2[*model.Operation, error]
}

//go:generate mockgen -destination=../mocks/mock_balance_repo.go -package=mocks . BalanceRepo
type BalanceRepo interface {
	AddUserBalance(context.Context, model.UserID) error
	GetByUserID(context.Context, model.UserID) (*model.Balance, error)
	ReduceBalance(context.Context, model.UserID, model.MoneyQty) (*model.Balance, error)
	IncreaseBalance(context.Context, model.UserID, model.MoneyQty) (*model.Balance, error)
	AddOperation(context.Context, model.Operation) error
	GetWithdrawalsByUserID(context.Context, model.UserID) iter.Seq2[*model.Operation, error]
}

type BalanceServiceImpl struct {
	balanceRepo BalanceRepo
	TxManager   TransactionManager
}

func NewBalanceService(repo BalanceRepo, txManager TransactionManager) BalanceServiceImpl {
	return BalanceServiceImpl{
		balanceRepo: repo,
		TxManager:   txManager,
	}
}

func (svc BalanceServiceImpl) GetUserBalance(ctx context.Context, userID model.UserID) (*model.Balance, error) {
	return svc.balanceRepo.GetByUserID(ctx, userID)
}

func (svc BalanceServiceImpl) ExecOper(ctx context.Context,
	oper model.Operation) (*model.Balance, error) {

	if oper.Sum <= 0 {
		return nil, fmt.Errorf("%w: %v", ErrIncorrectOperSum, oper.Sum)
	}
	var newBalance *model.Balance

	//Выполнение нескольких действий атомарно
	err := svc.TxManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		var (
			txErr error
			b     *model.Balance
		)
		//Уменьшение или увеличение баланса
		switch oper.OpType {
		case model.OperationWithdraw:
			b, txErr = svc.balanceRepo.ReduceBalance(txCtx, oper.UserID, oper.Sum)
		case model.OperationDeposit:
			b, txErr = svc.balanceRepo.IncreaseBalance(txCtx, oper.UserID, oper.Sum)
		default:
			return fmt.Errorf("неожиданный тип операции: %s", oper.OpType)
		}

		if txErr != nil {
			return txErr
		}

		//Добавление записи об операции в журнал
		txErr = svc.balanceRepo.AddOperation(txCtx, oper)
		if txErr != nil {
			return fmt.Errorf("Ошибка сохранения информации об операции в БД: %w", txErr)
		}
		newBalance = b
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("Ошибка выполнения транзакции в БД: %w", err)
	}

	return newBalance, nil
}

func (svc BalanceServiceImpl) GetUserWithdrawals(ctx context.Context, userID model.UserID) iter.Seq2[*model.Operation, error] {
	return svc.balanceRepo.GetWithdrawalsByUserID(ctx, userID)
}

func (svc BalanceServiceImpl) AddUserBalance(ctx context.Context, uID model.UserID) error {
	return svc.balanceRepo.AddUserBalance(ctx, uID)
}
