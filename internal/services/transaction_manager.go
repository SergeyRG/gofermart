package services

import "context"

//go:generate mockgen -destination=../mocks/mock_transaction_manager.go -package=mocks . TransactionManager
type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(txCtx context.Context) error) error
}
