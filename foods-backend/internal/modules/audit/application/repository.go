package application

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

type Repository interface {
	Record(context.Context, requestscope.Scope, string, string, string) error
}
