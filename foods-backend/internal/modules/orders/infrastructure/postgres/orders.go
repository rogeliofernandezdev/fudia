package postgres

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/orders/application"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type scope = requestscope.Scope
type Orders struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Orders                       { return &Orders{pool} }
func (r *Orders) db(ctx context.Context) database.Executor { return database.ExecutorFor(ctx, r.pool) }
func orderError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrNotFound
	}
	return err
}

var _ application.OrderRepository = (*Orders)(nil)
