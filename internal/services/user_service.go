package services

import (
	"context"
	"errors"

	"github.com/SergeyRG/gofermart/internal/auth"
	"github.com/SergeyRG/gofermart/internal/model"
)

var (
	ErrLoginBusy            = errors.New("данный логин уже занят")
	ErrWrongLoginOrPassword = errors.New("неверная пара логин/пароль;")
)

//go:generate mockgen -destination=../mocks/mock_user_service.go -package=mocks . UserService
type UserService interface {
	GetUserByLogin(ctx context.Context, login string) (*model.User, error)
	AddUser(ctx context.Context, login string, password string) (model.UserID, error)
	AuthUser(ctx context.Context, login string, pwdHash string) (model.UserID, error)
}

//go:generate mockgen -destination=../mocks/mock_user_repo.go -package=mocks . UserRepo
type UserRepo interface {
	GetUserByLogin(ctx context.Context, login string) (*model.User, error)
	AddUser(ctx context.Context, user model.User) (model.UserID, error)
}

type UserServiceImpl struct {
	repo           UserRepo
	balanceService BalanceService
	txm            TransactionManager
}

func NewUserService(repo UserRepo, txm TransactionManager, bSvc BalanceService) *UserServiceImpl {
	return &UserServiceImpl{repo: repo, txm: txm, balanceService: bSvc}
}

func (svc UserServiceImpl) GetUserByLogin(ctx context.Context, login string) (*model.User, error) {
	return svc.repo.GetUserByLogin(ctx, login)
}

func (svc UserServiceImpl) AuthUser(ctx context.Context, login string, pwd string) (model.UserID, error) {
	user, err := svc.GetUserByLogin(ctx, login)
	if err != nil {
		return 0, err
	}

	if !auth.CheckPasswordHash(pwd, user.PwdHash) {
		return 0, ErrWrongLoginOrPassword
	}

	return user.UserID, nil
}

func (svc UserServiceImpl) AddUser(ctx context.Context, login string, password string) (model.UserID, error) {
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return 0, err
	}
	uID := model.UserID(0)
	err = svc.txm.WithinTransaction(ctx, func(txCtx context.Context) error {
		id, err := svc.repo.AddUser(txCtx, model.User{Login: login, PwdHash: passwordHash})
		if err != nil {
			return err
		}
		err = svc.balanceService.AddUserBalance(txCtx, id)
		if err != nil {
			return err
		}
		uID = id
		return nil
	})
	return uID, err
}
