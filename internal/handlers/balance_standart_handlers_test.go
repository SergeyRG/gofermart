package handlers_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SergeyRG/gofermart/internal/auth"
	"github.com/SergeyRG/gofermart/internal/handlers"
	"github.com/SergeyRG/gofermart/internal/mocks"
	"github.com/SergeyRG/gofermart/internal/model"
	"github.com/SergeyRG/gofermart/internal/services"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type headerWant struct {
	header   string
	contains string
}

func TestStandartHandlers_GetUserBalance(t *testing.T) {
	endpoint := "/api/user/balance"
	positiveCaseBalance := &model.Balance{
		UserID:    1,
		Current:   200,
		Withdrawn: 300,
	}
	expectedJSON := `{"current":2, "withdrawn":3}`
	tests := []struct {
		name                string
		ctx                 context.Context
		setupOrderSvcMock   func(m *mocks.MockOrderService)
		setupBalanceSvcMock func(m *mocks.MockBalanceService)
		wantStatus          int
		wantHeader          *headerWant
		wantBody            string
	}{
		{
			name:                "Вызов хэндлера без передачи userID в контексте приводит к ошибке http.StatusUnauthorized",
			ctx:                 context.Background(),
			setupOrderSvcMock:   func(m *mocks.MockOrderService) {},
			setupBalanceSvcMock: func(m *mocks.MockBalanceService) {},
			wantStatus:          http.StatusUnauthorized,
			wantHeader:          nil,
			wantBody:            "",
		},
		{
			name:              "Если balanceSvc возвращает ошибку, то хэндлер возвращает http.StatusInternalServerError",
			ctx:               auth.ContextWithUserID(context.Background(), model.UserID(1)),
			setupOrderSvcMock: func(m *mocks.MockOrderService) {},
			setupBalanceSvcMock: func(m *mocks.MockBalanceService) {
				m.EXPECT().GetUserBalance(gomock.Any(), model.UserID(1)).
					Return(nil, errors.New("test")).
					Times(1)
			},
			wantStatus: http.StatusInternalServerError,
			wantHeader: nil,
			wantBody:   "",
		},
		{
			name:              "Хэндлер возвращает balance в виде json",
			ctx:               auth.ContextWithUserID(context.Background(), model.UserID(1)),
			setupOrderSvcMock: func(m *mocks.MockOrderService) {},
			setupBalanceSvcMock: func(m *mocks.MockBalanceService) {

				m.EXPECT().GetUserBalance(gomock.Any(), model.UserID(1)).
					Return(positiveCaseBalance, nil).
					Times(1)
			},
			wantStatus: http.StatusOK,
			wantHeader: &headerWant{header: "Content-Type", contains: "application/json"},
			wantBody:   expectedJSON,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			orderSvcMock := mocks.NewMockOrderService(ctrl)
			tt.setupOrderSvcMock(orderSvcMock)
			balanceSvcMock := mocks.NewMockBalanceService(ctrl)
			tt.setupBalanceSvcMock(balanceSvcMock)
			h := handlers.NewStandartHandlers(orderSvcMock, balanceSvcMock)

			req := httptest.NewRequest(http.MethodGet, endpoint, nil)
			if tt.ctx != nil {
				req = req.WithContext(tt.ctx)
			}
			rw := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Get(endpoint, h.GetUserBalance())
			router.ServeHTTP(rw, req)

			res := rw.Result()
			defer res.Body.Close()

			bodyBytes, err := io.ReadAll(res.Body)
			assert.NoError(t, err)

			if tt.wantStatus != 0 {
				assert.Equal(t, tt.wantStatus, res.StatusCode)
			}

			if tt.wantHeader != nil {
				assert.Contains(t, res.Header.Get(tt.wantHeader.header), tt.wantHeader.contains)
			}

			if tt.wantBody != "" {
				assert.JSONEq(t, tt.wantBody, string(bodyBytes))
			} else {
				assert.Equal(t, tt.wantBody, string(bodyBytes))
			}

		})
	}
}

