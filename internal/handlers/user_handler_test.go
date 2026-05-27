package handlers_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SergeyRG/gofermart/internal/auth"
	"github.com/SergeyRG/gofermart/internal/handlers"
	"github.com/SergeyRG/gofermart/internal/mocks"
	"github.com/SergeyRG/gofermart/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestUserHandler_Register(t *testing.T) {
	endpoint := "/api/user/register"
	tests := []struct {
		name             string
		ctx              context.Context
		setupUserSvcMock func(m *mocks.MockUserService)
		setupRequest     func(r *http.Request)
		wantStatus       int
		wantHeader       *headerWant
		wantBody         string
	}{
		{
			name:             "Если запрос не содержит header application.json, то возвращается http.StatusBadRequest",
			ctx:              auth.ContextWithUserID(context.Background(), model.UserID(1)),
			setupUserSvcMock: func(m *mocks.MockUserService) {},
			setupRequest:     func(r *http.Request) {},
			wantStatus:       http.StatusBadRequest,
			wantHeader:       nil,
			wantBody:         "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			userSvcMock := mocks.NewMockUserService(ctrl)
			tt.setupUserSvcMock(userSvcMock)
			h := handlers.NewUserHandler(userSvcMock, *auth.NewJWTManager([]byte("test")))

			req := httptest.NewRequest(http.MethodPost, endpoint, nil)
			if tt.ctx != nil {
				req = req.WithContext(tt.ctx)
			}
			tt.setupRequest(req)
			rw := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Post(endpoint, h.Register())
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
