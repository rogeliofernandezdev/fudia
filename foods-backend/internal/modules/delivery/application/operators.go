package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"sync"
	"time"

	"github.com/foods-platform/foods-backend/internal/modules/delivery/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

// Acceso de las centrales de Delivery Fudia a su panel: operadores con correo y
// contraseña, sesiones de 12 horas y bloqueo tras intentos fallidos seguidos.
// Plataforma da de alta a los operadores y restablece sus contraseñas.

var ErrDuplicateEmail = errors.New("delivery operator email already registered")

var (
	errOperatorNotFound   = failure(apperror.NotFound, "delivery_operator_not_found", "El operador no existe.")
	errOperatorEmailTaken = failure(apperror.Conflict, "delivery_operator_email_taken", "Ese correo ya tiene acceso a una central.")
	errInvalidCredentials = failure(apperror.Unauthenticated, "invalid_credentials", "Correo o contraseña incorrectos.")
	errOperatorLocked     = failure(apperror.Forbidden, "delivery_operator_locked", "Demasiados intentos fallidos. Vuelve a intentarlo en 15 minutos.")
	errOperatorInactive   = failure(apperror.Forbidden, "delivery_operator_inactive", "Tu acceso al panel está desactivado. Comunícate con Fudia.")
	errOperatorSession    = failure(apperror.Unauthenticated, "session_expired", "Tu sesión venció. Ingresa nuevamente.")
	errPasswordRequired   = failure(apperror.Forbidden, "password_change_required", "Cambia tu contraseña temporal para continuar.")
)

// OperatorRepository guarda operadores, sesiones y el registro de sus acciones.
type OperatorRepository interface {
	PartnerExists(ctx context.Context, partnerID string) (bool, error)
	ListOperators(ctx context.Context, partnerID string) ([]domain.Operator, error)
	// CreateOperator devuelve ErrDuplicateEmail si el correo ya existe.
	CreateOperator(ctx context.Context, partnerID string, in domain.OperatorInput, passwordHash string) (domain.Operator, error)
	// UpdateOperator devuelve ErrNotFound o ErrDuplicateEmail.
	UpdateOperator(ctx context.Context, partnerID, id string, in domain.OperatorInput) (domain.Operator, error)
	// ResetOperatorPassword guarda la contraseña temporal y quita el bloqueo.
	ResetOperatorPassword(ctx context.Context, partnerID, id, passwordHash string) (domain.Operator, error)
	// RevokeOperatorSessions cierra las sesiones del operador salvo la indicada (puede ser nil).
	RevokeOperatorSessions(ctx context.Context, operatorID string, keep []byte) error
	OperatorForLogin(ctx context.Context, email string) (domain.OperatorLogin, error)
	// RecordFailedLogin suma un intento fallido y bloquea al llegar al máximo.
	RecordFailedLogin(ctx context.Context, operatorID string, max int, lock time.Duration) error
	// StartOperatorSession reinicia los intentos fallidos, marca el ingreso, crea la
	// sesión y la devuelve con su central, en una sola consulta.
	StartOperatorSession(ctx context.Context, operatorID string, tokenHash []byte, expiresAt time.Time) (domain.OperatorSession, error)
	OperatorSession(ctx context.Context, tokenHash []byte) (domain.OperatorSession, error)
	RevokeOperatorSession(ctx context.Context, tokenHash []byte) error
	OperatorPasswordHash(ctx context.Context, operatorID string) (string, error)
	// SetOperatorPassword guarda la contraseña elegida por el operador y quita la obligación de cambiarla.
	SetOperatorPassword(ctx context.Context, operatorID, passwordHash string) error
	RecordOperatorAction(ctx context.Context, partnerID, operatorID, action, outcome, orderCode string) error
	// PurgeOperatorSessions borra las sesiones vencidas o revocadas antes de la fecha indicada.
	PurgeOperatorSessions(ctx context.Context, before time.Time) (int64, error)
}

