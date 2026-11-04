// Package ports declares the two country extension points of EInvoicing:
// a Jurisdiction (fiscal rules, no network) and a Provider (transport to an
// authorized issuer, no business rules). Both are registered in cmd/api.
package ports

import (
	"context"
	"math/big"

	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/domain"
)

type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// CustomerDocumentOption is a customer identity document accepted by the
// country, with the pattern the frontend uses for immediate feedback.
type CustomerDocumentOption struct {
	Value   string `json:"value"`
	Label   string `json:"label"`
	Pattern string `json:"pattern"`
}

// IssueError is a rule violation the cashier can fix; Field points to the input.
type IssueError struct {
	Field   string
	Message string
}

func (e *IssueError) Error() string { return e.Message }

type Jurisdiction interface {
	Country() string
	// DocumentTypes lists the documents offered at checkout (neutral values).
	DocumentTypes() []Option
	CustomerDocumentTypes() []CustomerDocumentOption
	// InvoiceCustomerDocumentType is the business tax ID an invoice requires.
	InvoiceCustomerDocumentType() string
	// IdentityThreshold is the receipt total above which the buyer must be
	// identified, in the profile currency; empty when the country has none.
	IdentityThreshold() string
	ValidateIssue(documentType string, total *big.Rat, customer domain.Customer) *IssueError
	// ValidateIssuer checks the company data printed on every document.
	ValidateIssuer(issuer Issuer) *IssueError
	ValidSeries(documentType, series string) bool
	DefaultSeries(documentType string, position int) string
	FormatNumber(series string, number int) string
	DocumentLabel(documentType string) string
	FiscalCode(documentType string) string
	CustomerFiscalCode(customerDocumentType string) string
	// Printout returns the legal content of the printed representation
	// (title, amount in words, QR, legends); clients only lay it out.
	Printout(source PrintSource) Printout
}

// PrintSource is the stored, signed document a printout is made from.
type PrintSource struct {
	DocumentType string
	Series       string
	Number       int
	IssueDate    string
	Currency     string
	Issuer       Issuer
	Customer     domain.Customer
	TaxTotal     string
	Total        string
	DigestValue  string
}

// Printout is the printed representation of an electronic document. It
// exists once the document is signed, before the authority answers.
type Printout struct {
	Title            string   `json:"title"`
	Issuer           Issuer   `json:"issuer"`
	IssuerTaxIDLabel string   `json:"issuerTaxIdLabel"`
	AmountInWords    string   `json:"amountInWords"`
	QRData           string   `json:"qrData"`
	Legends          []string `json:"legends"`
	// Notice marks documents without fiscal value (test environment).
	Notice string `json:"notice,omitempty"`
}

type Issuer struct {
	TaxID     string `json:"taxId"`
	LegalName string `json:"legalName"`
	TradeName string `json:"tradeName"`
	Address   string `json:"address"`
}

// Document is what a provider transmits. Codes come from the jurisdiction so
// adapters stay free of country rules.
type Document struct {
	ID                 string
	Country            string
	DocumentType       string
	FiscalCode         string
	Series             string
	Number             int
	FormattedNumber    string
	IssueDate          string
	Currency           string
	Environment        string
	Issuer             Issuer
	Customer           domain.Customer
	CustomerFiscalCode string
	Breakdown          domain.Breakdown
	// Credentials are decrypted only for the send; never logged or stored.
	Credentials map[string]string
}

// Result statuses: ACCEPTED, OBSERVED or REJECTED. A returned error means a
// temporary failure (network, provider down) and the document is retried.
type Result struct {
	Status    string
	Reference string
	Code      string
	Message   string
	PDFURL    string
	XMLURL    string
	CDRURL    string
	// DigestValue is the signature digest printed on the receipt (hash).
	DigestValue string
	// Files are the legal artifacts to keep (signed XML, CDR).
	Files []File
}

// File is an artifact returned by the provider. Kind is "xml" or "cdr".
type File struct {
	Kind        string
	Name        string
	ContentType string
	Content     []byte
}

// CredentialField describes one credential a provider needs; the settings
// screen is built from this list, so a new provider needs no new screen.
type CredentialField struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Type     string `json:"type"` // text, secret or file (base64 content)
	Accept   string `json:"accept,omitempty"`
	Required bool   `json:"required"`
}

// CredentialError is a credential the administrator must fix.
type CredentialError struct{ Message string }

func (e *CredentialError) Error() string { return e.Message }

type Provider interface {
	Key() string
	Label() string
	// TestOnly providers (the simulator) cannot be used in production.
	TestOnly() bool
	CredentialFields() []CredentialField
	// RequiresCredentials tells whether the environment cannot work without
	// stored credentials (SUNAT beta has public test credentials).
	RequiresCredentials(environment string) bool
	// ValidateCredentials checks a full credential set and returns non-secret
	// metadata to display (e.g. certificate holder and expiry).
	ValidateCredentials(environment string, values map[string]string) (map[string]string, error)
	// Issue must be idempotent on Document.ID.
	Issue(ctx context.Context, document Document) (Result, error)
}

// Total converts a breakdown total to *big.Rat for rule checks.
func Total(value string) *big.Rat {
	r, ok := new(big.Rat).SetString(value)
	if !ok {
		return new(big.Rat)
	}
	return r
}
