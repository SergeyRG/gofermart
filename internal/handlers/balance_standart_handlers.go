package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/SergeyRG/gofermart/internal/model"
)

func (h StandartHandlers) GetUserBalance() http.HandlerFunc {
	hf := func(rw http.ResponseWriter, r *http.Request) {
		userID := model.UserID(1) // TODO

		balance, err := h.balanceSvc.GetUserBalance(r.Context(), userID)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		balanceJSON, err := json.Marshal(balance)

		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.Write(balanceJSON)
		rw.WriteHeader(http.StatusOK)
	}
	return http.HandlerFunc(hf)
}