// Passwords cifra y compara contraseñas y genera tokens de sesión.
type Passwords interface {
	Hash(password string) (string, error)
	Matches(hash, password string) bool
	Token() (string, error)
}

// Operators administra el acceso de las centrales a su panel.
type Operators struct {
	repo      OperatorRepository
	work      unitofwork.Manager
	passwords Passwords
	now       func() time.Time
	random    func([]byte) (int, error)
	dummy     sync.Once
	dummyHash string
}

func NewOperators(repo OperatorRepository, work unitofwork.Manager, passwords Passwords) *Operators {
	return &Operators{repo: repo, work: work, passwords: passwords, now: time.Now, random: rand.Read}
}

// OperatorLoginResult es la sesión creada: el token viaja solo en la cookie.
type OperatorLoginResult struct {
	Token     string
	ExpiresIn int
	Session   domain.OperatorSession
}

func operatorError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrNotFound):
		return errOperatorNotFound
	case errors.Is(err, ErrDuplicateEmail):
		return errOperatorEmailTaken
	default:
		return unavailable("delivery_operators_unavailable", "No pudimos guardar el acceso de la central.", err)
	}
}

func (a *Operators) partner(ctx context.Context, partnerID string) error {
	if !uuidPattern.MatchString(partnerID) {
		return errPartnerNotFound
	}
	exists, err := a.repo.PartnerExists(ctx, partnerID)
	if err != nil {
		return unavailable("delivery_operators_unavailable", "No pudimos cargar los operadores de la central.", err)
	}
	if !exists {
		return errPartnerNotFound
	}
	return nil
}

// List devuelve los operadores de una central.
func (a *Operators) List(ctx context.Context, partnerID string) ([]domain.Operator, error) {
	if err := a.partner(ctx, partnerID); err != nil {
		return nil, err
	}
	out, err := a.repo.ListOperators(ctx, partnerID)
	if err != nil {
		return nil, unavailable("delivery_operators_unavailable", "No pudimos cargar los operadores de la central.", err)
	}
	return out, nil
}

func (a *Operators) temporaryPassword() (string, string, error) {
	random := make([]byte, 64)
	for range 4 {
		if _, err := a.random(random); err != nil {
			return "", "", err
		}
		if password, ok := domain.TemporaryPassword(random); ok {
			hash, err := a.passwords.Hash(password)
			return password, hash, err
		}
	}
	return "", "", errors.New("temporary password: not enough random bytes")
}

// Create da de alta a un operador con una contraseña temporal que se muestra una sola vez.
func (a *Operators) Create(ctx context.Context, partnerID string, in domain.OperatorInput) (domain.OperatorCredentials, error) {
	if err := a.partner(ctx, partnerID); err != nil {
		return domain.OperatorCredentials{}, err
	}
	in, problem := domain.NormalizeOperator(in)
	if problem != "" {
		return domain.OperatorCredentials{}, failure(apperror.Invalid, "invalid_delivery_operator", problem)
	}
	password, hash, err := a.temporaryPassword()
	if err != nil {
		return domain.OperatorCredentials{}, operatorError(err)
	}
	operator, err := a.repo.CreateOperator(ctx, partnerID, in, hash)
	if err != nil {
		return domain.OperatorCredentials{}, operatorError(err)
	}
	return domain.OperatorCredentials{Operator: operator, TemporaryPassword: password}, nil
}

// Update cambia nombre, correo o estado; desactivar cierra sus sesiones.
func (a *Operators) Update(ctx context.Context, partnerID, id string, in domain.OperatorInput) (domain.Operator, error) {
	if !uuidPattern.MatchString(partnerID) {
		return domain.Operator{}, errPartnerNotFound
	}
	if !uuidPattern.MatchString(id) {
		return domain.Operator{}, errOperatorNotFound
	}
	in, problem := domain.NormalizeOperator(in)
	if problem != "" {
		return domain.Operator{}, failure(apperror.Invalid, "invalid_delivery_operator", problem)
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return domain.Operator{}, operatorError(err)
	}
	defer tx.Rollback(ctx)
	out, err := a.repo.UpdateOperator(tx.Context(), partnerID, id, in)
	if err == nil && !out.Active {
		err = a.repo.RevokeOperatorSessions(tx.Context(), id, nil)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return domain.Operator{}, operatorError(err)
	}
	return out, nil
}

