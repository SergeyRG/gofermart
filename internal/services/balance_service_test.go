package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SergeyRG/gofermart/internal/mocks"
	"github.com/SergeyRG/gofermart/internal/model"
	"github.com/SergeyRG/gofermart/internal/services"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type StubTxManager struct{}

func (m StubTxManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func TestBalanceServiceImpl_ExecOper(t *testing.T) {
	withdrawalTestOper := model.Operation{
		UserID:      1,
		OrderID:     "1",
		Sum:         100,
		OpType:      model.OperationWithdraw,
		ProcessedAt: time.Time{},
	}
	depositTestOper := model.Operation{
		UserID:      1,
		OrderID:     "1",
		Sum:         100,
		OpType:      model.OperationDeposit,
		ProcessedAt: time.Time{},
	}
	testBalance := model.Balance{
		UserID:    1,
		Current:   100,
		Withdrawn: 1000,
	}

	tests := []struct {
		name      string
		setupRepo func(m *mocks.MockBalanceRepo)
		oper      model.Operation
		want      *model.Balance
		wantErr   bool
		err       error
	}{
		{
			name:      "Если сумма операции меньше 0, то возвращается ошибка ErrIncorrectOperSum",
			setupRepo: func(m *mocks.MockBalanceRepo) {},
			oper: model.Operation{
				UserID:      1,
				OrderID:     "1",
				Sum:         -1,
				OpType:      model.OperationWithdraw,
				ProcessedAt: time.Time{},
			},
			want:    nil,
			wantErr: true,
			err:     services.ErrIncorrectOperSum,
		},
		{
			name: "Тест механики операции списания",
			setupRepo: func(m *mocks.MockBalanceRepo) {
				m.EXPECT().
					ReduceBalance(gomock.Any(), withdrawalTestOper.UserID, withdrawalTestOper.Sum).
					Return(&testBalance, nil)
				m.EXPECT().AddOperation(gomock.Any(), withdrawalTestOper).
					Return(nil)
			},
			oper:    withdrawalTestOper,
			want:    &testBalance,
			wantErr: false,
			err:     nil,
		},
		{
			name: "Тест механики операции зачисления",
			setupRepo: func(m *mocks.MockBalanceRepo) {
				m.EXPECT().
					IncreaseBalance(gomock.Any(), depositTestOper.UserID, depositTestOper.Sum).
					Return(&testBalance, nil)
				m.EXPECT().AddOperation(gomock.Any(), depositTestOper).
					Return(nil)
			},
			oper:    depositTestOper,
			want:    &testBalance,
			wantErr: false,
			err:     nil,
		},
		{
			name: "Если происходит ошибка транзакции БД, то возвращается ошибка",
			setupRepo: func(m *mocks.MockBalanceRepo) {
				m.EXPECT().
					IncreaseBalance(gomock.Any(), depositTestOper.UserID, depositTestOper.Sum).
					Return(nil, errors.New("test"))
			},
			oper:    depositTestOper,
			want:    nil,
			wantErr: true,
			err:     nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			repo := mocks.NewMockBalanceRepo(ctrl)
			tt.setupRepo(repo)

			txm := StubTxManager{}

			svc := services.NewBalanceService(repo, txm)

			got, gotErr := svc.ExecOper(context.Background(), tt.oper)

			if tt.wantErr {
				assert.NotNil(t, gotErr)
			}
			if tt.err != nil {
				assert.ErrorIs(t, gotErr, tt.err)
			}
			assert.Equal(t, tt.want, got)

		})
	}
}
