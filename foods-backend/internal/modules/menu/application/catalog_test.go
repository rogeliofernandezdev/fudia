package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"reflect"
	"testing"
)

type transactionContextKey struct{}
type catalogTransaction struct {
	ctx    context.Context
	events *[]string
}

func (t *catalogTransaction) Context() context.Context { return t.ctx }
func (t *catalogTransaction) Commit(context.Context) error {
	*t.events = append(*t.events, "commit")
	return nil
}
func (t *catalogTransaction) Rollback(context.Context) error {
	*t.events = append(*t.events, "rollback")
	return nil
}

type catalogWork struct{ events *[]string }

func (w catalogWork) Begin(ctx context.Context, _ unitofwork.Options) (unitofwork.Transaction, error) {
	*w.events = append(*w.events, "begin")
	return &catalogTransaction{context.WithValue(ctx, transactionContextKey{}, true), w.events}, nil
}

type productRepository struct {
	CatalogRepository
	t            *testing.T
	events       *[]string
	failPortions bool
}

func (r *productRepository) record(ctx context.Context, event string) {
	r.t.Helper()
	if ctx.Value(transactionContextKey{}) != true {
		r.t.Fatalf("%s escaped product transaction", event)
	}
	*r.events = append(*r.events, event)
}
func (r *productRepository) CreateProduct(ctx context.Context, _ scope, in domain.ProductInput, _ bool) (domain.Product, error) {
	r.record(ctx, "product")
	return domain.Product{ID: "product-id", Name: in.Name}, nil
}
func (r *productRepository) InitialPortions(ctx context.Context, _ scope, _ domain.ProductInput, _ domain.Product) (string, error) {
	r.record(ctx, "portions")
	if r.failPortions {
		return "", errors.New("storage failure")
	}
	return "product-id", nil
}
func (r *productRepository) ProductCreatedAudit(ctx context.Context, _ scope, _ domain.Product, _ domain.ProductInput) (int64, error) {
	r.record(ctx, "audit")
	return 1, nil
}
func TestProductAndInitialPortionsShareTransaction(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(map[bool]string{false: "commit", true: "rollback"}[failure], func(t *testing.T) {
			events := []string{}
			repo := &productRepository{t: t, events: &events, failPortions: failure}
			service := NewCatalog(repo, catalogWork{&events}, nil, nil)
			quantity := 5
			_, err := service.CreateProduct(context.Background(), scope{}, domain.ProductInput{SKU: "P1", Name: "Sopa", Price: "10", QuantityControl: "portions", InitialPortionQuantity: &quantity})
			want := []string{"begin", "product", "portions", "audit", "commit", "rollback"}
			if failure {
				want = []string{"begin", "product", "portions", "rollback"}
			}
			if (err != nil) != failure {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(events, want) {
				t.Fatalf("events=%v want=%v", events, want)
			}
		})
	}
}
func TestProductValidatesInitialPortionsBeforePersistence(t *testing.T) {
	service := NewCatalog(nil, nil, nil, nil)
	if _, err := service.CreateProduct(context.Background(), scope{}, domain.ProductInput{Name: "Sopa", Price: "10", QuantityControl: "portions"}); err == nil {
		t.Fatal("missing portions accepted")
	}
}
