package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/SergeyRG/gofermart/internal/auth"
	"github.com/SergeyRG/gofermart/internal/services"
)

type userAuthDTO struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type UserHandler struct {
	userSVC services.UserService
	jwtm    auth.JWTManager
}

func NewUserHandler(userSvc services.UserService, jwtm auth.JWTManager) *UserHandler {
	return &UserHandler{userSVC: userSvc, jwtm: jwtm}
}

func (h UserHandler) Register() http.HandlerFunc {
	hf := func(rw http.ResponseWriter, req *http.Request) {
		if !strings.Contains(req.Header.Get("Content-Type"), "application/json") {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		user := &userAuthDTO{}
		req.Body = http.MaxBytesReader(rw, req.Body, 1024*1024)
		body, err := io.ReadAll(req.Body)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		err = json.Unmarshal(body, user)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		if user.Login == "" || user.Password == "" {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		uID, err := h.userSVC.AddUser(req.Context(), user.Login, user.Password)

		if err != nil {
			if errors.Is(err, services.ErrLoginBusy) {
				rw.WriteHeader(http.StatusConflict)
				return
			}

			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		tokenString, err := h.jwtm.GenerateJWTAuthToken(uID)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		http.SetCookie(rw, &http.Cookie{
			Name:     "auth_token",
			Value:    tokenString,
			Path:     "/",
			HttpOnly: true,
		})

		rw.WriteHeader(http.StatusOK)
	}
	return hf
}

func (h UserHandler) Login() http.HandlerFunc {
	hf := func(rw http.ResponseWriter, req *http.Request) {
		if !strings.Contains(req.Header.Get("Content-Type"), "application/json") {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		user := &userAuthDTO{}
		req.Body = http.MaxBytesReader(rw, req.Body, 1024*1024)
		body, err := io.ReadAll(req.Body)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		err = json.Unmarshal(body, user)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		if user.Login == "" || user.Password == "" {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		uID, err := h.userSVC.AuthUser(req.Context(), user.Login, user.Password)

		if err != nil {
			if errors.Is(err, services.ErrWrongLoginOrPassword) {
				rw.WriteHeader(http.StatusUnauthorized)
				return
			}
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		tokenString, err := h.jwtm.GenerateJWTAuthToken(uID)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		http.SetCookie(rw, &http.Cookie{
			Name:     "auth_token",
			Value:    tokenString,
			Path:     "/",
			HttpOnly: true,
		})

		rw.WriteHeader(http.StatusOK)
	}
	return hf
}
