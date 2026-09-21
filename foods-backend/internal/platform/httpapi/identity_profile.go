package httpapi

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type myProfileView struct {
	FullName string `json:"fullName"`
	Email    string `json:"email"`
}

type myProfileInput struct {
	FullName        string `json:"fullName"`
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (a *API) getMyProfile(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var out myProfileView
	if err := a.db.QueryRow(r.Context(), `
		SELECT full_name,email
		FROM users
		WHERE id=$1 AND active
	`, s.UserID).Scan(&out.FullName, &out.Email); err != nil {
		fail(w, 404, "profile_not_found", "No pudimos cargar tu perfil.")
		return
	}
	writeJSON(w, 200, out)
}

func (a *API) updateMyProfile(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in myProfileInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_profile", "Revisa los datos enviados.")
		return
	}
	in.FullName = strings.TrimSpace(in.FullName)
	if in.FullName == "" || len(in.FullName) > 180 {
		fail(w, 400, "invalid_profile", "Ingresa un nombre válido.")
		return
	}
	if in.NewPassword != "" && len(in.NewPassword) < 8 {
		fail(w, 400, "invalid_password", "La nueva contraseña debe tener al menos 8 caracteres.")
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "profile_unavailable", "No pudimos actualizar tu perfil.")
		return
	}
	defer tx.Rollback(r.Context())

	var email, hash string
	if err = tx.QueryRow(r.Context(), `
		SELECT email,password_hash
		FROM users
		WHERE id=$1 AND active
		FOR UPDATE
	`, s.UserID).Scan(&email, &hash); errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "profile_not_found", "Tu cuenta ya no está disponible.")
		return
	} else if err != nil {
		fail(w, 503, "profile_unavailable", "No pudimos validar tu cuenta.")
		return
	}

	nextHash := hash
	if in.NewPassword != "" {
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.CurrentPassword)) != nil {
			fail(w, 409, "current_password_invalid", "La contraseña actual no es correcta.")
			return
		}
		generated, genErr := bcrypt.GenerateFromPassword([]byte(in.NewPassword), bcrypt.DefaultCost)
		if genErr != nil {
			fail(w, 500, "profile_unavailable", "No pudimos preparar la nueva contraseña.")
			return
		}
		nextHash = string(generated)
	}

	if _, err = tx.Exec(r.Context(), `
		UPDATE users
		SET full_name=$2,password_hash=$3,updated_at=now()
		WHERE id=$1
	`, s.UserID, in.FullName, nextHash); err != nil {
		fail(w, 503, "profile_unavailable", "No pudimos actualizar tu perfil.")
		return
	}

	if in.NewPassword != "" {
		if cookie, cookieErr := r.Cookie("foods_session"); cookieErr == nil && cookie.Value != "" {
			currentHash := sha256.Sum256([]byte(cookie.Value))
			if _, err = tx.Exec(r.Context(), `
				UPDATE sessions
				SET revoked_at=COALESCE(revoked_at,now())
				WHERE user_id=$1 AND revoked_at IS NULL AND token_hash<>$2
			`, s.UserID, currentHash[:]); err != nil {
				fail(w, 503, "profile_unavailable", "No pudimos cerrar las otras sesiones.")
				return
			}
		}
	}

	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "profile_unavailable", "No pudimos actualizar tu perfil.")
		return
	}
	a.audit(r, "user.profile_updated", "user", s.UserID)
	writeJSON(w, 200, myProfileView{FullName: in.FullName, Email: email})
}
