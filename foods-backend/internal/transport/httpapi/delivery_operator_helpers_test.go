package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Acceso de los operadores de una central al panel de reparto, compartido por
// las pruebas de integración del panel y del motorizado.

type operatorAccess interface {
	CreateOperator(http.ResponseWriter, *http.Request)
	PartnerLogin(http.ResponseWriter, *http.Request)
	PartnerPassword(http.ResponseWriter, *http.Request)
	OperatorReady(http.Handler) http.Handler
}

const partnerCookie = "fudia_partner_session"

// operatorRequest arma una petición con alcance de plataforma, valores de ruta y cookie opcional.
func operatorRequest(method, body string, path map[string]string, subject scope, session string) *http.Request {
	req := httptest.NewRequest(method, "/", bytes.NewReader([]byte(body)))
	for key, value := range path {
		req.SetPathValue(key, value)
	}
	if session != "" {
		req.AddCookie(&http.Cookie{Name: partnerCookie, Value: session})
	}
	return req.WithContext(context.WithValue(req.Context(), scopeKey{}, subject))
}

// sessionCookieOf devuelve el valor de la cookie de sesión que fijó la respuesta.
func sessionCookieOf(rec *httptest.ResponseRecorder) string {
	for _, c := range rec.Result().Cookies() {
		if c.Name == partnerCookie && c.MaxAge > 0 {
			return c.Value
		}
	}
	return ""
}

// createOperator da de alta un operador y devuelve su id y la contraseña temporal.
func createOperator(t *testing.T, d operatorAccess, subject scope, partnerID, name, email string) (string, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	d.CreateOperator(rec, operatorRequest("POST", fmt.Sprintf(`{"fullName":%q,"email":%q}`, name, email), map[string]string{"id": partnerID}, subject, ""))
	if rec.Code != 201 {
		t.Fatalf("create operator: %d %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Operator          struct{ ID string }
		TemporaryPassword string
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || out.Operator.ID == "" || len(out.TemporaryPassword) != 12 {
		t.Fatalf("operator credentials: %s", rec.Body.String())
	}
	return out.Operator.ID, out.TemporaryPassword
}

// operatorLogin inicia sesión y devuelve la respuesta (la cookie va en ella).
func operatorLogin(d operatorAccess, email, password string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	d.PartnerLogin(rec, operatorRequest("POST", fmt.Sprintf(`{"email":%q,"password":%q}`, email, password), nil, scope{}, ""))
	return rec
}

// changeOperatorPassword reemplaza la contraseña temporal desde la sesión.
func changeOperatorPassword(d operatorAccess, session, current, next string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	d.PartnerPassword(rec, operatorRequest("POST", fmt.Sprintf(`{"currentPassword":%q,"newPassword":%q}`, current, next), nil, scope{}, session))
	return rec
}

// operatorSession deja a la central con un operador listo para usar el panel y devuelve su sesión.
func operatorSession(t *testing.T, d operatorAccess, subject scope, partnerID, email string) string {
	t.Helper()
	_, temporary := createOperator(t, d, subject, partnerID, "Operador de prueba", email)
	login := operatorLogin(d, email, temporary)
	session := sessionCookieOf(login)
	if login.Code != 200 || session == "" {
		t.Fatalf("operator login: %d %s", login.Code, login.Body.String())
	}
	if rec := changeOperatorPassword(d, session, temporary, "clave-segura-1"); rec.Code != 200 {
		t.Fatalf("change password: %d %s", rec.Code, rec.Body.String())
	}
	return session
}

// panelCall ejecuta un handler del panel con la sesión del operador, como lo hace la ruta.
func panelCall(d operatorAccess, handler func(http.ResponseWriter, *http.Request), method, body, session string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	d.OperatorReady(http.HandlerFunc(handler)).ServeHTTP(rec, operatorRequest(method, body, nil, scope{}, session))
	return rec
}
