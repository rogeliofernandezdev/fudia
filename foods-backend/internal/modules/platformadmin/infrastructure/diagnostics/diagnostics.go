package diagnostics

import (
	"context"
	"errors"
	"fmt"
	"github.com/foods-platform/foods-backend/internal/modules/platformadmin/application"
	subscriptioncontracts "github.com/foods-platform/foods-backend/internal/modules/subscriptions/contracts"
	"github.com/foods-platform/foods-backend/internal/platform/observability"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"log/slog"
	"strings"
)

type Diagnostics struct{}

func (Diagnostics) Classify(err error) application.FailureCategory {
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, subscriptioncontracts.ErrNotFound) {
		return application.FailureMissing
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return application.FailureTransient
	}
	var p *pgconn.PgError
	if errors.As(err, &p) {
		switch p.ConstraintName {
		case "organizations_tax_id_key":
			return application.FailureTaxID
		case "users_organization_id_email_key":
			return application.FailureEmail
		case "locations_organization_id_code_key":
			return application.FailureLocationCode
		}
		if p.Code == "22P02" {
			return application.FailureIdentifier
		}
		if strings.HasPrefix(p.Code, "08") || strings.HasPrefix(p.Code, "53") {
			return application.FailureTransient
		}
		switch p.Code {
		case "40001", "40P01", "57P01", "57P02", "57P03":
			return application.FailureTransient
		}
	}
	return application.FailureUnknown
}
func (Diagnostics) Log(ctx context.Context, event, step string, err error, extra ...any) {
	m := observability.From(ctx)
	attrs := []any{"correlation_id", m.CorrelationID, "flow", "platform.onboarding", "method", m.Method, "path", m.Path}
	if step != "" {
		attrs = append(attrs, "step", step)
	}
	attrs = append(attrs, extra...)
	if err != nil {
		attrs = append(attrs, SafeErrorLogAttrs(err)...)
		slog.Error(event, attrs...)
	} else {
		slog.Info(event, attrs...)
	}
}
func SafeErrorLogAttrs(err error) []any {
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
