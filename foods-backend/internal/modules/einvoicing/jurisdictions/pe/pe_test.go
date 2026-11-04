package pe

import (
	"math/big"
	"testing"

	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/domain"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/ports"
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

func TestAmountInWords(t *testing.T) {
	for amount, want := range map[string]string{
		"80.30": "OCHENTA CON 30/100 SOLES", "85.30": "OCHENTA Y CINCO CON 30/100 SOLES", "0.50": "CERO CON 50/100 SOLES",
		"100.00": "CIEN CON 00/100 SOLES", "121.00": "CIENTO VEINTIUNO CON 00/100 SOLES", "1000": "MIL CON 00/100 SOLES",
		"21000.05": "VEINTIUN MIL CON 05/100 SOLES", "1500250.99": "UN MILLON QUINIENTOS MIL DOSCIENTOS CINCUENTA CON 99/100 SOLES",
	} {
		if got := AmountInWords(amount, "PEN"); got != want {
			t.Errorf("%s: %s", amount, got)
		}
	}
	if got := AmountInWords("10", "USD"); got != "DIEZ CON 00/100 DÓLARES AMERICANOS" {
		t.Errorf("usd: %s", got)
	}
}

func TestPrintout(t *testing.T) {
	p := New()
	issuer := ports.Issuer{TaxID: "20000000001", LegalName: "A MI MANERA E.I.R.L.", TradeName: "La Rinconada", Address: "Jr. Lima 123"}
	receipt := p.Printout(ports.PrintSource{DocumentType: domain.TypeReceipt, Series: "B001", Number: 1, IssueDate: "2026-11-03", Currency: "PEN",
		Issuer: issuer, Customer: domain.Customer{DocumentType: "DNI", DocumentNumber: "46351741", Name: "Ana"}, TaxTotal: "1.5300", Total: "10", DigestValue: "abc="})
	if receipt.Title != "BOLETA DE VENTA ELECTRÓNICA" || receipt.IssuerTaxIDLabel != "RUC" || receipt.AmountInWords != "SON: DIEZ CON 00/100 SOLES" {
		t.Fatalf("receipt printout: %+v", receipt)
	}
	if want := "20000000001|03|B001|1|1.53|10.00|2026-11-03|1|46351741|abc=|"; receipt.QRData != want {
		t.Fatalf("qr: %s", receipt.QRData)
	}
	if len(receipt.Legends) == 0 || receipt.Legends[0] != "Representación impresa de la Boleta de Venta Electrónica." {
		t.Fatalf("legends: %v", receipt.Legends)
	}
	anonymous := p.Printout(ports.PrintSource{DocumentType: domain.TypeInvoice, Series: "F001", Number: 7, IssueDate: "2026-11-03", Currency: "PEN", Issuer: issuer, TaxTotal: "0", Total: "5.5", DigestValue: "x"})
	if anonymous.Title != "FACTURA ELECTRÓNICA" || anonymous.QRData != "20000000001|01|F001|7|0.00|5.50|2026-11-03|0|-|x|" {
		t.Fatalf("invoice printout: %+v", anonymous)
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