func TestStandartHandlers_Withdraw(t *testing.T) {
	endpoint := "/api/user/balance/withdraw"
	tests := []struct {
		name                string
		ctx                 context.Context
		setupOrderSvcMock   func(m *mocks.MockOrderService)
		setupBalanceSvcMock func(m *mocks.MockBalanceService)
		setupRequest        func(r *http.Request)
		wantStatus          int
		wantHeader          *headerWant
		wantBody            string
	}{
		{
			name:                "Вызов хэндлера без передачи userID в контексте приводит к ошибке http.StatusUnauthorized",
			ctx:                 context.Background(),
			setupOrderSvcMock:   func(m *mocks.MockOrderService) {},
			setupBalanceSvcMock: func(m *mocks.MockBalanceService) {},
			setupRequest:        func(r *http.Request) {},
			wantStatus:          http.StatusUnauthorized,
			wantHeader:          nil,
			wantBody:            "",
		},
		{
			name:                "Если запрос не содержит header application.json, то возвращается http.StatusBadRequest",
			ctx:                 auth.ContextWithUserID(context.Background(), model.UserID(1)),
			setupOrderSvcMock:   func(m *mocks.MockOrderService) {},
			setupBalanceSvcMock: func(m *mocks.MockBalanceService) {},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Content-Type", "application/json")
				jsonBody := `test`
				r.Body = io.NopCloser(strings.NewReader(jsonBody))
			},
			wantStatus: http.StatusBadRequest,
			wantHeader: nil,
			wantBody:   "",
		},
		{
			name:                "Если тело запроса содержит некорректный json, то возвращается http.StatusBadRequest",
			ctx:                 auth.ContextWithUserID(context.Background(), model.UserID(1)),
			setupOrderSvcMock:   func(m *mocks.MockOrderService) {},
			setupBalanceSvcMock: func(m *mocks.MockBalanceService) {},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Content-Type", "application/json")
				jsonBody := `test`
				r.Body = io.NopCloser(strings.NewReader(jsonBody))
			},
			wantStatus: http.StatusBadRequest,
			wantHeader: nil,
			wantBody:   "",
		},
		{
			name:              "Если balanceSvc возвращает services.ErrInsufficientBalance, то хэндлер возвращает http.StatusPaymentRequired",
			ctx:               auth.ContextWithUserID(context.Background(), model.UserID(1)),
			setupOrderSvcMock: func(m *mocks.MockOrderService) {},
			setupBalanceSvcMock: func(m *mocks.MockBalanceService) {
				m.EXPECT().ExecOper(gomock.Any(), gomock.Any()).
					Return(nil, services.ErrInsufficientBalance).
					Times(1)
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Content-Type", "application/json")
				jsonBody := `{"order": "12345674", "sum": 100}`
				r.Body = io.NopCloser(strings.NewReader(jsonBody))
			},
			wantStatus: http.StatusPaymentRequired,
			wantHeader: nil,
			wantBody:   "",
		},
		{
			name:              "Тест позитивного сценария",
			ctx:               auth.ContextWithUserID(context.Background(), model.UserID(1)),
			setupOrderSvcMock: func(m *mocks.MockOrderService) {},
			setupBalanceSvcMock: func(m *mocks.MockBalanceService) {
				m.EXPECT().
					ExecOper(gomock.Any(), gomock.Cond(func(x any) bool {
						oper, ok := x.(model.Operation)
						if !ok {
							return false
						}
						if oper.OrderID != "12345674" || oper.Sum != 10000 {
							return false
						}
						return true
					})).
					Return(nil, nil).
					Times(1)
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Content-Type", "application/json")
				jsonBody := `{"order": "12345674", "sum": 100}`
				r.Body = io.NopCloser(strings.NewReader(jsonBody))
			},
			wantStatus: http.StatusOK,
			wantHeader: nil,
			wantBody:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			orderSvcMock := mocks.NewMockOrderService(ctrl)
			tt.setupOrderSvcMock(orderSvcMock)
			balanceSvcMock := mocks.NewMockBalanceService(ctrl)
			tt.setupBalanceSvcMock(balanceSvcMock)
			h := handlers.NewStandartHandlers(orderSvcMock, balanceSvcMock)

			req := httptest.NewRequest(http.MethodPost, endpoint, nil)
			if tt.ctx != nil {
				req = req.WithContext(tt.ctx)
			}
			tt.setupRequest(req)
			rw := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Post(endpoint, h.Withdraw())
			router.ServeHTTP(rw, req)

			res := rw.Result()
			defer res.Body.Close()

			bodyBytes, err := io.ReadAll(res.Body)
			assert.NoError(t, err)

			if tt.wantStatus != 0 {
				assert.Equal(t, tt.wantStatus, res.StatusCode)
			}

			if tt.wantHeader != nil {
				assert.Contains(t, res.Header.Get(tt.wantHeader.header), tt.wantHeader.contains)
			}

			if tt.wantBody != "" {
				assert.JSONEq(t, tt.wantBody, string(bodyBytes))
			} else {
				assert.Equal(t, tt.wantBody, string(bodyBytes))
			}

		})
	}
}

