package sqlite

import (
	"context"
	"database/sql"

	db "transport-app/db/generated/sqlite"
	"transport-app/internal/repository"
)

// SQLRepository implements all repository interfaces using SQLite.
type SQLRepository struct {
	db *sql.DB
	q  *db.Queries
}

// NewRepository creates a new SQLRepository with the given database connection.
func NewRepository(dbConn *sql.DB) *SQLRepository {
	return &SQLRepository{
		db: dbConn,
		q:  db.New(dbConn),
	}
}

// DB returns the underlying database connection.
func (r *SQLRepository) DB() *sql.DB {
	return r.db
}

// Q returns the Queries to use for this request, picking up a transaction
// from context if one is active. This allows repository methods to
// transparently use tx-bound queries when called within a transaction.
func (r *SQLRepository) Q(ctx context.Context) *db.Queries {
	if tx := repository.TxFromContext(ctx); tx != nil {
		return r.q.WithTx(tx)
	}
	return r.q
}

func (r *SQLRepository) exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if tx := repository.TxFromContext(ctx); tx != nil {
		return tx.ExecContext(ctx, query, args...)
	}
	return r.db.ExecContext(ctx, query, args...)
}

func (r *SQLRepository) queryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if tx := repository.TxFromContext(ctx); tx != nil {
		return tx.QueryRowContext(ctx, query, args...)
	}
	return r.db.QueryRowContext(ctx, query, args...)
}
