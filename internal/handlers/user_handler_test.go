package handlers_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
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

type CookieTest struct {
	Name  string
	Value string
}

func TestUserHandler_Register(t *testing.T) {
	endpoint := "/api/user/register"
	jwtm := *auth.NewJWTManager([]byte("test"))
	cookieValue, _ := jwtm.GenerateJWTAuthToken(1)
	wantCookie := CookieTest{
		Name:  "auth_token",
		Value: cookieValue,
	}

	tests := []struct {
		name             string
		ctx              context.Context
		setupUserSvcMock func(m *mocks.MockUserService)
		requestBody      []byte
		setupRequest     func(r *http.Request)
		wantStatus       int
		wantHeader       *headerWant
		wantBody         string
		wantCookie       *CookieTest
	}{
		{
			name:             "Если запрос не содержит header application.json, то возвращается http.StatusBadRequest",
			ctx:              auth.ContextWithUserID(context.Background(), model.UserID(1)),
			setupUserSvcMock: func(m *mocks.MockUserService) {},
			setupRequest:     func(r *http.Request) {},
			requestBody:      nil,
			wantStatus:       http.StatusBadRequest,
			wantHeader:       nil,
			wantBody:         "",
		},
		{
			name:             "Если запрос не содержит некорректны json, то возвращается http.StatusBadRequest",
			ctx:              auth.ContextWithUserID(context.Background(), model.UserID(1)),
			setupUserSvcMock: func(m *mocks.MockUserService) {},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Content-Type", "application/json")
			},
			requestBody: []byte("test"),
			wantStatus:  http.StatusBadRequest,
			wantHeader:  nil,
			wantBody:    "",
		},
		{
			name: "Если UserSvc возвращает ErrLoginBusy, то возвращается http.StatusConflict",
			ctx:  auth.ContextWithUserID(context.Background(), model.UserID(1)),
			setupUserSvcMock: func(m *mocks.MockUserService) {
				m.EXPECT().AddUser(gomock.Any(), "test_login", "test_password").
					Return(model.UserID(1), services.ErrLoginBusy)
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Content-Type", "application/json")
			},
			requestBody: []byte("{\"login\":\"test_login\",\"password\":\"test_password\"}"),
			wantStatus:  http.StatusConflict,
			wantHeader:  nil,
			wantBody:    "",
		},
		{
			name: "Позитивный сценарий",
			ctx:  auth.ContextWithUserID(context.Background(), model.UserID(1)),
			setupUserSvcMock: func(m *mocks.MockUserService) {
				m.EXPECT().AddUser(gomock.Any(), "test_login", "test_password").
					Return(model.UserID(1), nil)
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Content-Type", "application/json")
			},
			requestBody: []byte("{\"login\":\"test_login\",\"password\":\"test_password\"}"),
			wantStatus:  http.StatusOK,
			wantHeader:  nil,
			wantBody:    "",
			wantCookie:  &wantCookie,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			userSvcMock := mocks.NewMockUserService(ctrl)
			tt.setupUserSvcMock(userSvcMock)
			h := handlers.NewUserHandler(userSvcMock, jwtm)

			req := httptest.NewRequest(http.MethodPost, endpoint, bytes.NewReader(tt.requestBody))
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

			if tt.wantCookie != nil {
				var foundCookie *http.Cookie
				for _, cookie := range res.Cookies() {
					if cookie.Name == tt.wantCookie.Name {
						foundCookie = cookie
						break
					}
				}

				if assert.NotNil(t, foundCookie, "Куки с именем %s не найден", tt.wantCookie.Name) {
					assert.Equal(t, tt.wantCookie.Value, foundCookie.Value)
				}
			}

			if tt.wantBody != "" {
				assert.JSONEq(t, tt.wantBody, string(bodyBytes))
			} else {
				assert.Equal(t, tt.wantBody, string(bodyBytes))
			}

		})
	}
}
