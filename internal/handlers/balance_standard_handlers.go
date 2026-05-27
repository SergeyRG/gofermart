package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/SergeyRG/gofermart/internal/auth"
	"github.com/SergeyRG/gofermart/internal/model"
	"github.com/SergeyRG/gofermart/internal/services"
)

func (h StandardHandlers) GetUserBalance() http.HandlerFunc {
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

func (h StandardHandlers) Withdraw() http.HandlerFunc {
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

func (h StandardHandlers) GetWithdrawals() http.HandlerFunc {
	hf := func(rw http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}
		withdrawals := h.balanceSvc.GetUserWithdrawals(r.Context(), userID)

		encoder := json.NewEncoder(rw)
		isFirst := true
		hasData := false

		for op, err := range withdrawals {
			if err != nil {
				log.Printf("ошибка чтения данных из БД: %v", err)
				if !hasData {
					rw.WriteHeader(http.StatusInternalServerError)
				}
				return
			}

			if !hasData {
				hasData = true
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusOK)
				_, _ = rw.Write([]byte("["))
			}

			if !isFirst {
				_, _ = rw.Write([]byte(","))
			}
			isFirst = false

			if err := encoder.Encode(op); err != nil {
				log.Printf("ошибка кодирования JSON: %v", err)
				return
			}
		}

		if !hasData {
			rw.WriteHeader(http.StatusNoContent)
			return
		}

		_, _ = rw.Write([]byte("]"))
	}
	return http.HandlerFunc(hf)
}
