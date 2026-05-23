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

type PSQLBalanceRepo struct {
	BaseRepository
}

func NewPSQLBalanceRepo(db *sql.DB) *PSQLBalanceRepo {
	return &PSQLBalanceRepo{
		BaseRepository: BaseRepository{db: db},
	}
}

func (repo PSQLBalanceRepo) GetByUserID(ctx context.Context, userID model.UserID) (*model.Balance, error) {
	qe := repo.GetExecutor(ctx)
	query := `SELECT current, withdrawn
			FROM balances
			WHERE user_id = $1`

	sqlResults := qe.QueryRowContext(ctx, query, userID)

	balance := model.Balance{}
	err := sqlResults.Scan(&balance.Current, &balance.Withdrawn)

	if err != nil {
		return nil, fmt.Errorf(
			"ошибка получения баланса из БД: %w", err)
	}

	return &balance, nil
}

func (repo PSQLBalanceRepo) ReduceBalance(
	ctx context.Context,
	userID model.UserID,
	sum model.MoneyQty) (*model.Balance, error) {

	qe := repo.GetExecutor(ctx)

	query := `UPDATE balances
			SET current = current - $1, withdrawn = withdrawn + $1
			WHERE user_id = $2
			RETURNING current, withdrawn`

	sqlResults := qe.QueryRowContext(ctx, query, sum, userID)

	balance := model.Balance{}
	err := sqlResults.Scan(&balance.Current, &balance.Withdrawn)

	if err != nil {
		var errPSQL *pgconn.PgError
		if errors.As(err, &errPSQL) && errPSQL.Code == "23514" && errPSQL.ConstraintName == "chk_balances_not_negative" {
			return nil, services.ErrInsufficientBalance
		}

		return nil, fmt.Errorf(
			"ошибка обновления баланса в БД: %w", err)
	}

	return &balance, nil
}

func (repo PSQLBalanceRepo) IncreaseBalance(
	ctx context.Context,
	userID model.UserID,
	sum model.MoneyQty) (*model.Balance, error) {

	qe := repo.GetExecutor(ctx)
	query := `UPDATE balances
			SET current = current + $1
			WHERE user_id = $2
			RETURNING current, withdrawn`

	sqlResults := qe.QueryRowContext(ctx, query, sum, userID)

	balance := model.Balance{}
	err := sqlResults.Scan(&balance.Current, &balance.Withdrawn)

	if err != nil {
		return nil, fmt.Errorf(
			"ошибка обновления баланса в БД: %w", err)
	}

	return &balance, nil
}

func (repo PSQLBalanceRepo) AddOperation(
	ctx context.Context,
	oper model.Operation) error {
	qe := repo.GetExecutor(ctx)
	query := `INSERT INTO operations (user_id, order_id, op_type, sum, processed_at) 
		VALUES ($1, $2, $3, $4, $5)`

	_, err := qe.ExecContext(ctx, query,
		oper.UserID,
		oper.OrderID,
		oper.OpType,
		oper.Sum,
		oper.ProcessedAt,
	)

	if err != nil {
		return fmt.Errorf(
			"ошибка записи операции в БД: %w", err)
	}

	return nil
}

func (repo PSQLBalanceRepo) GetWithdrawalsByUserID(ctx context.Context, userID model.UserID) ([]model.Operation, error) {
	qe := repo.GetExecutor(ctx)
	query := `SELECT user_id, order_id, op_type, sum, processed_at 
			FROM operations WHERE user_id = $1 and op_type = 'WITHDRAW'`

	rows, err := qe.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf(
			"ошибка выполнения SQL запроса: %w", err)
	}
	defer rows.Close()
	withdrawals := make([]model.Operation, 0)
	for rows.Next() {
		o := model.Operation{}
		err = rows.Scan(&o.UserID, &o.OrderID, &o.OpType, &o.Sum, &o.ProcessedAt)
		if err != nil {
			return nil, fmt.Errorf(
				"ошибка парсинга результата SQL запроса: %w", err)
		}
		withdrawals = append(withdrawals, o)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"ошибка записи операции в БД: %w", err)
	}

	return withdrawals, nil
}

func (repo PSQLBalanceRepo) AddUserBalance(
	ctx context.Context,
	uID model.UserID) error {

	qe := repo.GetExecutor(ctx)
	query := `
			INSERT INTO balances (user_id, current, withdrawn) 
			VALUES ($1, 0, 0)`

	_, err := qe.ExecContext(ctx, query, uID)

	if err != nil {
		return fmt.Errorf(
			"ошибка записи операции в БД: %w", err)
	}

	return nil
}
