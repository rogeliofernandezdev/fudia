package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/domain"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/ports"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

// Cipher encrypts tenant credentials; the key lives outside the database.
type Cipher interface {
	Seal(plaintext, aad []byte) (int, []byte, []byte, error)
	Open(version int, nonce, ciphertext, aad []byte) ([]byte, error)
}

// SealedCredentials is what PostgreSQL stores: ciphertext plus non-secret
// metadata (certificate holder and expiry) that the settings screen shows.
type SealedCredentials struct {
	KeyVersion int
	Nonce      []byte
	Ciphertext []byte
	Metadata   map[string]string
	UpdatedAt  string
}

type CredentialsInput struct {
	Provider string            `json:"provider"`
	Values   map[string]string `json:"values"`
}

type CredentialStatus struct {
	Configured bool                    `json:"configured"`
	Metadata   map[string]string       `json:"metadata"`
	UpdatedAt  string                  `json:"updatedAt"`
	Fields     []ports.CredentialField `json:"fields"`
	Required   bool                    `json:"required"`
}

// credentialsAAD binds the ciphertext to its tenant, profile and provider.
func credentialsAAD(organizationID, profileID, provider string) []byte {
	return []byte("einvoicing|" + organizationID + "|" + profileID + "|" + provider)
}

func openCredentials(cipher Cipher, sealed SealedCredentials, organizationID, profileID, provider string) (map[string]string, error) {
	if cipher == nil {
		return nil, errors.New("credential encryption not configured")
	}
	plain, err := cipher.Open(sealed.KeyVersion, sealed.Nonce, sealed.Ciphertext, credentialsAAD(organizationID, profileID, provider))
	if err != nil {
		return nil, err
	}
	values := map[string]string{}
	return values, json.Unmarshal(plain, &values)
}

// SaveCredentials merges the new values over the stored ones (a blank secret
// keeps the previous one), validates the full set with the provider and
// stores it encrypted. Secrets are never returned.
func (a *Settings) SaveCredentials(ctx context.Context, s scope, profileID string, in CredentialsInput) (SettingsView, error) {
	in.Provider = strings.TrimSpace(in.Provider)
	provider, ok := a.registry.Provider(in.Provider)
	if !ok {
		return SettingsView{}, failure(apperror.Invalid, "invalid_einvoicing_settings", "Elige un proveedor válido.")
	}
	profile, err := a.repo.ProfileSetting(ctx, s, profileID)
	if errors.Is(err, ErrNotFound) {
		return SettingsView{}, failure(apperror.NotFound, "profile_not_found", "No encontramos el perfil fiscal.")
	}
	if err != nil {
		return SettingsView{}, failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos validar el perfil fiscal.")
	}
	if a.cipher == nil {
		return SettingsView{}, failure(apperror.Unavailable, "einvoicing_credentials_unavailable", "El cifrado de credenciales no está configurado en el servidor.")
	}
	allowed := map[string]bool{}
	for _, field := range provider.CredentialFields() {
		allowed[field.Key] = true
	}
	values := map[string]string{}
	if sealed, found, err := a.repo.Credentials(ctx, s, profileID, in.Provider); err != nil {
		return SettingsView{}, failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos leer las credenciales guardadas.")
	} else if found {
		if values, err = openCredentials(a.cipher, sealed, s.OrganizationID, profileID, in.Provider); err != nil {
			return SettingsView{}, failure(apperror.Unavailable, "einvoicing_credentials_unavailable", "No pudimos descifrar las credenciales guardadas.")
		}
	}
	for key, value := range in.Values {
		if !allowed[key] {
			return SettingsView{}, failure(apperror.Invalid, "invalid_einvoicing_credentials", "Hay un dato de credenciales que el proveedor no reconoce.")
		}
		if strings.TrimSpace(value) != "" {
			values[key] = value
		}
	}
	environment := profile.Environment
	if environment == "" {
		environment = domain.EnvironmentTest
	}
	metadata, err := provider.ValidateCredentials(environment, values)
	var credentialErr *ports.CredentialError
	if errors.As(err, &credentialErr) {
		return SettingsView{}, failure(apperror.Invalid, "invalid_einvoicing_credentials", credentialErr.Message)
	}
	if err != nil {
		return SettingsView{}, failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos validar las credenciales.")
	}
	plain, err := json.Marshal(values)
	if err != nil {
		return SettingsView{}, failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos preparar las credenciales.")
	}
	version, nonce, ciphertext, err := a.cipher.Seal(plain, credentialsAAD(s.OrganizationID, profileID, in.Provider))
	if err != nil {
		return SettingsView{}, failure(apperror.Unavailable, "einvoicing_credentials_unavailable", "No pudimos cifrar las credenciales.")
	}
	if err := a.repo.SaveCredentials(ctx, s, profileID, in.Provider, SealedCredentials{KeyVersion: version, Nonce: nonce, Ciphertext: ciphertext, Metadata: metadata}); err != nil {
		return SettingsView{}, failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos guardar las credenciales.")
	}
	_ = a.audit.Record(ctx, s, "einvoicing.credentials.updated", "fiscal_profile", profileID)
	return a.View(ctx, s)
}

func (a *Settings) credentialStatus(ctx context.Context, s scope, profileID string, provider ports.Provider, environment string) (CredentialStatus, error) {
	status := CredentialStatus{Metadata: map[string]string{}, Fields: provider.CredentialFields(), Required: provider.RequiresCredentials(environment)}
	sealed, found, err := a.repo.Credentials(ctx, s, profileID, provider.Key())
	if err != nil || !found {
		return status, err
	}
	status.Configured, status.Metadata, status.UpdatedAt = true, sealed.Metadata, sealed.UpdatedAt
	return status, nil
}
