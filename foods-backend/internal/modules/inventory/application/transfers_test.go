package application

import (
	"context"
	"strings"
	"testing"

	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

const (
	sourceLocation      = "11111111-1111-4111-8111-111111111111"
	destinationLocation = "22222222-2222-4222-8222-222222222222"
)

type stockTransaction struct{ ctx context.Context }

func (t stockTransaction) Context() context.Context     { return t.ctx }
func (stockTransaction) Commit(context.Context) error   { return nil }
func (stockTransaction) Rollback(context.Context) error { return nil }

type stockWork struct{}

func (stockWork) Begin(ctx context.Context, _ unitofwork.Options) (unitofwork.Transaction, error) {
	return stockTransaction{ctx}, nil
}

type noAudit struct{}

func (noAudit) Record(context.Context, scope, string, string, string) error { return nil }

// transferRepository behaves like PostgreSQL: it accepts any UUID spelling and
// returns balances keyed by the canonical id. Any other call panics.
type transferRepository struct {
	StockRepository
	balances    map[string]domain.LocationBalance
	created     bool
	destination float64
	movements   []Movement
}

func (r *transferRepository) TransferDestination(_ context.Context, _ scope, in domain.TransferInput) (string, string, error) {
	return strings.ToLower(in.ToLocationID), "Local destino", nil
}
func (r *transferRepository) CreateTransfer(context.Context, scope, domain.TransferInput) (string, string, error) {
	r.created = true
	return "transfer-1", "TRF-0001", nil
}
func (r *transferRepository) TransferItem(context.Context, scope, domain.TransferLine) (*string, string, error) {
	return nil, "Arroz", nil
}
func (r *transferRepository) EnsureTransferBalances(context.Context, scope, domain.TransferLine, domain.TransferInput) (int64, error) {
	return 0, nil
}
func (r *transferRepository) LockTransferBalances(context.Context, scope, string, string) (map[string]domain.LocationBalance, error) {
	return r.balances, nil
}
func (r *transferRepository) TransferSourceBalance(context.Context, scope, domain.TransferLine, float64, float64) (int64, error) {
	return 1, nil
}
func (r *transferRepository) TransferDestinationBalance(_ context.Context, _ scope, _ domain.TransferLine, in domain.TransferInput, after float64, _ float64) (int64, error) {
	if in.ToLocationID != destinationLocation {
		panic("destination must use the canonical id")
	}
	r.destination = after
	return 1, nil
}
func (r *transferRepository) CreateTransferItem(context.Context, scope, string, domain.TransferLine, domain.LocationBalance) (int64, error) {
	return 1, nil
}
func (r *transferRepository) SaveMovement(_ context.Context, _ scope, m Movement) error {
	r.movements = append(r.movements, m)
	return nil
}
func (r *transferRepository) TransferIncomingMovement(context.Context, scope, domain.TransferInput, *string, domain.TransferLine, float64, string, string, domain.LocationBalance, float64, float64) (int64, error) {
	return 1, nil
}

func TestTransferKeepsDestinationStockWithNonCanonicalLocationID(t *testing.T) {
	repo := &transferRepository{balances: map[string]domain.LocationBalance{
		sourceLocation:      {Quantity: 10, Cost: 2},
		destinationLocation: {Quantity: 5, Cost: 4},
	}}
	stock := NewStock(repo, stockWork{}, noAudit{})
	_, err := stock.Transfer(context.Background(), scope{OrganizationID: "org-1", LocationID: sourceLocation}, domain.TransferInput{
		ToLocationID: strings.ToUpper(destinationLocation),
		Items:        []domain.TransferLine{{InventoryItemID: "item-1", Quantity: 3}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if repo.destination != 8 {
		t.Fatalf("destination must add to its existing stock: got %v, want 8", repo.destination)
	}
}

func TestTransferRejectsSameLocationWrittenDifferently(t *testing.T) {
	repo := &transferRepository{}
	stock := NewStock(repo, stockWork{}, noAudit{})
	_, err := stock.Transfer(context.Background(), scope{OrganizationID: "org-1", LocationID: sourceLocation}, domain.TransferInput{
		ToLocationID: strings.ToUpper(sourceLocation),
		Items:        []domain.TransferLine{{InventoryItemID: "item-1", Quantity: 3}},
	})
	if linkFailureCode(err) != "invalid_transfer" || repo.created {
		t.Fatalf("same location must be rejected before creating the transfer: %v", err)
	}
}

func TestTransferFailsWhenABalanceIsMissing(t *testing.T) {
	repo := &transferRepository{balances: map[string]domain.LocationBalance{sourceLocation: {Quantity: 10, Cost: 2}}}
	stock := NewStock(repo, stockWork{}, noAudit{})
	_, err := stock.Transfer(context.Background(), scope{OrganizationID: "org-1", LocationID: sourceLocation}, domain.TransferInput{
		ToLocationID: destinationLocation,
		Items:        []domain.TransferLine{{InventoryItemID: "item-1", Quantity: 3}},
	})
	if linkFailureCode(err) != "transfer_unavailable" || repo.destination != 0 {
		t.Fatalf("a missing destination balance must not be treated as zero: %v", err)
	}
}
