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

// GetUserBalance godoc
// @Summary       Получить текущий баланс пользователя
// @Description   Возвращает в виде JSON текущий баланс для текущего пользователя
// @Tags          balance
// @Produce       json
// @Security      CookieAuth
// @Success       200  {object}   model.Balance   "Успешное получение списка заказов"
// @Failure       401  {string}   string          "Пользователь не авторизован"
// @Failure       500  {string}   string          "Внутренняя ошибка сервера"
// @Router        /user/balance   [get]
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

// Withdraw godoc
// @Summary      Списать бонусные баллы
// @Description  Вызов эндпойнта приводит к списанию баллов с бонусного баланса, если их достаточно
// @Param        request   body      model.Operation  true  "Данные для списания (номер заказа и сумма)"
// @Tags         balance
// @Security     CookieAuth
// @Success      200  {string}  string 		    "Успешное списание"
// @Failure      400  {string}  string          "Некорректный запрос"
// @Failure      401  {string}  string          "Пользователь не авторизован"
// @Failure      402  {string}  string          "Недостаточно бонусов нв балансе"
// @Failure      500  {string}  string          "Внутренняя ошибка сервера"
// @Router       /user/balance/withdraw [post]
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
		defer r.Body.Close()

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

// GetWithdrawals godoc
// @Summary      Получить список всех списаний
// @Description  Возвращает JSON-массив всех списания текущего пользователя
// @Tags         balance
// @Security     CookieAuth
// @Success      200  {array}   model.Operation "Успешное списание"
// @Success      204  {string}  string          "Отсутствуют операции"
// @Failure      401  {string}  string          "Пользователь не авторизован"
// @Failure      500  {string}  string          "Внутренняя ошибка сервера"
// @Router       /api/user/withdrawals [get]
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
