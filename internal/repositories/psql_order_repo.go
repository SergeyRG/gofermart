package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/SergeyRG/gofermart/internal/model"
	"github.com/SergeyRG/gofermart/internal/services"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type PSQLOrderRepo struct {
	BaseRepository
}

func NewPSQLOrderRepo(db *sql.DB) *PSQLOrderRepo {
	return &PSQLOrderRepo{BaseRepository: BaseRepository{db: db}}
}

func (repo PSQLOrderRepo) Add(ctx context.Context, order model.Order) error {
	qe := repo.GetExecutor(ctx)
	query := `INSERT INTO orders (id, user_id, status, accrual, added_at, updated_at)
			VALUES ($1, $2 , $3, $4, $5, $6)`

	_, err := qe.ExecContext(ctx, query,
		order.ID,
		order.UserID,
		order.Status,
		order.Accrual,
		order.AddedAt,
		order.UpdatedAt,
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
	row := qe.QueryRowContext(ctx, get_user_query, order.ID)

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
	qe := repo.GetExecutor(ctx)
	query := `SELECT id, status, COALESCE(accrual, 0), added_at
			FROM orders
			WHERE user_id = $1
			ORDER BY added_at DESC`

	orders := make([]model.Order, 0)

	sqlResults, err := qe.QueryContext(ctx, query, userID)

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

func (repo PSQLOrderRepo) ChangeOrderStatus(ctx context.Context, oID model.OrderID, oStatus model.OrderStatus) error {
	query := `UPDATE orders
			  SET status = $1, updated_at = Now()
			  WHERE id = $2;`
	qEx := repo.GetExecutor(ctx)

	res, err := qEx.ExecContext(ctx, query, oStatus, oID)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса заказа: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения информации об обновленных заказах: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("заказ %v не найден", oID)
	}

	return nil
}

func (repo PSQLOrderRepo) UpdateOrder(ctx context.Context, o model.Order) error {
	query := `UPDATE orders
			  SET status = $1, updated_at = Now(), accrual = $2
			  WHERE id = $3;`
	qEx := repo.GetExecutor(ctx)

	res, err := qEx.ExecContext(ctx, query, o.Status, o.Accrual, o.ID)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса заказа: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения информации об обновленных заказах: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("заказ %v не найден", o.ID)
	}

	return nil
}

func (repo PSQLOrderRepo) GetByIDForUpdate(ctx context.Context, oID model.OrderID) (*model.Order, error) {
	qe := repo.GetExecutor(ctx)
	query := `
			SELECT id, user_id, status, COALESCE(accrual, 0), added_at
			FROM orders
			WHERE id = $1 FOR UPDATE;`

	res := qe.QueryRowContext(ctx, query, oID)
	o := model.Order{}
	err := res.Scan(&o.ID, &o.UserID, &o.Status, &o.Accrual, &o.AddedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, services.ErrNotFound
		}
		return nil, fmt.Errorf(
			"ошибка информации об заказе из БД по ИД: %w", err)
	}

	return &o, nil
}

func (repo PSQLOrderRepo) GetNextOrderIDForProcessing(ctx context.Context) (*model.Order, error) {
	qe := repo.GetExecutor(ctx)
	query := `
		UPDATE orders 
		SET status = 'PROCESSING', updated_at = NOW()
		WHERE id = (
			SELECT id 
			FROM orders 
			WHERE (status = 'NEW') 
			   OR (status = 'PROCESSING' AND updated_at < NOW() - INTERVAL '5 minutes')
			ORDER BY added_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED 
		)
		RETURNING id, status, COALESCE(accrual, 0), added_at;`

	res := qe.QueryRowContext(ctx, query)
	o := model.Order{}
	err := res.Scan(&o.ID, &o.Status, &o.Accrual, &o.AddedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, services.ErrNoOrdersForProcessing
		}
		return nil, fmt.Errorf(
			"ошибка запроса заказа для обработки в БД: %w", err)
	}

	return &o, nil
}
