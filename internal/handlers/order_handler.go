package handlers

import (
	"net/http"

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
		rw.Header().Add("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("{\"status\": \"OK\"}"))
	}
	return http.HandlerFunc(hf)
}