func TestStandartHandlers_GetWithdrawals(t *testing.T) {
	endpoint := "/api/user/withdrawals"
	tests := []struct {
		name                string
		ctx                 context.Context
		setupOrderSvcMock   func(m *mocks.MockOrderService)
		setupBalanceSvcMock func(m *mocks.MockBalanceService)
		setupRequest        func(r *http.Request)
		wantStatus          int
		wantHeader          *headerWant
		wantBody            string
	}{
		{
			name:                "Вызов хэндлера без передачи userID в контексте приводит к ошибке http.StatusUnauthorized",
			ctx:                 context.Background(),
			setupOrderSvcMock:   func(m *mocks.MockOrderService) {},
			setupBalanceSvcMock: func(m *mocks.MockBalanceService) {},
			setupRequest:        func(r *http.Request) {},
			wantStatus:          http.StatusUnauthorized,
			wantHeader:          nil,
			wantBody:            "",
		},
		{
			name:              "Если balanceSvc возвращает ошибку, то хэндлер возвращает http.StatusInternalServerError",
			ctx:               auth.ContextWithUserID(context.Background(), model.UserID(1)),
			setupOrderSvcMock: func(m *mocks.MockOrderService) {},
			setupBalanceSvcMock: func(m *mocks.MockBalanceService) {
				m.EXPECT().
					GetUserWithdrawals(gomock.Any(), model.UserID(1)).
					Times(1).Return(nil, errors.New("тест"))
			},
			setupRequest: func(r *http.Request) {},
			wantStatus:   http.StatusInternalServerError,
			wantHeader:   nil,
			wantBody:     "",
		},
		{
			name:              "Если balanceSvc возвращает пустой список, то хэндлер возвращает http.StatusNoContent",
			ctx:               auth.ContextWithUserID(context.Background(), model.UserID(1)),
			setupOrderSvcMock: func(m *mocks.MockOrderService) {},
			setupBalanceSvcMock: func(m *mocks.MockBalanceService) {
				m.EXPECT().
					GetUserWithdrawals(gomock.Any(), model.UserID(1)).
					Times(1).Return([]model.Operation{}, nil)
			},
			setupRequest: func(r *http.Request) {},
			wantStatus:   http.StatusNoContent,
			wantHeader:   nil,
			wantBody:     "",
		},
		{
			name:              "Тест позитивного сценария",
			ctx:               auth.ContextWithUserID(context.Background(), model.UserID(1)),
			setupOrderSvcMock: func(m *mocks.MockOrderService) {},
			setupBalanceSvcMock: func(m *mocks.MockBalanceService) {
				m.EXPECT().
					GetUserWithdrawals(gomock.Any(), model.UserID(1)).
					Times(1).Return([]model.Operation{
					{
						OrderID:     "2377225624",
						Sum:         100,
						ProcessedAt: time.Time{},
					},
				}, nil)
			},
			setupRequest: func(r *http.Request) {},
			wantStatus:   http.StatusOK,
			wantHeader:   &headerWant{header: "Content-Type", contains: "application/json"},
			wantBody:     `[{"order": "2377225624","sum": 1,"processed_at": "0001-01-01T00:00:00Z"}]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			orderSvcMock := mocks.NewMockOrderService(ctrl)
			tt.setupOrderSvcMock(orderSvcMock)
			balanceSvcMock := mocks.NewMockBalanceService(ctrl)
			tt.setupBalanceSvcMock(balanceSvcMock)
			h := handlers.NewStandartHandlers(orderSvcMock, balanceSvcMock)

			req := httptest.NewRequest(http.MethodGet, endpoint, nil)
			if tt.ctx != nil {
				req = req.WithContext(tt.ctx)
			}
			tt.setupRequest(req)
			rw := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Get(endpoint, h.GetWithdrawals())
			router.ServeHTTP(rw, req)

			res := rw.Result()
			defer res.Body.Close()

			bodyBytes, err := io.ReadAll(res.Body)
			assert.NoError(t, err)

			if tt.wantStatus != 0 {
				assert.Equal(t, tt.wantStatus, res.StatusCode)
			}

			if tt.wantHeader != nil {
				assert.Contains(t, res.Header.Get(tt.wantHeader.header), tt.wantHeader.contains)
			}

			if tt.wantBody != "" {
				assert.JSONEq(t, tt.wantBody, string(bodyBytes))
			} else {
				assert.Equal(t, tt.wantBody, string(bodyBytes))
			}

		})
	}
}
