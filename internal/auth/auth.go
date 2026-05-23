package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/SergeyRG/gofermart/internal/model"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

type ctxKey int

const (
	userIDKey ctxKey = iota
)

func UserIDFromContext(ctx context.Context) (model.UserID, bool) {
	userID, ok := ctx.Value(userIDKey).(model.UserID)
	return userID, ok
}

func ContextWithUserID(ctx context.Context, userID model.UserID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

type JWTManager struct {
	secretKey []byte
}

func NewJWTManager(secretKey []byte) *JWTManager {
	return &JWTManager{secretKey: secretKey}
}

var ErrUnexpectedSigningMethod = errors.New("unexpected signing method")
var ErrTokenIsNotValid = errors.New("token is not valid")

type Claims struct {
	jwt.RegisteredClaims
	UserID model.UserID
}

func (jwtm JWTManager) GenerateJWTAuthToken(userID model.UserID) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID: userID,
	})

	tokenString, err := token.SignedString(jwtm.secretKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func (jwtm JWTManager) ValidateAndParseJWTAuthToken(tokenString string) (model.UserID, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("%w: %v", ErrUnexpectedSigningMethod, t.Header["alg"])
			}
			return jwtm.secretKey, nil
		})
	if err != nil {
		return 0, err
	}

	if !token.Valid {
		return 0, ErrTokenIsNotValid
	}

	return claims.UserID, nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
