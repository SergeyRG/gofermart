package model

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
)

var ErrInvalidOrderID = errors.New(
	"Некорректный номер заказа")

type OrderStatus string

const (
	StatusNew        OrderStatus = "NEW"
	StatusProcessing OrderStatus = "PROCESSING"
	StatusInvalid    OrderStatus = "INVALID"
	StatusProcessed  OrderStatus = "PROCESSED"
)

type OperationType string

const (
	OperationWithdraw OperationType = "WITHDRAW"
	OperationDeposit  OperationType = "DEPOSIT"
)

type UserID int64

type MoneyQty int64

func (m MoneyQty) MarshalJSON() ([]byte, error) {
	floatValue := float64(m) / 100.0
	return []byte(fmt.Sprintf("%.2f", floatValue)), nil
}

func (m *MoneyQty) UnmarshalJSON(data []byte) error {
	val, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return fmt.Errorf("невалидное число: %w", err)
	}

	*m = MoneyQty(math.Round(val * 100))
	return nil
}

type OrderID string

func (o *OrderID) UnmarshalJSON(data []byte) error {
	orderID, err := NewOrderID(string(bytes.Trim(data, "\"")))
	if err != nil {
		return err
	}
	*o = orderID
	return nil
}

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
	ID        OrderID     `json:"number"`
	UserID    UserID      `json:"-"`
	Status    OrderStatus `json:"status"`
	Accrual   MoneyQty    `json:"accrual"`
	AddedAt   time.Time   `json:"uploaded_at"`
	UpdatedAt time.Time   `json:"-"`
}

func (o Order) MarshalJSON() ([]byte, error) {
	type OrderCopy Order

	var accrual *MoneyQty = nil

	if o.Status == StatusProcessed {
		accrual = &o.Accrual
	}

	jsonStruct := struct {
		ID      OrderID     `json:"number"`
		Status  OrderStatus `json:"status"`
		Accrual *MoneyQty   `json:"accrual,omitempty"`
		AddedAt time.Time   `json:"uploaded_at"`
	}{
		ID:      o.ID,
		Status:  o.Status,
		Accrual: accrual,
		AddedAt: o.AddedAt,
	}
	return json.Marshal(jsonStruct)
}

func NewOrder(ID OrderID, userID UserID) Order {
	return Order{
		ID:        ID,
		UserID:    userID,
		Status:    StatusNew,
		AddedAt:   time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

type Balance struct {
	UserID    UserID   `json:"-"`
	Current   MoneyQty `json:"current"`
	Withdrawn MoneyQty `json:"withdrawn"`
}

type Operation struct {
	UserID      UserID        `json:"-"`
	OrderID     OrderID       `json:"order"`
	Sum         MoneyQty      `json:"sum"`
	OpType      OperationType `json:"-"`
	ProcessedAt time.Time     `json:"processed_at"`
}

// func NewOperation(userID UserID,
// 	orderID string,
// 	sum string,
// 	opType string,
// 	processedAt *time.Time) (*Operation, error) {

// 	orderIDVerified, err := NewOrderID(orderID)
// 	if err != nil {
// 		return nil, fmt.Errorf("ошибка проверки данных: %w", err)
// 	}

// 	return &Operation{
// 		UserID:      userID,
// 		OrderID:     orderIDVerified,
// 		Sum:         sum,
// 		OpType:      opType,
// 		ProcessedAt: *processedAt,
// 	}, nil
// }
