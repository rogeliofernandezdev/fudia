package observability

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrorAttrs describes a technical failure for structured logs without query
// arguments, credentials or tenant data.
func ErrorAttrs(err error) []any {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return []any{
			"error_type", "postgres",
			"sql_state", pgErr.Code,
			"severity", pgErr.Severity,
			"constraint", pgErr.ConstraintName,
			"table", pgErr.TableName,
			"column", pgErr.ColumnName,
			"database_message", pgErr.Message,
		}
	}
	if err == nil {
		return []any{"error_type", "unknown"}
	}
	return []any{
		"error_type", fmt.Sprintf("%T", err),
		"error", err.Error(),
	}
}