// ResetPassword genera una contraseña temporal nueva, quita el bloqueo y cierra sus sesiones.
func (a *Operators) ResetPassword(ctx context.Context, partnerID, id string) (domain.OperatorCredentials, error) {
	if !uuidPattern.MatchString(partnerID) {
		return domain.OperatorCredentials{}, errPartnerNotFound
	}
	if !uuidPattern.MatchString(id) {
		return domain.OperatorCredentials{}, errOperatorNotFound
	}
	password, hash, err := a.temporaryPassword()
	if err != nil {
		return domain.OperatorCredentials{}, operatorError(err)
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return domain.OperatorCredentials{}, operatorError(err)
	}
	defer tx.Rollback(ctx)
	operator, err := a.repo.ResetOperatorPassword(tx.Context(), partnerID, id, hash)
	if err == nil {
		err = a.repo.RevokeOperatorSessions(tx.Context(), id, nil)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return domain.OperatorCredentials{}, operatorError(err)
	}
	return domain.OperatorCredentials{Operator: operator, TemporaryPassword: password}, nil
}

func tokenHash(token string) []byte {
	hash := sha256.Sum256([]byte(token))
	return hash[:]
}

// matchesDummy iguala el tiempo de respuesta cuando el correo no existe.
func (a *Operators) matchesDummy(password string) {
	a.dummy.Do(func() { a.dummyHash, _ = a.passwords.Hash("fudia-delivery-operator-dummy") })
	if a.dummyHash != "" {
		a.passwords.Matches(a.dummyHash, password)
	}
}

// Login valida correo y contraseña y abre una sesión de 12 horas. Cinco
// intentos fallidos seguidos bloquean el acceso durante 15 minutos.
func (a *Operators) Login(ctx context.Context, in domain.LoginInput) (OperatorLoginResult, error) {
	email := domain.NormalizeEmail(in.Email)
	if email == "" || in.Password == "" {
		return OperatorLoginResult{}, failure(apperror.Invalid, "invalid_request", "Correo y contraseña son obligatorios.")
	}
	fail := func(err error) (OperatorLoginResult, error) {
		return OperatorLoginResult{}, unavailable("delivery_login_unavailable", "No pudimos validar tu acceso. Intenta nuevamente.", err)
	}
	operator, err := a.repo.OperatorForLogin(ctx, email)
	if errors.Is(err, ErrNotFound) {
		a.matchesDummy(in.Password)
		return OperatorLoginResult{}, errInvalidCredentials
	}
	if err != nil {
		return fail(err)
	}
	if operator.Locked {
		return OperatorLoginResult{}, errOperatorLocked
	}
	if len(in.Password) > domain.OperatorPasswordMaxBytes || !a.passwords.Matches(operator.PasswordHash, in.Password) {
		if err = a.repo.RecordFailedLogin(ctx, operator.ID, domain.OperatorMaxFailedLogins, domain.OperatorLockMinutes*time.Minute); err != nil {
			return fail(err)
		}
		return OperatorLoginResult{}, errInvalidCredentials
	}
	if !operator.Active {
		return OperatorLoginResult{}, errOperatorInactive
	}
	token, err := a.passwords.Token()
	if err != nil {
		return fail(err)
	}
	session, err := a.repo.StartOperatorSession(ctx, operator.ID, tokenHash(token), a.now().Add(domain.OperatorSessionHours*time.Hour))
	if errors.Is(err, ErrNotFound) {
		// Lo desactivaron entre la validación y el inicio de sesión.
		return OperatorLoginResult{}, errOperatorInactive
	}
	if err != nil {
		return fail(err)
	}
	return OperatorLoginResult{Token: token, ExpiresIn: domain.OperatorSessionHours * 3600, Session: session}, nil
}

