package domain

import (
	"fmt"
	"math"
	"sort"
)

// FeeTier cobra Fee por entregas hasta MaxDistanceKm desde el local.
type FeeTier struct {
	MaxDistanceKm float64
	Fee           float64
}

const (
	MaxFeeTiers         = 10
	MaxDeliveryKm       = 50.0
	MaxDeliveryFee      = 999.0
	earthRadiusKm       = 6371.0088
	distancePrecisionKm = 100.0
)

func ValidCoordinates(latitude, longitude float64) bool {
	return !math.IsNaN(latitude) && !math.IsNaN(longitude) &&
		latitude >= -90 && latitude <= 90 && longitude >= -180 && longitude <= 180
}

// DistanceKm es la distancia en línea recta (haversine) redondeada a 10 m.
// El mismo valor redondeado se muestra al cliente y se compara con los rangos.
func DistanceKm(fromLat, fromLng, toLat, toLng float64) float64 {
	rad := math.Pi / 180
	dLat := (toLat - fromLat) * rad
	dLng := (toLng - fromLng) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(fromLat*rad)*math.Cos(toLat*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	km := 2 * earthRadiusKm * math.Asin(math.Min(1, math.Sqrt(a)))
	return math.Round(km*distancePrecisionKm) / distancePrecisionKm
}

// SortedTiers devuelve una copia ordenada por distancia ascendente.
func SortedTiers(tiers []FeeTier) []FeeTier {
	out := append([]FeeTier(nil), tiers...)
	sort.Slice(out, func(i, j int) bool { return out[i].MaxDistanceKm < out[j].MaxDistanceKm })
	return out
}

// QuoteFee elige el primer rango que cubre la distancia; false si ninguno.
func QuoteFee(tiers []FeeTier, distanceKm float64) (FeeTier, bool) {
	for _, tier := range SortedTiers(tiers) {
		if distanceKm <= tier.MaxDistanceKm {
			return tier, true
		}
	}
	return FeeTier{}, false
}

// CoverageKm es el mayor límite configurado (0 sin rangos).
func CoverageKm(tiers []FeeTier) float64 {
	coverage := 0.0
	for _, tier := range tiers {
		coverage = math.Max(coverage, tier.MaxDistanceKm)
	}
	return coverage
}

// ValidateFeeTiers devuelve un mensaje para el usuario o "" si son válidos.
func ValidateFeeTiers(tiers []FeeTier) string {
	if len(tiers) > MaxFeeTiers {
		return fmt.Sprintf("Configura como máximo %d rangos de distancia.", MaxFeeTiers)
	}
	seen := map[float64]bool{}
	for _, tier := range tiers {
		if math.IsNaN(tier.MaxDistanceKm) || tier.MaxDistanceKm <= 0 || tier.MaxDistanceKm > MaxDeliveryKm {
			return fmt.Sprintf("Cada rango debe cubrir entre 0.01 y %.0f km.", MaxDeliveryKm)
		}
		if math.IsNaN(tier.Fee) || tier.Fee < 0 || tier.Fee > MaxDeliveryFee {
			return fmt.Sprintf("El costo de envío debe estar entre 0 y %.0f.", MaxDeliveryFee)
		}
		if !roundedTo(tier.MaxDistanceKm) || !roundedTo(tier.Fee) {
			return "Usa como máximo dos decimales en distancias y costos."
		}
		if seen[tier.MaxDistanceKm] {
			return "No repitas el mismo límite de distancia en dos rangos."
		}
		seen[tier.MaxDistanceKm] = true
	}
	return ""
}

func roundedTo(value float64) bool {
	return math.Abs(value*100-math.Round(value*100)) < 1e-6
}
