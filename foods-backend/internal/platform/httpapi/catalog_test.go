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

func TestNormalizeProductDefaultsPreparedAndNone(t *testing.T) {
	in, invalid := normalizeProduct(productInput{Name: "Ceviche", Price: "45.00"})
	if invalid != "" {
		t.Fatalf("unexpected error %q", invalid)
	}
	if in.ProductType != "prepared" {
		t.Fatalf("expected prepared, got %q", in.ProductType)
	}
	if in.QuantityControl != "none" {
		t.Fatalf("expected none, got %q", in.QuantityControl)
	}
}

func TestNormalizeProductAcceptsProductTypes(t *testing.T) {
	for _, productType := range []string{"prepared", "retail"} {
		in, invalid := normalizeProduct(productInput{Name: "Producto", Price: "10.00", ProductType: productType})
		if invalid != "" {
			t.Fatalf("expected %q to be valid, got %q", productType, invalid)
		}
		if in.ProductType != productType {
			t.Fatalf("expected %q, got %q", productType, in.ProductType)
		}
	}
}

func TestNormalizeProductRejectsUnknownProductType(t *testing.T) {
	if _, invalid := normalizeProduct(productInput{Name: "Producto", Price: "10.00", ProductType: "ingredient"}); invalid == "" {
		t.Fatal("expected unknown product type to be rejected")
	}
}

func TestNormalizeProductAcceptsQuantityControls(t *testing.T) {
	for _, control := range []string{"none", "portions", "inventory"} {
		in, invalid := normalizeProduct(productInput{Name: "Producto", Price: "10.00", QuantityControl: control})
		if invalid != "" {
			t.Fatalf("expected %q to be valid, got %q", control, invalid)
		}
		if in.QuantityControl != control {
			t.Fatalf("expected %q, got %q", control, in.QuantityControl)
		}
	}
}

func TestNormalizeProductRejectsUnknownQuantityControl(t *testing.T) {
	if _, invalid := normalizeProduct(productInput{Name: "Producto", Price: "10.00", QuantityControl: "manual"}); invalid == "" {
		t.Fatal("expected legacy manual control to be rejected")
	}
}
