package handlers

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/SergeyRG/gofermart/internal/model"
	"github.com/SergeyRG/gofermart/internal/services"
)

type OrderHandler struct {
	svc services.OrderService
}

func NewOrderHandler(svc services.OrderService) OrderHandler {
	return OrderHandler{svc: svc}
}

func (h OrderHandler) AddNewOrder() http.HandlerFunc {
	hf := func(rw http.ResponseWriter, r *http.Request) {
		userID := model.UserID(1) // TODO

		reqHeader := r.Header.Get("Content-Type")

		if !strings.Contains(reqHeader, "text/plain") {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		orderIDStr, err := io.ReadAll(r.Body)

		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		orderID, err := model.NewOrderID(string(orderIDStr))
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		err = h.svc.Add(r.Context(), userID, orderID)
		if err != nil {
			if errors.Is(err, services.ErrOrderAlreadyExist) {
				rw.WriteHeader(http.StatusOK)
				return
			}
			if errors.Is(err, services.ErrOrderAlreadyAddedByAnotherUser) {
				rw.WriteHeader(http.StatusConflict)
				return
			}

			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusAccepted)
	}
	return http.HandlerFunc(hf)
}
