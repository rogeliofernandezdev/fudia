// Package contracts publica el tipo de atención de las mesas para los módulos
// que deciden el flujo de un QR (Concierge) sin importar el dominio de Service.
package contracts

import "github.com/foods-platform/foods-backend/internal/modules/service/domain"

type ServiceType = domain.ServiceType

const (
	ServiceTypeTable   = domain.ServiceTypeTable
	ServiceTypeCounter = domain.ServiceTypeCounter
	// ServiceTypeDelivery: enlace de delivery del local (pide delivery o recojo).
	ServiceTypeDelivery = domain.ServiceTypeDelivery
)
