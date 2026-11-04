// Package contracts is the public surface of EInvoicing for Billing.
package contracts

import (
	"context"

	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/domain"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

type FiscalRequest = domain.Request
type Customer = domain.Customer

const DocumentNone = domain.TypeNone

// DocumentRef identifies the document created for a payment. Number is the
// legal fiscal number (e.g. B001-45).
type DocumentRef struct {
	ID           string `json:"id"`
	DocumentType string `json:"documentType"`
	Label        string `json:"label"`
	Number       string `json:"number"`
	Status       string `json:"status"`
}

// Issuer is optional for Billing: without it, or with the company not
// entitled, a checkout that asks for a document is rejected.
type Issuer interface {
	// Request validates and stores the document inside the caller's
	// transaction; an error must roll the payment back.
	Request(ctx context.Context, s requestscope.Scope, orderID string, request FiscalRequest) (DocumentRef, error)
	// Dispatch hands the stored document to the asynchronous sender after commit.
	Dispatch(documentID string)
	// OrderDocument returns the document issued for an order of the session
	// location, or nil when the order was charged without one.
	OrderDocument(ctx context.Context, s requestscope.Scope, orderID string) (*DocumentRef, error)
}
