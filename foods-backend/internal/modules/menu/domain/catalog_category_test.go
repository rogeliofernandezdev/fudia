package domain

import "testing"

func TestNormalizeProductCategoryRequiresCategory(t *testing.T) {
	blank, malformed := "   ", "sin-categoria"
	for name, category := range map[string]*string{"missing": nil, "blank": &blank, "malformed": &malformed} {
		t.Run(name, func(t *testing.T) {
			if _, invalid := NormalizeProductCategory(ProductInput{CategoryID: category}); invalid == "" {
				t.Fatal("product without a valid category accepted")
			}
		})
	}
	padded := " 7f1c2a4e-3b6d-4c8e-9a10-2b3c4d5e6f70 "
	in, invalid := NormalizeProductCategory(ProductInput{CategoryID: &padded})
	if invalid != "" || in.CategoryID == nil || *in.CategoryID != "7f1c2a4e-3b6d-4c8e-9a10-2b3c4d5e6f70" {
		t.Fatalf("valid category rejected or not trimmed: %v %q", in.CategoryID, invalid)
	}
}
