package contracts

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

type RestaurantBranding struct {
	LogoURL         *string  `json:"restaurantLogoUrl,omitempty"`
	VirtualMenuURLs []string `json:"virtualMenuUrls,omitempty"`
}
type BrandingReader interface {
	Branding(context.Context, requestscope.Scope) (RestaurantBranding, error)
}
