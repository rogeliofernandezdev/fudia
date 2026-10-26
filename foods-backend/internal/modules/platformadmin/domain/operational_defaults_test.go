package domain

import (
	"reflect"
	"testing"
)

func TestTenantOperationalDefaultsPreloadRestaurant(t *testing.T) {
	d := NewTenantOperationalDefaults()
	if err := d.Validate(); err != nil {
		t.Fatalf("defaults must be valid: %v", err)
	}
	if d.ProductCategory != "General" || d.ProductCategoryScope != "both" {
		t.Fatalf("unexpected category: %q scope %q", d.ProductCategory, d.ProductCategoryScope)
	}
	if d.ServiceMode != "dine_in" {
		t.Fatalf("initial service mode must be dine_in, got %q", d.ServiceMode)
	}
	if d.TableZone != "Principal" {
		t.Fatalf("tables must belong to Principal, got %q", d.TableZone)
	}
	want := []DefaultTable{{Name: "Mesa 1", Seats: 2, QREnabled: true}, {Name: "Mesa 2", Seats: 2, QREnabled: true}}
	if !reflect.DeepEqual(d.Tables, want) {
		t.Fatalf("tables=%v want=%v", d.Tables, want)
	}
	if !reflect.DeepEqual(d.TableNames(), []string{"Mesa 1", "Mesa 2"}) ||
		!reflect.DeepEqual(d.TableSeats(), []int32{2, 2}) ||
		!reflect.DeepEqual(d.TableQREnabled(), []bool{true, true}) {
		t.Fatalf("parallel SQL arrays out of sync: %v %v %v", d.TableNames(), d.TableSeats(), d.TableQREnabled())
	}
}

func TestTenantOperationalDefaultsRejectBrokenInvariants(t *testing.T) {
	cases := map[string]func(*TenantOperationalDefaults){
		"empty category":   func(d *TenantOperationalDefaults) { d.ProductCategory = " " },
		"category scope":   func(d *TenantOperationalDefaults) { d.ProductCategoryScope = "ingredient" },
		"service mode":     func(d *TenantOperationalDefaults) { d.ServiceMode = "delivery" },
		"zone":             func(d *TenantOperationalDefaults) { d.TableZone = "" },
		"seats":            func(d *TenantOperationalDefaults) { d.Tables[1].Seats = 0 },
		"table name":       func(d *TenantOperationalDefaults) { d.Tables[0].Name = "" },
		"duplicated table": func(d *TenantOperationalDefaults) { d.Tables[1].Name = d.Tables[0].Name },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			d := NewTenantOperationalDefaults()
			mutate(&d)
			if err := d.Validate(); err == nil {
				t.Fatal("broken defaults accepted")
			}
		})
	}
}
