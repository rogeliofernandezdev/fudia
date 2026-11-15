package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/foods-platform/foods-backend/internal/modules/delivery/application"
	"github.com/foods-platform/foods-backend/internal/modules/delivery/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
)

// Panel web del proveedor de Delivery Fudia. El operador de la central entra
// con su cuenta; la sesión de la cookie identifica a la central y el caso de
// uso vuelve a validar que cada oferta sea suya.

// PartnerSessionCookie guarda la sesión del operador de la central.
const PartnerSessionCookie = "fudia_partner_session"

type operatorKey struct{}

func sessionToken(r *http.Request) string {
	if c, err := r.Cookie(PartnerSessionCookie); err == nil {
		return c.Value
	}
	return ""
}

func (h *Handler) sessionCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: PartnerSessionCookie, Value: token, Path: "/", HttpOnly: true, Secure: h.secureCookie, SameSite: http.SameSiteLaxMode, MaxAge: maxAge})
}

func operatorFrom(ctx context.Context) domain.OperatorSession {
	session, _ := ctx.Value(operatorKey{}).(domain.OperatorSession)
	return session
}

// OperatorAuth exige la sesión de un operador activo; sin ella responde 401 y borra la cookie.
func (h *Handler) OperatorAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		session, err := h.operators.Authenticate(r.Context(), sessionToken(r))
		if err != nil {
			if sessionToken(r) != "" {
				h.sessionCookie(w, "", -1)
			}
			shared.WriteError(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), operatorKey{}, session)))
	})
}

