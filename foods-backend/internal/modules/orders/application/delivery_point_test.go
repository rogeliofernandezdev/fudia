package application

import (
	"context"
	"errors"
	"testing"

	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

type locationPoint struct {
	OrderRepository
	latitude, longitude *float64
}

func (r locationPoint) LocationPoint(context.Context, scope) (*float64, *float64, error) {
	return r.latitude, r.longitude, nil
}

func point(value float64) *float64 { return &value }

func TestDeliveryPointComputesDistanceFromLocation(t *testing.T) {
	service := New(locationPoint{latitude: point(-12.1219), longitude: point(-77.0297)}, nil, nil, nil)
	in := domain.OrderInput{Channel: "delivery", DeliveryLatitude: point(-12.1119), DeliveryLongitude: point(-77.0297)}
	if err := service.resolveDeliveryPoint(context.Background(), scope{}, &in); err != nil {
		t.Fatal(err)
	}
	if in.DeliveryDistanceKm == nil || *in.DeliveryDistanceKm < 1.1 || *in.DeliveryDistanceKm > 1.12 {
		t.Fatalf("distance must be computed by the server, got %v", in.DeliveryDistanceKm)
	}
}

func TestDeliveryPointKeepsPointWithoutLocationCoordinates(t *testing.T) {
	service := New(locationPoint{}, nil, nil, nil)
	in := domain.OrderInput{Channel: "delivery", DeliveryLatitude: point(-12.1), DeliveryLongitude: point(-77)}
	if err := service.resolveDeliveryPoint(context.Background(), scope{}, &in); err != nil {
		t.Fatal(err)
	}
	if in.DeliveryLatitude == nil || in.DeliveryDistanceKm != nil {
		t.Fatalf("point must be kept without distance: %+v", in)
	}
}

func TestDeliveryPointDroppedOutsideDelivery(t *testing.T) {
	service := New(locationPoint{}, nil, nil, nil)
	in := domain.OrderInput{Channel: "recojo", DeliveryLatitude: point(-12.1), DeliveryLongitude: point(-77)}
	if err := service.resolveDeliveryPoint(context.Background(), scope{}, &in); err != nil {
		t.Fatal(err)
	}
	if in.DeliveryLatitude != nil || in.DeliveryLongitude != nil {
		t.Fatalf("pickup orders must not store a delivery point: %+v", in)
	}
}

func TestDeliveryPointRejectsIncompleteOrInvalidCoordinates(t *testing.T) {
	service := New(locationPoint{}, nil, nil, nil)
	for _, in := range []domain.OrderInput{
		{Channel: "delivery", DeliveryLatitude: point(-12.1)},
		{Channel: "delivery", DeliveryLatitude: point(95), DeliveryLongitude: point(-77)},
	} {
		err := service.resolveDeliveryPoint(context.Background(), scope{}, &in)
		var failure *apperror.Error
		if !errors.As(err, &failure) || failure.Code != "invalid_delivery_point" {
			t.Fatalf("expected invalid_delivery_point, got %v", err)
		}
	}
}