// Authenticate resuelve la sesión de la cookie: operador activo y su central.
func (a *Operators) Authenticate(ctx context.Context, token string) (domain.OperatorSession, error) {
	if token == "" {
		return domain.OperatorSession{}, failure(apperror.Unauthenticated, "unauthenticated", "Inicia sesión para continuar.")
	}
	session, err := a.repo.OperatorSession(ctx, tokenHash(token))
	if errors.Is(err, ErrNotFound) {
		return session, errOperatorSession
	}
	if err != nil {
		return session, unavailable("delivery_login_unavailable", "No pudimos validar tu sesión.", err)
	}
	return session, nil
}

// RequirePassword impide operar el panel con la contraseña temporal.
func RequirePassword(session domain.OperatorSession) error {
	if session.MustChangePassword {
		return errPasswordRequired
	}
	return nil
}

// Logout revoca la sesión de la cookie.
func (a *Operators) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	if err := a.repo.RevokeOperatorSession(ctx, tokenHash(token)); err != nil {
		return unavailable("delivery_logout_unavailable", "No pudimos cerrar la sesión. Intenta nuevamente.", err)
	}
	return nil
}

// ChangePassword cambia la contraseña del operador de la sesión y cierra sus
// otras sesiones; la actual sigue abierta.
func (a *Operators) ChangePassword(ctx context.Context, token string, in domain.PasswordChangeInput) (domain.OperatorSession, error) {
	session, err := a.Authenticate(ctx, token)
	if err != nil {
		return session, err
	}
	fail := func(err error) (domain.OperatorSession, error) {
		return domain.OperatorSession{}, unavailable("delivery_password_unavailable", "No pudimos cambiar la contraseña.", err)
	}
	current, err := a.repo.OperatorPasswordHash(ctx, session.OperatorID)
	if err != nil {
		return fail(err)
	}
	if len(in.CurrentPassword) > domain.OperatorPasswordMaxBytes || !a.passwords.Matches(current, in.CurrentPassword) {
		return domain.OperatorSession{}, failure(apperror.Invalid, "invalid_current_password", "La contraseña actual no es correcta.")
	}
	if problem := domain.OperatorPasswordProblem(in.NewPassword); problem != "" {
		return domain.OperatorSession{}, failure(apperror.Invalid, "invalid_password", problem)
	}
	if in.NewPassword == in.CurrentPassword {
		return domain.OperatorSession{}, failure(apperror.Invalid, "invalid_password", "La contraseña nueva debe ser distinta de la actual.")
	}
	hash, err := a.passwords.Hash(in.NewPassword)
	if err != nil {
		return fail(err)
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return fail(err)
	}
	defer tx.Rollback(ctx)
	err = a.repo.SetOperatorPassword(tx.Context(), session.OperatorID, hash)
	if err == nil {
		err = a.repo.RevokeOperatorSessions(tx.Context(), session.OperatorID, tokenHash(token))
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return fail(err)
	}
	session.MustChangePassword = false
	return session, nil
}

// Record guarda qué operador hizo una acción del panel. No interrumpe la
// acción ya aplicada si falla: el registro es informativo.
func (a *Operators) Record(ctx context.Context, session domain.OperatorSession, action, outcome, orderCode string) {
	_ = a.repo.RecordOperatorAction(ctx, session.PartnerID, session.OperatorID, action, outcome, orderCode)
}

// PurgeSessions borra las sesiones de operadores vencidas o revocadas hace más
// de 30 días; la validación de cada petición no las necesita y la tabla no crece.
func (a *Operators) PurgeSessions(ctx context.Context) (int64, error) {
	return a.repo.PurgeOperatorSessions(ctx, a.now().AddDate(0, 0, -domain.OperatorSessionRetentionDays))
}

// PanelPartner es la central de la sesión, como la usa el despacho.
func PanelPartnerOf(session domain.OperatorSession) PanelPartner {
	return PanelPartner{ID: session.PartnerID, Name: session.PartnerName, Active: session.PartnerActive}
}
