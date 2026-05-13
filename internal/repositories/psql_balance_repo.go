package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/SergeyRG/gofermart/internal/model"
	"github.com/SergeyRG/gofermart/internal/storage"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type PSQLBalanceRepo struct {
	db *sql.DB
}

func NewPSQLBalanceRepo(DBDSN string) (*PSQLBalanceRepo, error) {
	db, err := storage.CreateAndCheckPSQLCon(DBDSN)
	if err != nil {
		return nil, err
	}
	return &PSQLBalanceRepo{db: db}, nil
}

func (repo PSQLBalanceRepo) GetByUserID(ctx context.Context, userID model.UserID) (*model.Balance, error) {
	query := `SELECT current, withdrawn
			FROM balances
			WHERE user_id = $1`

	sqlResults := repo.db.QueryRowContext(ctx, query, userID)

	balance := model.Balance{}
	err := sqlResults.Scan(&balance.Current, &balance.Withdrawn)

	if err != nil {
		return nil, fmt.Errorf(
			"ошибка получения баланса из БД: %w", err)
	}

	return &balance, nil
}
