package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/SergeyRG/gofermart/internal/services"
)

type txCtxKeyType struct{}

var txCtxKey = txCtxKeyType{}

type TxManager struct {
	db *sql.DB
}

func (txm TxManager) WithinTransaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	if _, ok := ctx.Value(txCtxKey).(*sql.Tx); ok {
		// Транзакция уже открыта выше по коду!
		// Просто выполняем функцию в текущем контексте, не создавая новый BEGIN/COMMIT
		return fn(ctx)
	}

	tx, err := txm.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ошибка создания транзакции СУБД: %w", err)
	}
	defer tx.Rollback()

	txCtx := context.WithValue(ctx, txCtxKey, tx)
	err = fn(txCtx)
	if err != nil {
		return fmt.Errorf("ошибка выполнения транзакции: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("ошибка фиксации транзакции: %w", err)
	}

	return nil
}

func NewTXManager(db *sql.DB) services.TransactionManager {
	return TxManager{db: db}
}
