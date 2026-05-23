package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/SergeyRG/gofermart/internal/auth"
	"github.com/SergeyRG/gofermart/internal/model"
	"github.com/SergeyRG/gofermart/internal/services"
)

func (h StandartHandlers) GetUserBalance() http.HandlerFunc {
	hf := func(rw http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}

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
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write(balanceJSON)

	}
	return http.HandlerFunc(hf)
}

func (h StandartHandlers) Withdraw() http.HandlerFunc {
	hf := func(rw http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}

		reqHeader := r.Header.Get("Content-Type")
		if !strings.Contains(reqHeader, "application/json") {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		oper := model.Operation{}
		err = json.Unmarshal(body, &oper)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		oper.UserID = userID
		oper.OpType = model.OperationWithdraw
		oper.ProcessedAt = time.Now().UTC()

		_, err = h.balanceSvc.ExecOper(r.Context(), oper)
		if err != nil {
			if errors.Is(err, services.ErrInsufficientBalance) {
				rw.WriteHeader(http.StatusPaymentRequired)
				return
			}
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
	return http.HandlerFunc(hf)
}

func (h StandartHandlers) GetWithdrawals() http.HandlerFunc {
	hf := func(rw http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}

		withdrawals, err := h.balanceSvc.GetUserWithdrawals(r.Context(), userID)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		if len(withdrawals) == 0 {
			rw.WriteHeader(http.StatusNoContent)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(rw).Encode(withdrawals)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	return http.HandlerFunc(hf)
}
