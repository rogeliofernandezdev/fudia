package httpapi

import (
	platformdomain "github.com/foods-platform/foods-backend/internal/modules/platformadmin/domain"
)

type platformWhatsAppChannelInput = platformdomain.PlatformWhatsAppChannelInput

var validatePlatformWhatsAppChannel = platformdomain.ValidatePlatformWhatsAppChannel