// OperatorReady exige además que el operador ya haya cambiado su contraseña temporal.
func (h *Handler) OperatorReady(next http.Handler) http.Handler {
	return h.OperatorAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := application.RequirePassword(operatorFrom(r.Context())); err != nil {
			shared.WriteError(w, err)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func sessionView(session domain.OperatorSession) map[string]any {
	return map[string]any{
		"operator": map[string]any{"name": session.OperatorName, "email": session.Email, "mustChangePassword": session.MustChangePassword},
		"partner":  map[string]any{"name": session.PartnerName, "active": session.PartnerActive},
	}
}

// PartnerLogin abre la sesión del operador y la guarda en una cookie HttpOnly.
func (h *Handler) PartnerLogin(w http.ResponseWriter, r *http.Request) {
	var in domain.LoginInput
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil {
		shared.Fail(w, http.StatusBadRequest, "invalid_request", "Revisa los datos enviados.")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	out, err := h.operators.Login(r.Context(), in)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	h.sessionCookie(w, out.Token, out.ExpiresIn)
	view := sessionView(out.Session)
	view["expiresIn"] = out.ExpiresIn
	shared.WriteJSON(w, http.StatusOK, view)
}

// PartnerLogout revoca la sesión y borra la cookie, aunque ya haya vencido.
func (h *Handler) PartnerLogout(w http.ResponseWriter, r *http.Request) {
	if err := h.operators.Logout(r.Context(), sessionToken(r)); err != nil {
		shared.WriteError(w, err)
		return
	}
	h.sessionCookie(w, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

// PartnerSession devuelve el operador y la central de la sesión.
func (h *Handler) PartnerSession(w http.ResponseWriter, r *http.Request) {
	shared.WriteJSON(w, http.StatusOK, sessionView(operatorFrom(r.Context())))
}

// PartnerPassword cambia la contraseña del operador; sus otras sesiones se cierran.
func (h *Handler) PartnerPassword(w http.ResponseWriter, r *http.Request) {
	var in domain.PasswordChangeInput
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil {
		shared.Fail(w, http.StatusBadRequest, "invalid_request", "Revisa los datos enviados.")
		return
	}
	session, err := h.operators.ChangePassword(r.Context(), sessionToken(r), in)
	respond(w, http.StatusOK, sessionView(session), err)
}

func (h *Handler) Panel(w http.ResponseWriter, r *http.Request) {
	out, err := h.dispatcher.Panel(r.Context(), application.PanelPartnerOf(operatorFrom(r.Context())))
	w.Header().Set("Cache-Control", "no-store")
	respond(w, http.StatusOK, out, err)
}

func (h *Handler) PanelAction(w http.ResponseWriter, r *http.Request) {
	var in domain.PanelActionInput
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil {
		shared.Fail(w, http.StatusBadRequest, "invalid_delivery_action", "La acción no es válida.")
		return
	}
	session := operatorFrom(r.Context())
	out, err := h.dispatcher.PanelAct(r.Context(), application.PanelPartnerOf(session), in)
	if err == nil {
		h.operators.Record(r.Context(), session, in.Action, out.Outcome, out.OrderCode)
	}
	respond(w, http.StatusOK, out, err)
}

// PanelCourier asigna o cambia el motorizado de un reparto y devuelve su enlace.
func (h *Handler) PanelCourier(w http.ResponseWriter, r *http.Request) {
	var in domain.CourierInput
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil {
		shared.Fail(w, http.StatusBadRequest, "invalid_courier_name", "Escribe el nombre del motorizado (2 a 60 caracteres).")
		return
	}
	session := operatorFrom(r.Context())
	out, err := h.dispatcher.PanelCourier(r.Context(), application.PanelPartnerOf(session), in)
	if err == nil {
		h.operators.Record(r.Context(), session, "assign_courier", out.Outcome, "")
	}
	respond(w, http.StatusOK, out, err)
}

// Plataforma: operadores de cada central.

func (h *Handler) ListOperators(w http.ResponseWriter, r *http.Request) {
	out, err := h.operators.List(r.Context(), r.PathValue("id"))
	respond(w, http.StatusOK, map[string]any{"items": out}, err)
}

func (h *Handler) CreateOperator(w http.ResponseWriter, r *http.Request) {
	var in domain.OperatorInput
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil {
		shared.Fail(w, http.StatusBadRequest, "invalid_delivery_operator", "Revisa los datos del operador.")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	out, err := h.operators.Create(r.Context(), r.PathValue("id"), in)
	respond(w, http.StatusCreated, out, err)
}

func (h *Handler) UpdateOperator(w http.ResponseWriter, r *http.Request) {
	var in domain.OperatorInput
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil {
		shared.Fail(w, http.StatusBadRequest, "invalid_delivery_operator", "Revisa los datos del operador.")
		return
	}
	out, err := h.operators.Update(r.Context(), r.PathValue("id"), r.PathValue("operatorId"), in)
	respond(w, http.StatusOK, out, err)
}

func (h *Handler) ResetOperatorPassword(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	out, err := h.operators.ResetPassword(r.Context(), r.PathValue("id"), r.PathValue("operatorId"))
	respond(w, http.StatusOK, out, err)
}

// Enlace del motorizado: ruta pública; el enlace de la ruta identifica el reparto.

func (h *Handler) Courier(w http.ResponseWriter, r *http.Request) {
	out, err := h.dispatcher.CourierView(r.Context(), r.PathValue("token"))
	w.Header().Set("Cache-Control", "no-store")
	respond(w, http.StatusOK, out, err)
}

func (h *Handler) CourierAction(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Action string `json:"action"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil {
		shared.Fail(w, http.StatusBadRequest, "invalid_delivery_action", "La acción no es válida.")
		return
	}
	out, err := h.dispatcher.CourierAct(r.Context(), r.PathValue("token"), in.Action)
	respond(w, http.StatusOK, out, err)
}

func (h *Handler) CourierLocation(w http.ResponseWriter, r *http.Request) {
	var in domain.CourierLocation
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil {
		shared.Fail(w, http.StatusBadRequest, "invalid_location", "La ubicación no es válida.")
		return
	}
	if err := h.dispatcher.CourierLocation(r.Context(), r.PathValue("token"), in); err != nil {
		shared.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
