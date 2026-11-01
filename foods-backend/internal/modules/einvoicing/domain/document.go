// Package domain holds the country-neutral model of electronic fiscal
// documents. Country rules live in jurisdictions; transport in providers.
package domain

import (
	"errors"
	"math/big"
	"strings"
)

const (
	TypeNone       = "none"
	TypeReceipt    = "receipt"
	TypeInvoice    = "invoice"
	TypeCreditNote = "credit_note"

	StatusPending   = "PENDING"
	StatusSending   = "SENDING"
	StatusAccepted  = "ACCEPTED"
	StatusObserved  = "OBSERVED"
	StatusRejected  = "REJECTED"
	StatusError     = "ERROR"
	StatusCancelled = "CANCELLED"

	EnvironmentTest       = "test"
	EnvironmentProduction = "production"

	// ModuleKey is the commercial module that entitles a company to issue.
	ModuleKey = "facturacion"
)

// Customer is the buyer as typed by the cashier. DocumentType uses the
// customer catalog values ("", DNI, CE, RUC, PASAPORTE); the jurisdiction maps
// them to fiscal codes.
type Customer struct {
	DocumentType   string `json:"documentType"`
	DocumentNumber string `json:"documentNumber"`
	Name           string `json:"name"`
	Address        string `json:"address"`
}

// Request is what checkout asks for: a document type and its customer.
type Request struct {
	DocumentType string   `json:"documentType"`
	Customer     Customer `json:"customer"`
}

func (r *Request) Normalize() {
	r.DocumentType = strings.TrimSpace(r.DocumentType)
	if r.DocumentType == "" {
		r.DocumentType = TypeNone
	}
	r.Customer.DocumentType = strings.ToUpper(strings.TrimSpace(r.Customer.DocumentType))
	r.Customer.DocumentNumber = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(r.Customer.DocumentNumber), " ", ""))
	r.Customer.Name = strings.Join(strings.Fields(r.Customer.Name), " ")
	r.Customer.Address = strings.Join(strings.Fields(r.Customer.Address), " ")
	if r.Customer.DocumentType == "" {
		r.Customer.DocumentNumber = ""
	}
}
func (r Request) Wanted() bool { return r.DocumentType != TypeNone }

// TaxComponent is a rate in force on the issue date (exact fraction, e.g. "0.080000").
type TaxComponent struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Rate string `json:"rate"`
}

// SourceLine is a charged line. Prices are what the customer paid, so the
// tax is always broken down from them.
type SourceLine struct {
	Description string
	// UnitCode follows UN/ECE rec 20: NIU for goods, ZZ for services.
	UnitCode  string
	Quantity  *big.Rat
	UnitPrice *big.Rat
}

type Line struct {
	No          int    `json:"no"`
	Description string `json:"description"`
	UnitCode    string `json:"unitCode"`
	Quantity    string `json:"quantity"`
	UnitPrice   string `json:"unitPrice"`
	NetAmount   string `json:"netAmount"`
	TaxAmount   string `json:"taxAmount"`
	Total       string `json:"total"`
}
type TaxAmount struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Rate   string `json:"rate"`
	Amount string `json:"amount"`
}
type Breakdown struct {
	Lines    []Line      `json:"lines"`
	Taxes    []TaxAmount `json:"taxes"`
	NetTotal string      `json:"netTotal"`
	TaxTotal string      `json:"taxTotal"`
	Total    string      `json:"total"`
}

var (
	ErrNoLines          = errors.New("fiscal document without lines")
	ErrTotalMismatch    = errors.New("fiscal document lines do not match the charged total")
	maxRoundingMismatch = big.NewRat(5, 100)
)

// Round2 rounds half away from zero to cents.
func Round2(value *big.Rat) *big.Rat {
	rounded, _ := new(big.Rat).SetString(value.FloatString(2))
	return rounded
}
func Money(value *big.Rat) string { return value.FloatString(2) }

// ParseAmount reads a decimal string; empty means zero.
func ParseAmount(value string) (*big.Rat, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return new(big.Rat), true
	}
	return new(big.Rat).SetString(value)
}

// Compute breaks the charged total down into net and tax with exact decimals.
// Each line's tax is total - total/(1+R), rounded to cents; components share
// the tax in proportion to their rate and the last one absorbs the remainder.
// When the rounded lines differ from the charged total by up to 5 cents, the
// last line absorbs the difference so the document matches the payment.
func Compute(source []SourceLine, components []TaxComponent, charged *big.Rat) (Breakdown, error) {
	var out Breakdown
	if len(source) == 0 {
		return out, ErrNoLines
	}
	rate := new(big.Rat)
	rates := make([]*big.Rat, len(components))
	for i, c := range components {
		r, ok := new(big.Rat).SetString(c.Rate)
		if !ok {
			r = new(big.Rat)
		}
		rates[i] = r
		rate.Add(rate, r)
	}
	divisor := new(big.Rat).Add(big.NewRat(1, 1), rate)

	totals := make([]*big.Rat, len(source))
	sum := new(big.Rat)
	for i, line := range source {
		totals[i] = Round2(new(big.Rat).Mul(line.Quantity, line.UnitPrice))
		sum.Add(sum, totals[i])
	}
	if charged != nil {
		diff := new(big.Rat).Sub(charged, sum)
		if new(big.Rat).Abs(diff).Cmp(maxRoundingMismatch) > 0 {
			return out, ErrTotalMismatch
		}
		last := len(totals) - 1
		totals[last] = new(big.Rat).Add(totals[last], diff)
		if totals[last].Sign() < 0 {
			return out, ErrTotalMismatch
		}
	}

	net, tax, total := new(big.Rat), new(big.Rat), new(big.Rat)
	for i, line := range source {
		lineNet := Round2(new(big.Rat).Quo(totals[i], divisor))
		lineTax := new(big.Rat).Sub(totals[i], lineNet)
		net.Add(net, lineNet)
		tax.Add(tax, lineTax)
		total.Add(total, totals[i])
		out.Lines = append(out.Lines, Line{
			No: i + 1, Description: line.Description, UnitCode: unitCode(line.UnitCode), Quantity: line.Quantity.FloatString(2), UnitPrice: Money(line.UnitPrice),
			NetAmount: Money(lineNet), TaxAmount: Money(lineTax), Total: Money(totals[i]),
		})
	}
	assigned := new(big.Rat)
	for i, c := range components {
		amount := new(big.Rat)
		switch {
		case i == len(components)-1:
			amount.Sub(tax, assigned)
		case rate.Sign() > 0:
			amount = Round2(new(big.Rat).Quo(new(big.Rat).Mul(tax, rates[i]), rate))
		}
		assigned.Add(assigned, amount)
		out.Taxes = append(out.Taxes, TaxAmount{Code: c.Code, Name: c.Name, Rate: rates[i].FloatString(6), Amount: Money(amount)})
	}
	out.NetTotal, out.TaxTotal, out.Total = Money(net), Money(tax), Money(total)
	return out, nil
}

func unitCode(value string) string {
	if value == "" {
		return "NIU"
	}
	return value
}

// RetryDelayMinutes is the wait before the next attempt after a temporary failure.
func RetryDelayMinutes(attempts int) int {
	delay := 1
	for i := 1; i < attempts && delay < 60; i++ {
		delay *= 2
	}
	if delay > 60 {
		delay = 60
	}
	return delay
}
