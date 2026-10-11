package database

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Executor interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}
type transactionKey struct{}
type binding struct {
	pool *pgxpool.Pool
	tx   pgx.Tx
}

// ExecutorFor binds all repositories using this pool to the same transaction.
func ExecutorFor(ctx context.Context, pool *pgxpool.Pool) Executor {
	if b, ok := ctx.Value(transactionKey{}).(binding); ok && b.pool == pool {
		return b.tx
	}
	return pool
}

type UnitOfWork struct{ pool *pgxpool.Pool }

func NewUnitOfWork(pool *pgxpool.Pool) *UnitOfWork { return &UnitOfWork{pool} }

type transaction struct {
	ctx context.Context
	tx  pgx.Tx
}

func (t *transaction) Context() context.Context           { return t.ctx }
func (t *transaction) Commit(ctx context.Context) error   { return t.tx.Commit(ctx) }
func (t *transaction) Rollback(ctx context.Context) error { return t.tx.Rollback(ctx) }
func (u *UnitOfWork) Begin(ctx context.Context, o unitofwork.Options) (unitofwork.Transaction, error) {
	if _, ok := ctx.Value(transactionKey{}).(binding); ok {
		return nil, errors.New("nested unit of work is not supported")
	}
	opts := pgx.TxOptions{}
	if o.ReadOnly {
		opts.AccessMode = pgx.ReadOnly
	}
	if o.RepeatableRead {
		opts.IsoLevel = pgx.RepeatableRead
	}
	tx, err := u.pool.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &transaction{context.WithValue(ctx, transactionKey{}, binding{u.pool, tx}), tx}, nil
}
