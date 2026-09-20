package httpapi

import (
	"net/http/httptest"
	"testing"
)

func TestPageParamsBounds(t *testing.T) {
	r := httptest.NewRequest("GET", "/v1/admin/products?page=2&pageSize=500", nil)
	page, size := pageParams(r)
	if page != 2 || size != 100 {
		t.Fatalf("expected 2/100, got %d/%d", page, size)
	}
}

func TestNormalizeProductRequiresIdentity(t *testing.T) {
	if _, invalid := normalizeProduct(productInput{Name: "Ceviche", Price: "45.00"}); invalid != "" {
		t.Fatalf("expected valid product, got %q", invalid)
	}
	if _, invalid := normalizeProduct(productInput{Name: "", Price: "45.00"}); invalid == "" {
		t.Fatal("expected invalid product without name")
	}
}

func TestNormalizeProductDefaultsToNone(t *testing.T) {
	in, invalid := normalizeProduct(productInput{Name: "Ceviche", Price: "45.00"})
	if invalid != "" {
		t.Fatalf("unexpected error %q", invalid)
	}
	if in.StockMode != "none" {
		t.Fatalf("expected none, got %q", in.StockMode)
	}
}

func TestNormalizeProductClearsQuotaWhenUnlimited(t *testing.T) {
	quota := 15
	in, invalid := normalizeProduct(productInput{Name: "Ceviche", Price: "45.00", StockMode: "none", DefaultDailyQuota: &quota})
	if invalid != "" {
		t.Fatalf("unexpected error %q", invalid)
	}
	if in.DefaultDailyQuota != nil {
		t.Fatal("expected quota cleared when product is always available")
	}
}

func TestNormalizeProductManualRequiresQuota(t *testing.T) {
	if _, invalid := normalizeProduct(productInput{Name: "Ceviche", Price: "45.00", StockMode: "manual"}); invalid == "" {
		t.Fatal("expected manual mode to require a quota")
	}
	negative := -1
	if _, invalid := normalizeProduct(productInput{Name: "Ceviche", Price: "45.00", StockMode: "manual", DefaultDailyQuota: &negative}); invalid == "" {
		t.Fatal("expected negative quota to be rejected")
	}
	zero := 0
	if _, invalid := normalizeProduct(productInput{Name: "Ceviche", Price: "45.00", StockMode: "manual", DefaultDailyQuota: &zero}); invalid == "" {
		t.Fatal("expected zero quota to be rejected")
	}
	quota := 15
	in, invalid := normalizeProduct(productInput{Name: "Ceviche", Price: "45.00", StockMode: "manual", DefaultDailyQuota: &quota})
	if invalid != "" {
		t.Fatalf("unexpected error %q", invalid)
	}
	if in.DefaultDailyQuota == nil || *in.DefaultDailyQuota != 15 {
		t.Fatal("expected quota preserved")
	}
}

func TestNormalizeProductRejectsInventoryModesAndUnknown(t *testing.T) {
	for _, mode := range []string{"linked", "recipe", "whatever"} {
		if _, invalid := normalizeProduct(productInput{Name: "Ceviche", Price: "45.00", StockMode: mode}); invalid == "" {
			t.Fatalf("expected %q to be rejected", mode)
		}
	}
}
