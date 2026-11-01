package domain

import (
	"math/big"
	"testing"
)

func rat(value string) *big.Rat { r, _ := new(big.Rat).SetString(value); return r }

func TestComputeBreaksDownChargedTotalExactly(t *testing.T) {
	lines := []SourceLine{
		{Description: "Lomo saltado", Quantity: rat("2"), UnitPrice: rat("35.90")},
		{Description: "Chicha morada", Quantity: rat("1"), UnitPrice: rat("8.50")},
		{Description: "Delivery", Quantity: rat("1"), UnitPrice: rat("5")},
	}
	mype := []TaxComponent{{Code: "igv", Name: "IGV", Rate: "0.080000"}, {Code: "ipm", Name: "IPM", Rate: "0.025000"}}
	out, err := Compute(lines, mype, rat("85.30"))
	if err != nil {
		t.Fatal(err)
	}
	if out.Total != "85.30" || len(out.Lines) != 3 || out.Lines[0].Total != "71.80" {
		t.Fatalf("totals: %+v", out)
	}
	net, tax := rat(out.NetTotal), rat(out.TaxTotal)
	if new(big.Rat).Add(net, tax).Cmp(rat("85.30")) != 0 {
		t.Fatalf("net + tax must equal the charged total: %s + %s", out.NetTotal, out.TaxTotal)
	}
	components := new(big.Rat).Add(rat(out.Taxes[0].Amount), rat(out.Taxes[1].Amount))
	if components.Cmp(tax) != 0 || out.Taxes[1].Rate != "0.025000" {
		t.Fatalf("components must add up to the tax: %+v", out.Taxes)
	}
	// 71.80 / 1.105 = 64.977… → 64.98 net, 6.82 tax.
	if out.Lines[0].NetAmount != "64.98" || out.Lines[0].TaxAmount != "6.82" {
		t.Fatalf("line breakdown: %+v", out.Lines[0])
	}
}

func TestComputeAbsorbsCentRoundingButRejectsRealMismatch(t *testing.T) {
	general := []TaxComponent{{Code: "general", Name: "IGV", Rate: "0.180000"}}
	lines := []SourceLine{{Description: "Menú", Quantity: rat("3"), UnitPrice: rat("3.33")}}
	out, err := Compute(lines, general, rat("10.00"))
	if err != nil || out.Total != "10.00" || out.Lines[0].Total != "10.00" {
		t.Fatalf("one cent difference is absorbed: %+v %v", out, err)
	}
	if _, err := Compute(lines, general, rat("12.00")); err != ErrTotalMismatch {
		t.Fatalf("a real mismatch must fail: %v", err)
	}
	if _, err := Compute(nil, general, rat("0")); err != ErrNoLines {
		t.Fatalf("no lines: %v", err)
	}
}

func TestComputeWithoutTax(t *testing.T) {
	out, err := Compute([]SourceLine{{Description: "Agua", Quantity: rat("1"), UnitPrice: rat("4")}}, []TaxComponent{{Code: "general", Name: "IVA", Rate: "0"}}, nil)
	if err != nil || out.TaxTotal != "0.00" || out.NetTotal != "4.00" || out.Taxes[0].Amount != "0.00" {
		t.Fatalf("zero rate: %+v %v", out, err)
	}
}

func TestRequestNormalize(t *testing.T) {
	r := Request{Customer: Customer{DocumentType: " dni ", DocumentNumber: " 4567 8901 ", Name: "  Ana   Pérez "}}
	r.Normalize()
	if r.DocumentType != TypeNone || r.Wanted() || r.Customer.DocumentType != "DNI" || r.Customer.DocumentNumber != "45678901" || r.Customer.Name != "Ana Pérez" {
		t.Fatalf("normalize: %+v", r)
	}
	r = Request{DocumentType: "receipt", Customer: Customer{DocumentNumber: "123"}}
	r.Normalize()
	if !r.Wanted() || r.Customer.DocumentNumber != "" {
		t.Fatalf("a number without type is dropped: %+v", r)
	}
}

func TestRetryDelayMinutes(t *testing.T) {
	for attempts, want := range map[int]int{1: 1, 2: 2, 3: 4, 7: 60, 20: 60} {
		if got := RetryDelayMinutes(attempts); got != want {
			t.Errorf("attempt %d: want %d, got %d", attempts, want, got)
		}
	}
}
