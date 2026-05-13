package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/SergeyRG/gofermart/internal/model"
	"github.com/SergeyRG/gofermart/internal/services"
)

func (h StandartHandlers) AddNewOrder() http.HandlerFunc {
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

		err = h.orderSvc.AddOrder(r.Context(), userID, orderID)
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

func (h StandartHandlers) GetUserOrders() http.HandlerFunc {
	hf := func(rw http.ResponseWriter, r *http.Request) {
		userID := model.UserID(1) // TODO

		defer r.Body.Close()

		orders, err := h.orderSvc.GetUserOrders(r.Context(), userID)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		if len(orders) == 0 {
			rw.WriteHeader(http.StatusNoContent)
			return
		}

		ordersJSON, err := json.Marshal(orders)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write(ordersJSON)

	}
	return http.HandlerFunc(hf)
}
