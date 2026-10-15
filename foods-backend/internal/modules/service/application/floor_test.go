package application

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/foods-platform/foods-backend/internal/modules/service/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

type batchRepository struct {
	FloorRepository
	err error
}

func (r batchRepository) Batch(context.Context, scope, []domain.TableBatchItem) (domain.TableBatchResult, bool, error) {
	return domain.TableBatchResult{}, false, r.err
}

func TestTableBatchFailureCodesDistinguishRejectedFromUnconfirmed(t *testing.T) {
	cause := errors.New("function gen_random_bytes(integer) does not exist")
	cases := []struct {
		name, code string
		repoErr    error
	}{
		{"rechazo de PostgreSQL", "tables_not_saved", fmt.Errorf("%w: %w", ErrNotSaved, cause)},
		{"resultado incierto", "tables_unavailable", cause},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			floor := NewFloor(batchRepository{err: tc.repoErr}, nil, nil)
			_, err := floor.Batch(context.Background(), scope{}, []domain.TableInput{{Name: "Mesa 1"}})
			var failure *apperror.Error
			if !errors.As(err, &failure) || failure.Code != tc.code || failure.Kind != apperror.Unavailable {
				t.Fatalf("unexpected failure: %#v", err)
			}
			if !errors.Is(failure.Cause, cause) {
				t.Fatalf("the technical cause must reach the server log: %v", failure.Cause)
			}
		})
	}
	floor := NewFloor(batchRepository{err: ErrConflict}, nil, nil)
	var failure *apperror.Error
	if _, err := floor.Batch(context.Background(), scope{}, []domain.TableInput{{Name: "Mesa 1"}}); !errors.As(err, &failure) || failure.Code != "table_conflict" {
		t.Fatalf("duplicates stay as conflicts: %v", err)
	}
}
