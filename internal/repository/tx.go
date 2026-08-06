package repository

import (
	"context"
	"database/sql"
)

type txKey struct{}

// WithTxInContext stores a *sql.Tx in context for repository methods to pick up.
func WithTxInContext(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// TxFromContext retrieves the *sql.Tx from context, or nil if none.
func TxFromContext(ctx context.Context) *sql.Tx {
	tx, _ := ctx.Value(txKey{}).(*sql.Tx)
	return tx
}

// DBGetter provides access to the underlying *sql.DB.
type DBGetter interface {
	DB() *sql.DB
}

// TxManager manages database transactions, ensuring atomicity across
// multiple repository operations.
type TxManager interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type txManager struct {
	db *sql.DB
}

// NewTxManager creates a TxManager backed by the given DBGetter.
func NewTxManager(getter DBGetter) TxManager {
	return &txManager{db: getter.DB()}
}

// WithTransaction begins a transaction, injects it into the context, runs fn,
// and commits or rolls back based on fn's outcome.
func (tm *txManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := tm.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	ctx = WithTxInContext(ctx, tx)
	if err := fn(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
