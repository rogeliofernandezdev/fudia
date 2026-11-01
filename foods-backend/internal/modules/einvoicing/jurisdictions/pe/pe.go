// Package pe holds Peru's fiscal rules for electronic receipts and invoices.
package pe

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/domain"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/ports"
)

type Peru struct{}

func New() Peru { return Peru{} }

var (
	_                 ports.Jurisdiction = Peru{}
	identityThreshold                    = big.NewRat(700, 1)
	customerPatterns                     = map[string]*regexp.Regexp{
		"DNI": regexp.MustCompile(`^\d{8}$`), "CE": regexp.MustCompile(`^[A-Z0-9]{9,12}$`),
		"RUC": regexp.MustCompile(`^\d{11}$`), "PASAPORTE": regexp.MustCompile(`^[A-Z0-9]{5,12}$`),
	}
	seriesPatterns = map[string]*regexp.Regexp{
		domain.TypeInvoice: regexp.MustCompile(`^F[A-Z0-9]{3}$`), domain.TypeReceipt: regexp.MustCompile(`^B[A-Z0-9]{3}$`),
	}
)

func (Peru) Country() string { return "PE" }
func (Peru) DocumentTypes() []ports.Option {
	return []ports.Option{{Value: domain.TypeReceipt, Label: "Boleta"}, {Value: domain.TypeInvoice, Label: "Factura"}}
}
func (Peru) CustomerDocumentTypes() []ports.CustomerDocumentOption {
	return []ports.CustomerDocumentOption{
		{Value: "DNI", Label: "DNI", Pattern: `^\d{8}$`},
		{Value: "RUC", Label: "RUC", Pattern: `^\d{11}$`},
		{Value: "CE", Label: "Carné de extranjería", Pattern: `^[A-Za-z0-9]{9,12}$`},
		{Value: "PASAPORTE", Label: "Pasaporte", Pattern: `^[A-Za-z0-9]{5,12}$`},
	}
}
func (Peru) IdentityThreshold() string           { return "700.00" }
func (Peru) InvoiceCustomerDocumentType() string { return "RUC" }

// ValidRUC checks the 11-digit RUC check digit (weights 5,4,3,2,7,6,5,4,3,2).
func ValidRUC(value string) bool {
	if !customerPatterns["RUC"].MatchString(value) {
		return false
	}
	weights := []int{5, 4, 3, 2, 7, 6, 5, 4, 3, 2}
	sum := 0
	for i, w := range weights {
		sum += int(value[i]-'0') * w
	}
	check := 11 - sum%11
	if check == 10 {
		check = 0
	} else if check == 11 {
		check = 1
	}
	return int(value[10]-'0') == check
}

func (p Peru) ValidateIssue(documentType string, total *big.Rat, c domain.Customer) *ports.IssueError {
	switch documentType {
	case domain.TypeInvoice:
		if c.DocumentType != "RUC" || !ValidRUC(c.DocumentNumber) {
			return &ports.IssueError{Field: "documentNumber", Message: "La factura requiere un RUC válido de 11 dígitos."}
		}
		if c.Name == "" || len([]rune(c.Name)) > 200 {
			return &ports.IssueError{Field: "name", Message: "Ingresa la razón social del cliente."}
		}
	case domain.TypeReceipt:
		if c.DocumentType == "" {
			if total.Cmp(identityThreshold) > 0 {
				return &ports.IssueError{Field: "documentNumber", Message: "Las boletas mayores a S/ 700 requieren el documento del cliente."}
			}
			return nil
		}
		pattern, ok := customerPatterns[c.DocumentType]
		if !ok || !pattern.MatchString(c.DocumentNumber) || (c.DocumentType == "RUC" && !ValidRUC(c.DocumentNumber)) {
			return &ports.IssueError{Field: "documentNumber", Message: "Revisa el número del documento del cliente."}
		}
		if c.Name == "" || len([]rune(c.Name)) > 200 {
			return &ports.IssueError{Field: "name", Message: "Ingresa el nombre del cliente."}
		}
	default:
		return &ports.IssueError{Field: "documentType", Message: "Elige boleta o factura."}
	}
	return nil
}
func (Peru) ValidateIssuer(issuer ports.Issuer) *ports.IssueError {
	if !ValidRUC(strings.TrimSpace(issuer.TaxID)) {
		return &ports.IssueError{Field: "taxId", Message: "La identificación fiscal de Mi restaurante debe ser un RUC válido de 11 dígitos."}
	}
	if strings.TrimSpace(issuer.LegalName) == "" {
		return &ports.IssueError{Field: "legalName", Message: "Completa la razón social en Mi restaurante."}
	}
	return nil
}
func (Peru) ValidSeries(documentType, series string) bool {
	pattern, ok := seriesPatterns[documentType]
	return ok && pattern.MatchString(series)
}

// DefaultSeries suggests B001/F001 for the first location, B002/F002 next.
func (Peru) DefaultSeries(documentType string, position int) string {
	prefix := map[string]string{domain.TypeInvoice: "F", domain.TypeReceipt: "B"}[documentType]
	if prefix == "" {
		return ""
	}
	return fmt.Sprintf("%s%03d", prefix, position%1000)
}
func (Peru) FormatNumber(series string, number int) string {
	return fmt.Sprintf("%s-%d", series, number)
}
func (Peru) DocumentLabel(documentType string) string {
	return map[string]string{domain.TypeInvoice: "Factura", domain.TypeReceipt: "Boleta", domain.TypeCreditNote: "Nota de crédito"}[documentType]
}
func (Peru) FiscalCode(documentType string) string {
	return map[string]string{domain.TypeInvoice: "01", domain.TypeReceipt: "03", domain.TypeCreditNote: "07"}[documentType]
}
func (Peru) CustomerFiscalCode(customerDocumentType string) string {
	code, ok := map[string]string{"DNI": "1", "CE": "4", "RUC": "6", "PASAPORTE": "7"}[strings.ToUpper(customerDocumentType)]
	if !ok {
		return "0"
	}
	return code
}
