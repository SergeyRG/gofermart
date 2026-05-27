package clients

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/SergeyRG/gofermart/internal/config"
	"github.com/SergeyRG/gofermart/internal/model"
	"github.com/SergeyRG/gofermart/internal/services"
	"github.com/go-resty/resty/v2"
)

type RestyAccrualClient struct {
	client *resty.Client
}

func NewRestyAccrualClient(cfg config.Config) *RestyAccrualClient {
	cl := resty.New().SetBaseURL(cfg.AccrualSystemAddress).
		SetTimeout(3 * time.Second).SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second)
	return &RestyAccrualClient{client: cl}
}

func (rc RestyAccrualClient) RequestAccrualSystemOrderStatus(
	ctx context.Context,
	orderID model.OrderID,
) (*services.AccrualSystemOrderInfo, error) {

	resp, err := rc.client.R().SetContext(ctx).SetPathParam("orderID", string(orderID)).
		Get("/api/orders/{orderID}")

	if err != nil {
		return nil, err
	}

	switch resp.StatusCode() {
	case 500:
		return nil, errors.New("внутренняя ошибка сервера расчета начислений")

	case 204:
		return nil, services.ErrOrderNotRegistered

	case 429:
		retryAfterString := resp.Header().Get("Retry-After")
		retryAfter, err := strconv.Atoi(retryAfterString)
		if err != nil {
			retryAfter = 60
		}
		return nil, services.ErrTooManyRequests{FreezeDuration: retryAfter}

	case 200:
		orderInfo := services.AccrualSystemOrderInfo{}
		err := json.Unmarshal(resp.Body(), &orderInfo)
		if err != nil {
			return nil, errors.New("неккоректный ответ от сервиса расчета начислений")
		}

		return &orderInfo, nil
	}

	return nil, errors.New("некорректный ответ от сервиса расчета начислений")
}
