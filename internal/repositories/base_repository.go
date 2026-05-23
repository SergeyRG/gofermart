package repositories

import (
	"context"
	"database/sql"
)

type QueryExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type BaseRepository struct {
	db *sql.DB
}

func NewBaseRepo(db *sql.DB) *BaseRepository {
	return &BaseRepository{db: db}
}

func (br BaseRepository) GetExecutor(ctx context.Context) QueryExecutor {
	if tx, ok := ctx.Value(txCtxKey).(*sql.Tx); ok && tx != nil {
		return tx
	}
	return br.db
}
