package model

import (
	"errors"
	"time"
)

var ErrInvalidOrderID = errors.New(
	"Некорректный номер заказа")

type OrderStatus string

const (
	StatusNew        OrderStatus = "NEW"        // заказ принят, но не обработан
	StatusProcessing OrderStatus = "PROCESSING" // расчет в процессе
	StatusInvalid    OrderStatus = "INVALID"    // система расчета признала номер неверным
	StatusProcessed  OrderStatus = "PROCESSED"  // расчет завершен
)

type UserID int64
type MoneyQty int64

func (m MoneyQty) ToFloat() float64 {
	return float64(m) / 100.0
}

type OrderID string

func (ID OrderID) isValid() bool {
	var sum int
	shouldDouble := false

	if ID == "" {
		return false
	}

	for i := len(ID) - 1; i >= 0; i-- {
		digit := int(ID[i] - '0')

		if digit < 0 || digit > 9 {
			return false
		}

		if shouldDouble {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		shouldDouble = !shouldDouble
	}

	return sum%10 == 0
}

func NewOrderID(value string) (OrderID, error) {
	newID := OrderID(value)
	if !newID.isValid() {
		return "", ErrInvalidOrderID
	}
	return newID, nil
}

type User struct {
	ID        UserID
	Login     string
	Current   MoneyQty
	Withdrawn MoneyQty
}

type Order struct {
	ID      OrderID
	UserID  UserID
	Status  OrderStatus
	Accrual MoneyQty
	AddedAt time.Time
}
