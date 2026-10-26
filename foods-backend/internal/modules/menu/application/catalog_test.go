package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
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
	t                   *testing.T
	events              *[]string
	failPortions        bool
	categoryUnavailable bool
	categoryProductID   *string
}

func (r *productRepository) record(ctx context.Context, event string) {
	r.t.Helper()
	if ctx.Value(transactionContextKey{}) != true {
		r.t.Fatalf("%s escaped product transaction", event)
	}
	*r.events = append(*r.events, event)
}
func (r *productRepository) ProductCategoryAvailable(ctx context.Context, _ scope, categoryID string, productID *string) (bool, error) {
	r.record(ctx, "category")
	if categoryID != testCategoryID {
		r.t.Fatalf("category must be trimmed before validation, got %q", categoryID)
	}
	r.categoryProductID = productID
	return !r.categoryUnavailable, nil
}
func (r *productRepository) LockProduct(ctx context.Context, _ scope, _ string) (string, string, string, string, error) {
	r.record(ctx, "lock")
	return "P1", "none", "prepared", "kitchen", nil
}
func (r *productRepository) UpdateProduct(ctx context.Context, _ scope, id string, in domain.ProductInput, _ string, _ bool, _ bool) (domain.Product, error) {
	r.record(ctx, "update")
	return domain.Product{ID: id, Name: in.Name, CategoryID: in.CategoryID}, nil
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
			_, err := service.CreateProduct(context.Background(), scope{}, domain.ProductInput{SKU: "P1", Name: "Sopa", Price: "10", CategoryID: categoryID(), QuantityControl: "portions", InitialPortionQuantity: &quantity})
			want := []string{"begin", "category", "product", "portions", "audit", "commit", "rollback"}
			if failure {
				want = []string{"begin", "category", "product", "portions", "rollback"}
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
	if _, err := service.CreateProduct(context.Background(), scope{}, domain.ProductInput{Name: "Sopa", Price: "10", CategoryID: categoryID(), QuantityControl: "portions"}); err == nil {
		t.Fatal("missing portions accepted")
	}
}

type noAudit struct{}

func (noAudit) Record(context.Context, scope, string, string, string) error { return nil }

const testCategoryID = "7f1c2a4e-3b6d-4c8e-9a10-2b3c4d5e6f70"

func categoryID() *string {
	id := "  " + testCategoryID + " "
	return &id
}
func errorCode(err error) string {
	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return ""
}
func TestProductRequiresCategoryBeforePersistence(t *testing.T) {
	blank, invalid := "  ", "sin-categoria"
	for name, category := range map[string]*string{"missing": nil, "blank": &blank, "malformed": &invalid} {
		t.Run(name, func(t *testing.T) {
			// A nil repository and unit of work prove nothing reaches persistence.
			service := NewCatalog(nil, nil, nil, nil)
			_, err := service.CreateProduct(context.Background(), scope{}, domain.ProductInput{Name: "Sopa", Price: "10", CategoryID: category})
			if errorCode(err) != "category_required" {
				t.Fatalf("create without a valid category: %v", err)
			}
			_, err = service.UpdateProduct(context.Background(), scope{}, "product-id", domain.ProductInput{Name: "Sopa", Price: "10", CategoryID: category})
			if errorCode(err) != "category_required" {
				t.Fatalf("update without a valid category: %v", err)
			}
		})
	}
}
func TestProductRejectsUnavailableCategoryInsideTransaction(t *testing.T) {
	events := []string{}
	repo := &productRepository{t: t, events: &events, categoryUnavailable: true}
	service := NewCatalog(repo, catalogWork{&events}, nil, nil)
	_, err := service.CreateProduct(context.Background(), scope{}, domain.ProductInput{SKU: "P1", Name: "Sopa", Price: "10", CategoryID: categoryID()})
	if errorCode(err) != "invalid_category" {
		t.Fatalf("inactive or foreign category accepted: %v", err)
	}
	if want := []string{"begin", "category", "rollback"}; !reflect.DeepEqual(events, want) {
		t.Fatalf("events=%v want=%v", events, want)
	}
	if repo.categoryProductID != nil {
		t.Fatalf("a new product must only accept active categories, got product %q", *repo.categoryProductID)
	}
}
func TestProductUpdateValidatesCategoryForThatProduct(t *testing.T) {
	events := []string{}
	repo := &productRepository{t: t, events: &events}
	service := NewCatalog(repo, catalogWork{&events}, noAudit{}, nil)
	if _, err := service.UpdateProduct(context.Background(), scope{}, "product-id", domain.ProductInput{SKU: "P1", Name: "Sopa", Price: "10", CategoryID: categoryID()}); err != nil {
		t.Fatalf("update with an available category: %v", err)
	}
	if repo.categoryProductID == nil || *repo.categoryProductID != "product-id" {
		t.Fatalf("update must let the product keep its current category, got %v", repo.categoryProductID)
	}
	if len(events) < 4 || !reflect.DeepEqual(events[:4], []string{"begin", "lock", "category", "update"}) {
		t.Fatalf("category must be validated after locking and before updating: %v", events)
	}
}
