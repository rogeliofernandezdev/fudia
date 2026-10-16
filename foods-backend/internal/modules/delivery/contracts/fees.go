package contracts

import "github.com/foods-platform/foods-backend/internal/modules/delivery/domain"

// Tarifas por distancia publicadas para Concierge y futuros canales de delivery.
type FeeTier = domain.FeeTier

var (
	ValidCoordinates = domain.ValidCoordinates
	DistanceKm       = domain.DistanceKm
	SortedTiers      = domain.SortedTiers
	QuoteFee         = domain.QuoteFee
	CoverageKm       = domain.CoverageKm
	ValidateFeeTiers = domain.ValidateFeeTiers
)
