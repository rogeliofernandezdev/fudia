package application

import (
	"context"
	"crypto/sha256"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/identity/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"strings"
	"time"
)

var ErrNotFound = errors.New("identity not found")

// SessionRepository is the persistence port. It knows neither HTTP nor cookies.
type SessionRepository interface {
	Candidates(context.Context, string) ([]domain.Candidate, error)
	FirstLocation(context.Context, domain.Candidate) (string, error)
	CreateSession(context.Context, domain.Session) error
	RevokeSession(context.Context, []byte) error
	SessionScope(context.Context, []byte) (requestscope.Scope, error)
	IsPlatformAdmin(context.Context, string) (bool, error)
	HasPermission(context.Context, requestscope.Scope, string) (bool, error)
}
type Credentials interface {
	Matches(hash, password string) bool
	Token() (string, error)
}
type Service struct {
	repository  SessionRepository
	credentials Credentials
	now         func() time.Time
}

func New(repository SessionRepository, credentials Credentials) *Service {
	return &Service{repository, credentials, time.Now}
}
func (s *Service) Login(ctx context.Context, email, password string) (domain.LoginResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var out domain.LoginResult
	if email == "" || password == "" {
		return out, apperror.New(apperror.Invalid, "invalid_request", "Correo y contraseña son obligatorios.")
	}
	candidates, err := s.repository.Candidates(ctx, email)
	if err != nil {
		return out, unavailable("No pudimos validar la cuenta.")
	}
	matches := []domain.Candidate{}
	for _, c := range candidates {
		if s.credentials.Matches(c.Hash, password) {
			matches = append(matches, c)
		}
	}
	if len(matches) == 0 {
		return out, apperror.New(apperror.Unauthenticated, "invalid_credentials", "Correo o contraseña incorrectos.")
	}
	if len(matches) > 1 {
		return out, apperror.New(apperror.Conflict, "ambiguous_account", "Ese correo pertenece a más de una cuenta con la misma contraseña. Solicita al administrador diferenciar el acceso.")
	}
	c := matches[0]
	location, err := s.repository.FirstLocation(ctx, c)
	if errors.Is(err, ErrNotFound) {
		return out, apperror.New(apperror.Forbidden, "account_without_access", "Tu cuenta no tiene un rol activo en ningún local. Contacta a un administrador.")
	}
	if err != nil {
		return out, unavailable("No pudimos determinar tu local de trabajo.")
	}
	token, err := s.credentials.Token()
	if err != nil {
		return out, sessionError()
	}
	hash := sha256.Sum256([]byte(token))
	err = s.repository.CreateSession(ctx, domain.Session{Hash: hash[:], UserID: c.UserID, OrganizationID: c.OrganizationID, LocationID: location, ExpiresAt: s.now().Add(12 * time.Hour)})
	if err != nil {
		return out, sessionError()
	}
	return domain.LoginResult{Token: token, UserID: c.UserID, Name: c.Name, PlatformAdmin: c.PlatformAdmin, ExpiresIn: 43200}, nil
}
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	hash := sha256.Sum256([]byte(token))
	if s.repository.RevokeSession(ctx, hash[:]) != nil {
		return unavailable("No pudimos cerrar la sesión. Intenta nuevamente.")
	}
	return nil
}
func (s *Service) Authenticate(ctx context.Context, token string) (requestscope.Scope, error) {
	if token == "" {
		return requestscope.Scope{}, apperror.New(apperror.Unauthenticated, "unauthenticated", "Inicia sesión para continuar.")
	}
	hash := sha256.Sum256([]byte(token))
	scope, err := s.repository.SessionScope(ctx, hash[:])
	if errors.Is(err, ErrNotFound) {
		return scope, apperror.New(apperror.Unauthenticated, "session_expired", "Tu sesión venció. Ingresa nuevamente.")
	}
	if err != nil {
		return scope, unavailable("No pudimos validar la sesión.")
	}
	return scope, nil
}
func (s *Service) Authorize(ctx context.Context, scope requestscope.Scope, permission string) error {
	// Preserve the existing fallback: a failed platform lookup still checks tenant roles.
	if admin, err := s.repository.IsPlatformAdmin(ctx, scope.UserID); err == nil && admin {
		return nil
	}
	allowed, err := s.repository.HasPermission(ctx, scope, permission)
	if err != nil {
		return apperror.New(apperror.Unavailable, "permissions_unavailable", "No pudimos validar tus permisos.")
	}
	if !allowed {
		return apperror.New(apperror.Forbidden, "forbidden", "No tienes permisos para realizar esta acción.")
	}
	return nil
}
func unavailable(message string) error {
	return apperror.New(apperror.Unavailable, "session_unavailable", message)
}
func sessionError() error {
	return apperror.New(apperror.Internal, "session_error", "No se pudo iniciar la sesión.")
}
