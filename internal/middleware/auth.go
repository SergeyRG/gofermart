package middleware

import (
	"net/http"

	"github.com/SergeyRG/gofermart/internal/auth"
	"github.com/SergeyRG/gofermart/internal/model"
)

func Auth(jwtm auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			cookieToken, err := r.Cookie("auth_token")
			var tokenString string
			var userID model.UserID

			if err != nil {
				rw.WriteHeader(http.StatusUnauthorized)
				return
			}
			tokenString = cookieToken.Value
			userID, err = jwtm.ValidateAndParseJWTAuthToken(tokenString)

			if err != nil || userID == 0 {
				rw.WriteHeader(http.StatusUnauthorized)
				return
			}

			ctx := auth.ContextWithUserID(r.Context(), userID)
			next.ServeHTTP(rw, r.WithContext(ctx))
		})
	}
}
