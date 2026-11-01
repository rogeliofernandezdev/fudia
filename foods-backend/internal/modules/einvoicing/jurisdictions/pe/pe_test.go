package pe

import (
	"math/big"
	"testing"

	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/domain"
)

func TestValidRUC(t *testing.T) {
	for _, ruc := range []string{"20100047218", "10467793549", "20601030013"} {
		if !ValidRUC(ruc) {
			t.Errorf("%s must be valid", ruc)
		}
	}
	for _, ruc := range []string{"20100047219", "2010004721", "abcdefghijk", ""} {
		if ValidRUC(ruc) {
			t.Errorf("%s must be invalid", ruc)
		}
	}
}

func TestValidateIssue(t *testing.T) {
	p := New()
	small, large := big.NewRat(85, 1), big.NewRat(70001, 100)
	cases := []struct {
		name, docType string
		total         *big.Rat
		customer      domain.Customer
		field         string
	}{
		{"anonymous small receipt", domain.TypeReceipt, small, domain.Customer{}, ""},
		{"anonymous receipt above 700", domain.TypeReceipt, large, domain.Customer{}, "documentNumber"},
		{"identified receipt above 700", domain.TypeReceipt, large, domain.Customer{DocumentType: "DNI", DocumentNumber: "45678901", Name: "Ana Pérez"}, ""},
		{"receipt with bad DNI", domain.TypeReceipt, small, domain.Customer{DocumentType: "DNI", DocumentNumber: "4567", Name: "Ana"}, "documentNumber"},
		{"receipt document without name", domain.TypeReceipt, small, domain.Customer{DocumentType: "DNI", DocumentNumber: "45678901"}, "name"},
		{"invoice", domain.TypeInvoice, small, domain.Customer{DocumentType: "RUC", DocumentNumber: "20100047218", Name: "Empresa SAC"}, ""},
		{"invoice with DNI", domain.TypeInvoice, small, domain.Customer{DocumentType: "DNI", DocumentNumber: "45678901", Name: "Ana"}, "documentNumber"},
		{"invoice with bad check digit", domain.TypeInvoice, small, domain.Customer{DocumentType: "RUC", DocumentNumber: "20100047219", Name: "Empresa SAC"}, "documentNumber"},
		{"invoice without legal name", domain.TypeInvoice, small, domain.Customer{DocumentType: "RUC", DocumentNumber: "20100047218"}, "name"},
		{"unknown type", "ticket", small, domain.Customer{}, "documentType"},
	}
	for _, c := range cases {
		err := p.ValidateIssue(c.docType, c.total, c.customer)
		if (c.field == "") != (err == nil) || (err != nil && err.Field != c.field) {
			t.Errorf("%s: want field %q, got %v", c.name, c.field, err)
		}
	}
}

func TestSeriesAndCodes(t *testing.T) {
	p := New()
	if !p.ValidSeries(domain.TypeInvoice, "F001") || p.ValidSeries(domain.TypeInvoice, "B001") || !p.ValidSeries(domain.TypeReceipt, "B0A2") || p.ValidSeries(domain.TypeReceipt, "B01") {
		t.Fatal("series rules")
	}
	if p.DefaultSeries(domain.TypeReceipt, 2) != "B002" || p.DefaultSeries(domain.TypeInvoice, 1) != "F001" || p.FormatNumber("F001", 123) != "F001-123" {
		t.Fatal("default series and number format")
	}
	if p.FiscalCode(domain.TypeReceipt) != "03" || p.CustomerFiscalCode("ruc") != "6" || p.CustomerFiscalCode("") != "0" {
		t.Fatal("fiscal codes")
	}
}
