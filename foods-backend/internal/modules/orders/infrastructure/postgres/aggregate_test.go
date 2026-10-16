package postgres

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// fakeOrderRow imita pgx.Row: asigna en orden los valores de la fila y deja
// en nil los punteros cuyo valor es nil, como hace pgx con NULL.
type fakeOrderRow struct{ values []any }

func (r fakeOrderRow) Scan(dest ...any) error {
	if len(dest) != len(r.values) {
		return fmt.Errorf("scan got %d destinations, want %d", len(dest), len(r.values))
	}
	for i, d := range dest {
		v := r.values[i]
		switch target := d.(type) {
		case *string:
			*target = v.(string)
		case *[]string:
			*target = v.([]string)
		case **float64:
			if v == nil {
				*target = nil
			} else {
				f := v.(float64)
				*target = &f
			}
		}
	}
	return nil
}

func orderRowValues(lat, lng, km any) []any {
	return []any{
		"order-1", "D-001", "delivery", "nuevo", "", "Ana", "999888777", "Av. Arequipa 123", "Frente al parque",
		"", "", "", "40.00", "5.00", "45.00", "2026-10-16T12:00:00-05:00", "2026-10-16T12:00:00-05:00", "",
		"", "", []string{}, "",
		lat, lng, km,
	}
}

func TestOrderColumnsIncludeDeliveryPoint(t *testing.T) {
	for _, column := range []string{"delivery_latitude::float8", "delivery_longitude::float8", "delivery_distance_km::float8"} {
		if !strings.Contains(orderColumns, column) {
			t.Fatalf("orderColumns does not select %s", column)
		}
	}
}

func TestScanOrderMapsDeliveryPoint(t *testing.T) {
	o, err := scanOrder(fakeOrderRow{values: orderRowValues(-12.046374, -77.042793, 2.35)})
	if err != nil {
		t.Fatalf("scanOrder: %v", err)
	}
	if o.DeliveryLatitude == nil || *o.DeliveryLatitude != -12.046374 {
		t.Fatalf("latitude = %v", o.DeliveryLatitude)
	}
	if o.DeliveryLongitude == nil || *o.DeliveryLongitude != -77.042793 {
		t.Fatalf("longitude = %v", o.DeliveryLongitude)
	}
	if o.DeliveryDistanceKm == nil || *o.DeliveryDistanceKm != 2.35 {
		t.Fatalf("distance = %v", o.DeliveryDistanceKm)
	}
	body, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{`"deliveryLatitude":-12.046374`, `"deliveryLongitude":-77.042793`, `"deliveryDistanceKm":2.35`} {
		if !strings.Contains(string(body), fragment) {
			t.Fatalf("json %s missing %s", body, fragment)
		}
	}
}

func TestScanOrderWithoutDeliveryPointSerializesNull(t *testing.T) {
	o, err := scanOrder(fakeOrderRow{values: orderRowValues(nil, nil, nil)})
	if err != nil {
		t.Fatalf("scanOrder: %v", err)
	}
	if o.DeliveryLatitude != nil || o.DeliveryLongitude != nil || o.DeliveryDistanceKm != nil {
		t.Fatalf("expected nil delivery point, got %v %v %v", o.DeliveryLatitude, o.DeliveryLongitude, o.DeliveryDistanceKm)
	}
	body, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{`"deliveryLatitude":null`, `"deliveryLongitude":null`, `"deliveryDistanceKm":null`} {
		if !strings.Contains(string(body), fragment) {
			t.Fatalf("json %s missing %s", body, fragment)
		}
	}
}
