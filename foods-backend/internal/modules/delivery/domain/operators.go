package domain

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// Operadores de una central de Delivery Fudia: cada uno entra al panel con su
// correo y contraseña. Plataforma los da de alta con una contraseña temporal
// que el operador cambia en su primer ingreso.

const (
	// OperatorSessionHours es la duración de la sesión del panel.
	OperatorSessionHours = 12
	// OperatorMaxFailedLogins intentos fallidos seguidos bloquean el acceso.
	OperatorMaxFailedLogins = 5
	// OperatorLockMinutes es cuánto dura el bloqueo.
	OperatorLockMinutes = 15
	// OperatorPasswordMinLength y OperatorPasswordMaxBytes acotan la contraseña (bcrypt usa 72 bytes).
	OperatorPasswordMinLength = 8
	OperatorPasswordMaxBytes  = 72
	// TemporaryPasswordLength es el largo de la contraseña temporal generada.
	TemporaryPasswordLength = 12
)

// TemporaryPasswordAlphabet evita caracteres que se confunden al dictarlos (0/O, 1/l/I).
const TemporaryPasswordAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789"

// Operator es una persona de la central con acceso al panel.
type Operator struct {
	ID                 string `json:"id"`
	PartnerID          string `json:"partnerId"`
	FullName           string `json:"fullName"`
	Email              string `json:"email"`
	Active             bool   `json:"active"`
	MustChangePassword bool   `json:"mustChangePassword"`
	LastLoginAt        string `json:"lastLoginAt,omitempty"`
	CreatedAt          string `json:"createdAt"`
}

// OperatorInput crea o edita un operador; Active solo aplica al editar.
type OperatorInput struct {
	FullName string `json:"fullName"`
	Email    string `json:"email"`
	Active   *bool  `json:"active,omitempty"`
}

// OperatorCredentials devuelve la contraseña temporal una sola vez, al crearla.
type OperatorCredentials struct {
	Operator          Operator `json:"operator"`
	TemporaryPassword string   `json:"temporaryPassword"`
}

// OperatorSession es la sesión vigente de un operador y su central.
type OperatorSession struct {
	OperatorID         string
	OperatorName       string
	Email              string
	MustChangePassword bool
	PartnerID          string
	PartnerName        string
	PartnerActive      bool
}

// OperatorLogin es lo que el inicio de sesión necesita leer del operador.
type OperatorLogin struct {
	ID           string
	PasswordHash string
	Active       bool
	Locked       bool
}

// LoginInput y PasswordChangeInput son los cuerpos del inicio de sesión y del cambio de contraseña.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type PasswordChangeInput struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

var operatorEmailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// NormalizeEmail deja el correo como se guarda: sin espacios y en minúsculas.
func NormalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

// NormalizeOperator limpia nombre y correo; devuelve el problema a mostrar si no son válidos.
func NormalizeOperator(in OperatorInput) (OperatorInput, string) {
	in.FullName = strings.Join(strings.Fields(in.FullName), " ")
	in.Email = NormalizeEmail(in.Email)
	if n := utf8.RuneCountInString(in.FullName); n < 2 || n > 80 {
		return in, "Escribe el nombre del operador (2 a 80 caracteres)."
	}
	if len(in.Email) > 180 || !operatorEmailPattern.MatchString(in.Email) {
		return in, "Escribe un correo electrónico válido."
	}
	return in, ""
}

// OperatorPasswordProblem valida una contraseña elegida por el operador.
func OperatorPasswordProblem(password string) string {
	if utf8.RuneCountInString(password) < OperatorPasswordMinLength {
		return "La contraseña debe tener al menos 8 caracteres."
	}
	if len(password) > OperatorPasswordMaxBytes {
		return "La contraseña es demasiado larga."
	}
	return ""
}

// TemporaryPassword arma la contraseña temporal a partir de bytes aleatorios.
// Descarta los bytes que sesgarían el reparto entre caracteres; ok=false si no
// alcanzaron para el largo completo.
func TemporaryPassword(random []byte) (string, bool) {
	size := len(TemporaryPasswordAlphabet)
	limit := 256 - 256%size
	var out strings.Builder
	for _, value := range random {
		if int(value) >= limit {
			continue
		}
		out.WriteByte(TemporaryPasswordAlphabet[int(value)%size])
		if out.Len() == TemporaryPasswordLength {
			return out.String(), true
		}
	}
	return "", false
}
