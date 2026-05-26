package handlers_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SergeyRG/gofermart/internal/auth"
	"github.com/SergeyRG/gofermart/internal/handlers"
	"github.com/SergeyRG/gofermart/internal/mocks"
	"github.com/SergeyRG/gofermart/internal/model"
	"github.com/SergeyRG/gofermart/internal/services"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestStandartHandlers_AddNewOrder(t *testing.T) {
	endpoint := "/api/user/orders"
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
			name:                "Если запрос не содержит header Content-Type text/plain, возвращается ошибка http.StatusBadRequest",
			ctx:                 auth.ContextWithUserID(context.Background(), model.UserID(1)),
			setupOrderSvcMock:   func(m *mocks.MockOrderService) {},
			setupBalanceSvcMock: func(m *mocks.MockBalanceService) {},
			setupRequest:        func(r *http.Request) {},
			wantStatus:          http.StatusBadRequest,
			wantHeader:          nil,
			wantBody:            "",
		},
		{
			name:                "Если запрос содержит некорректный OrderID, то хэндлер возвращает http.StatusUnprocessableEntity",
			ctx:                 auth.ContextWithUserID(context.Background(), model.UserID(1)),
			setupOrderSvcMock:   func(m *mocks.MockOrderService) {},
			setupBalanceSvcMock: func(m *mocks.MockBalanceService) {},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Content-Type", "text/plain")
				body := "123"
				r.Body = io.NopCloser(strings.NewReader(body))
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantHeader: nil,
			wantBody:   "",
		},
		{
			name: "Если OrderSVC возвращает services.ErrOrderAlreadyExist, то хэндлер возвращает http.StatusOk",
			ctx:  auth.ContextWithUserID(context.Background(), model.UserID(1)),
			setupOrderSvcMock: func(m *mocks.MockOrderService) {
				m.EXPECT().
					AddOrder(gomock.Any(), model.UserID(1), model.OrderID("2377225624")).
					Times(1).Return(services.ErrOrderAlreadyExist)
			},
			setupBalanceSvcMock: func(m *mocks.MockBalanceService) {},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Content-Type", "text/plain")
				body := "2377225624"
				r.Body = io.NopCloser(strings.NewReader(body))
			},
			wantStatus: http.StatusOK,
			wantHeader: nil,
			wantBody:   "",
		},
		{
			name: "Если OrderSVC возвращает services.ErrOrderAlreadyAddedByAnotherUser, то хэндлер возвращает http.StatusConflict",
			ctx:  auth.ContextWithUserID(context.Background(), model.UserID(1)),
			setupOrderSvcMock: func(m *mocks.MockOrderService) {
				m.EXPECT().
					AddOrder(gomock.Any(), model.UserID(1), model.OrderID("2377225624")).
					Times(1).Return(services.ErrOrderAlreadyAddedByAnotherUser)
			},
			setupBalanceSvcMock: func(m *mocks.MockBalanceService) {},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Content-Type", "text/plain")
				body := "2377225624"
				r.Body = io.NopCloser(strings.NewReader(body))
			},
			wantStatus: http.StatusConflict,
			wantHeader: nil,
			wantBody:   "",
		},
		{
			name: "Тест позитивного сценария",
			ctx:  auth.ContextWithUserID(context.Background(), model.UserID(1)),
			setupOrderSvcMock: func(m *mocks.MockOrderService) {
				m.EXPECT().
					AddOrder(gomock.Any(), model.UserID(1), model.OrderID("2377225624")).
					Times(1).Return(nil)
			},
			setupBalanceSvcMock: func(m *mocks.MockBalanceService) {},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Content-Type", "text/plain")
				body := "2377225624"
				r.Body = io.NopCloser(strings.NewReader(body))
			},
			wantStatus: http.StatusAccepted,
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
			router.Post(endpoint, h.AddNewOrder())
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
