package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/SergeyRG/gofermart/internal/auth"
	"github.com/SergeyRG/gofermart/internal/model"
	"github.com/SergeyRG/gofermart/internal/services"
)

func (h StandardHandlers) AddNewOrder() http.HandlerFunc {
	hf := func(rw http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}

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
			rw.WriteHeader(http.StatusUnprocessableEntity)
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

func (h StandardHandlers) GetUserOrders() http.HandlerFunc {
	hf := func(rw http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}
		defer r.Body.Close()

		orders := h.orderSvc.GetUserOrders(r.Context(), userID)

		isFirst := true
		hasData := false

		for o, err := range orders {
			if err != nil {
				log.Printf("ошибка чтения данных из БД: %v", err)
				return
			}

			if !hasData {
				hasData = true
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusOK)
				_, _ = rw.Write([]byte("["))
			}

			if !isFirst {
				_, _ = rw.Write([]byte(",\n"))
			}
			isFirst = false

			oJSON, err := json.Marshal(o)
			if err != nil {
				log.Printf("ошибка кодирования JSON: %v", err)
				return
			}
			_, _ = rw.Write(oJSON)

		}

		if !hasData {
			rw.WriteHeader(http.StatusNoContent)
			return
		}

		_, _ = rw.Write([]byte("]"))

	}
	return http.HandlerFunc(hf)
}
