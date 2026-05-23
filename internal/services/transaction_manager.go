package services

import "context"

type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(txCtx context.Context) error) error
}
