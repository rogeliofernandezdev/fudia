package sunat

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/domain"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/ports"
)

func rat(value string) *big.Rat { r, _ := new(big.Rat).SetString(value); return r }

func sampleDocument(t *testing.T, fiscalCode, series string, number int, customer domain.Customer, customerCode string, components []domain.TaxComponent) ports.Document {
	t.Helper()
	breakdown, err := domain.Compute([]domain.SourceLine{
		{Description: "Lomo saltado", UnitCode: "NIU", Quantity: rat("2"), UnitPrice: rat("35.90")},
		{Description: "Chicha morada", UnitCode: "NIU", Quantity: rat("1"), UnitPrice: rat("8.50")},
		{Description: "Delivery", UnitCode: "ZZ", Quantity: rat("1"), UnitPrice: rat("5.00")},
	}, components, rat("85.30"))
	if err != nil {
		t.Fatal(err)
	}
	return ports.Document{
		ID: fmt.Sprintf("test-%d", number), Country: "PE", FiscalCode: fiscalCode, Series: series, Number: number,
		FormattedNumber: fmt.Sprintf("%s-%d", series, number), IssueDate: time.Now().Format("2006-01-02"), Currency: "PEN",
		Environment: domain.EnvironmentTest, Issuer: ports.Issuer{TaxID: "20000000001", LegalName: "FUDIA PRUEBAS S.A.C.", TradeName: "Fudia", Address: "Av. Larco 123, Miraflores"},
		Customer: customer, CustomerFiscalCode: customerCode, Breakdown: breakdown, Credentials: map[string]string{},
	}
}

// Sends real documents to SUNAT beta. Run with SUNAT_BETA=1.
func TestSUNATBetaAcceptsInvoiceAndReceipt(t *testing.T) {
	if os.Getenv("SUNAT_BETA") != "1" {
		t.Skip("set SUNAT_BETA=1 to send to SUNAT beta")
	}
	p := New()
	general := []domain.TaxComponent{{Code: "general", Name: "IGV", Rate: "0.180000"}}
	mype := []domain.TaxComponent{{Code: "igv", Name: "IGV", Rate: "0.080000"}, {Code: "ipm", Name: "IPM", Rate: "0.025000"}}
	number := int(time.Now().Unix() % 90000000)
	cases := []struct {
		name string
		doc  ports.Document
	}{
		{"factura", sampleDocument(t, "01", "F001", number, domain.Customer{DocumentType: "RUC", DocumentNumber: "20100047218", Name: "BANCO DE CREDITO DEL PERU"}, "6", general)},
		{"boleta con DNI", sampleDocument(t, "03", "B001", number, domain.Customer{DocumentType: "DNI", DocumentNumber: "45678901", Name: "Ana Perez"}, "1", general)},
		{"boleta anónima", sampleDocument(t, "03", "B001", number+1, domain.Customer{}, "0", general)},
		{"boleta régimen MYPE", sampleDocument(t, "03", "B001", number+2, domain.Customer{}, "0", mype)},
	}
	for i, c := range cases {
		if i > 0 {
			time.Sleep(3 * time.Second)
		}
		result, err := p.Issue(context.Background(), c.doc)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		t.Logf("%s: %s %s %s", c.name, result.Status, result.Code, result.Message)
		if result.Status != domain.StatusAccepted && result.Status != domain.StatusObserved {
			t.Errorf("%s not accepted: %s %s", c.name, result.Code, result.Message)
		}
	}
}
