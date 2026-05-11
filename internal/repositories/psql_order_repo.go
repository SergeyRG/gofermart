package repositories

import (
	"context"
	"database/sql"

	"github.com/SergeyRG/gofermart/internal/model"
)

type PSQLOrderRepo struct {
	db *sql.DB
}

func NewPSQLOrderRepo(db *sql.DB) PSQLOrderRepo {
	return PSQLOrderRepo{db: db}
}

func (repo PSQLOrderRepo) Add(ctx context.Context, order model.Order) error {
	return nil
}

func (repo PSQLOrderRepo) GetByID(ctx context.Context, order model.OrderID) (model.Order, error) {
	return model.Order{}, nil
}
