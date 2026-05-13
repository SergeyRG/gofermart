package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/SergeyRG/gofermart/internal/model"
	"github.com/SergeyRG/gofermart/internal/services"
	"github.com/SergeyRG/gofermart/internal/storage"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type PSQLOrderRepo struct {
	db *sql.DB
}

func NewPSQLOrderRepo(DBDSN string) (*PSQLOrderRepo, error) {
	db, err := storage.CreateAndCheckPSQLCon(DBDSN)
	if err != nil {
		return nil, err
	}
	return &PSQLOrderRepo{db: db}, nil
}

func (repo PSQLOrderRepo) Add(ctx context.Context, order model.Order) error {
	query := `INSERT INTO orders (id, user_id, status, accrual, added_at)
			VALUES ($1, $2 , $3, $4, $5)`

	_, err := repo.db.ExecContext(ctx, query,
		order.ID,
		order.UserID,
		order.Status,
		order.Accrual,
		order.AddedAt,
	)

	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return fmt.Errorf("Ошибка добавления заказа: %w", services.ErrUndefinedRepositoryError)
	}

	if pgErr.Code != "23505" {
		return fmt.Errorf("Ошибка добавления заказа: %w", services.ErrUndefinedRepositoryError)
	}

	if pgErr.ConstraintName != "orders_id_key" {
		return fmt.Errorf("Ошибка добавления заказа: %w", services.ErrUndefinedRepositoryError)
	}

	get_user_query := "SELECT user_id FROM orders WHERE id = $1"
	row := repo.db.QueryRowContext(ctx, get_user_query, order.ID)

	var existedUserID model.UserID
	err = row.Scan(&existedUserID)
	if err != nil {
		return fmt.Errorf("Ошибка добавления заказа: %w", services.ErrUndefinedRepositoryError)
	}

	if existedUserID == order.UserID {
		return services.ErrOrderAlreadyExist
	} else {
		return services.ErrOrderAlreadyAddedByAnotherUser
	}
}

func (repo PSQLOrderRepo) GetByUserID(ctx context.Context, userID model.UserID) ([]model.Order, error) {
	query := `SELECT id, status, accrual, added_at
			FROM orders
			WHERE user_id = $1
			ORDER BY added_at DESC`

	orders := make([]model.Order, 0)

	sqlResults, err := repo.db.QueryContext(ctx, query, userID)

	if err != nil {
		return nil, fmt.Errorf(
			"ошибка выполнения SQL запроса: %w", err)
	}
	defer sqlResults.Close()

	for sqlResults.Next() {
		o := model.Order{}
		err = sqlResults.Scan(&o.ID, &o.Status, &o.Accrual, &o.AddedAt)
		if err != nil {
			return nil, fmt.Errorf(
				"ошибка парсинга результата SQL запроса: %w", err)
		}
		orders = append(orders, o)
	}

	if err = sqlResults.Err(); err != nil {
		return nil, fmt.Errorf("ошибка парсинга результата SQL запроса: %w", err)
	}

	return orders, nil
}
