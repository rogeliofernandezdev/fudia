package application

import (
	"context"

	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	einvoicing "github.com/foods-platform/foods-backend/internal/modules/einvoicing/contracts"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

// WithFiscal enables electronic documents at checkout. Without it, a payment
// that asks for a document is rejected and every other payment is unchanged.
func (a *Payments) WithFiscal(issuer einvoicing.Issuer) *Payments { a.fiscal = issuer; return a }

func fiscalWanted(in *domain.FiscalRequestInput) bool {
	return in != nil && in.DocumentType != "" && in.DocumentType != einvoicing.DocumentNone
}

// requestFiscal runs inside the payment transaction, after the payment is
// stored: a rejected document request rolls the whole payment back. One
// document per order, so it is requested by the payment that settles it.
func (a *Payments) requestFiscal(ctx context.Context, s scope, orderID string, in *domain.FiscalRequestInput, settles bool) (*domain.FiscalDocumentRef, error) {
	if !fiscalWanted(in) {
		return nil, nil
	}
	if !settles {
		return nil, paymentFailure(apperror.Conflict, "fiscal_requires_full_payment", "Elige el comprobante en el cobro que completa la cuenta.")
	}
	if a.fiscal == nil {
		return nil, paymentFailure(apperror.Conflict, "einvoicing_not_enabled", "La facturación electrónica no está activa para este local.")
	}
	ref, err := a.fiscal.Request(ctx, s, orderID, einvoicing.FiscalRequest{DocumentType: in.DocumentType, Customer: einvoicing.Customer{
		DocumentType: in.Customer.DocumentType, DocumentNumber: in.Customer.DocumentNumber, Name: in.Customer.Name, Address: in.Customer.Address,
	}})
	if err != nil {
		return nil, err
	}
	return &domain.FiscalDocumentRef{ID: ref.ID, DocumentType: ref.DocumentType, Label: ref.Label, Number: ref.Number, Status: ref.Status}, nil
}

// orderFiscalDocument tells reprints which document was delivered: the
// electronic one when the order has it, the payment ticket otherwise.
func (a *Payments) orderFiscalDocument(ctx context.Context, s scope, orderID string) (*domain.FiscalDocumentRef, error) {
	if a.fiscal == nil {
		return nil, nil
	}
	ref, err := a.fiscal.OrderDocument(ctx, s, orderID)
	if err != nil || ref == nil {
		return nil, err
	}
	return &domain.FiscalDocumentRef{ID: ref.ID, DocumentType: ref.DocumentType, Label: ref.Label, Number: ref.Number, Status: ref.Status}, nil
}

func (a *Payments) dispatchFiscal(ref *domain.FiscalDocumentRef) {
	if ref != nil && a.fiscal != nil {
		a.fiscal.Dispatch(ref.ID)
	}
}
