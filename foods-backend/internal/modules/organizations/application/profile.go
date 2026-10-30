package application

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/menu/contracts"
	orgcontracts "github.com/foods-platform/foods-backend/internal/modules/organizations/contracts"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"time"
)

type ProfileRecord struct {
	FoundedOn    string
	LogoImageID  *string
	MenuImageIDs []string
}
type RestaurantProfileInput struct {
	FoundedOn    string   `json:"foundedOn"`
	LogoImageID  *string  `json:"logoImageId"`
	MenuImageIDs []string `json:"menuImageIds"`
}
type RestaurantProfile struct {
	FoundedOn    string                      `json:"foundedOn"`
	Logo         *contracts.RestaurantImage  `json:"logo"`
	MenuPages    []contracts.RestaurantImage `json:"menuPages"`
	MaxMenuPages int                         `json:"maxMenuPages"`
}
type ProfileRepository interface {
	Read(context.Context, scope) (ProfileRecord, error)
	Save(context.Context, scope, RestaurantProfileInput) error
	PlatformTarget(context.Context, string, string) (scope, error)
}
type RestaurantProfiles struct {
	repo  ProfileRepository
	media contracts.RestaurantMedia
}

func NewRestaurantProfiles(repo ProfileRepository, media contracts.RestaurantMedia) *RestaurantProfiles {
	return &RestaurantProfiles{repo, media}
}
func (a *RestaurantProfiles) Target(ctx context.Context, s scope, organizationID string) (scope, error) {
	if organizationID == "" {
		return s, nil
	}
	target, err := a.repo.PlatformTarget(ctx, s.UserID, organizationID)
	if err != nil {
		return scope{}, organizationFailure(apperror.Forbidden, "profile_forbidden", "No puedes administrar este restaurante.")
	}
	return target, nil
}
func (a *RestaurantProfiles) Read(ctx context.Context, s scope) (RestaurantProfile, error) {
	out := RestaurantProfile{MenuPages: []contracts.RestaurantImage{}, MaxMenuPages: 20}
	record, err := a.repo.Read(ctx, s)
	if err != nil {
		return out, organizationFailure(apperror.Unavailable, "profile_unavailable", "No pudimos cargar la identidad del restaurante.")
	}
	out.FoundedOn = record.FoundedOn
	if record.LogoImageID != nil {
		image, err := a.media.Read(ctx, s, *record.LogoImageID)
		if err != nil {
			return out, err
		}
		out.Logo = &image
	}
	for _, id := range record.MenuImageIDs {
		image, err := a.media.Read(ctx, s, id)
		if err != nil {
			return out, err
		}
		out.MenuPages = append(out.MenuPages, image)
	}
	return out, nil
}
func (a *RestaurantProfiles) Save(ctx context.Context, s scope, in RestaurantProfileInput) error {
	if in.FoundedOn != "" {
		if _, err := time.Parse("2006-01-02", in.FoundedOn); err != nil {
			return organizationFailure(apperror.Invalid, "invalid_profile", "La fecha de fundación no es válida.")
		}
	}
	if len(in.MenuImageIDs) > 20 {
		return organizationFailure(apperror.Invalid, "invalid_profile", "La carta virtual admite hasta 20 imágenes.")
	}
	seen := map[string]bool{}
	for _, id := range in.MenuImageIDs {
		if id == "" || seen[id] {
			return organizationFailure(apperror.Invalid, "invalid_profile", "Las imágenes de la carta no deben repetirse.")
		}
		seen[id] = true
	}
	if err := a.repo.Save(ctx, s, in); err != nil {
		return organizationFailure(apperror.Conflict, "profile_conflict", "No pudimos guardar el perfil. Verifica que las imágenes estén confirmadas y pertenezcan al restaurante.")
	}
	return nil
}
func (a *RestaurantProfiles) Authorize(ctx context.Context, s scope, in contracts.MediaMetadata) (contracts.MediaAuthorization, error) {
	return a.media.Authorize(ctx, s, in)
}
func (a *RestaurantProfiles) Complete(ctx context.Context, s scope, id string) (contracts.RestaurantImage, error) {
	return a.media.Complete(ctx, s, id)
}
func (a *RestaurantProfiles) Image(ctx context.Context, s scope, id string) (contracts.RestaurantImage, error) {
	return a.media.Read(ctx, s, id)
}

// Reports resolve only their logo; printing a ticket must not sign the entire menu.
func (a *RestaurantProfiles) ReportBranding(ctx context.Context, s scope) (orgcontracts.RestaurantBranding, error) {
	out := orgcontracts.RestaurantBranding{VirtualMenuURLs: []string{}}
	record, err := a.repo.Read(ctx, s)
	if err != nil {
		return out, organizationFailure(apperror.Unavailable, "profile_unavailable", "No pudimos cargar el logo del restaurante.")
	}
	if record.LogoImageID == nil {
		return out, nil
	}
	image, err := a.media.Read(ctx, s, *record.LogoImageID)
	if err == nil && image.Status == "READY" {
		out.LogoURL = image.URL
	}
	return out, err
}
func (a *RestaurantProfiles) Branding(ctx context.Context, s scope) (orgcontracts.RestaurantBranding, error) {
	out := orgcontracts.RestaurantBranding{VirtualMenuURLs: []string{}}
	profile, err := a.Read(ctx, s)
	if err != nil {
		return out, err
	}
	if profile.Logo != nil && profile.Logo.Status == "READY" {
		out.LogoURL = profile.Logo.URL
	}
	for _, image := range profile.MenuPages {
		if image.DeliveryURL != nil && image.Status == "READY" {
			out.VirtualMenuURLs = append(out.VirtualMenuURLs, *image.DeliveryURL)
		}
	}
	return out, nil
}
