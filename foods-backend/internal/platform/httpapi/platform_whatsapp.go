package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type platformWhatsAppChannel struct {
	ID            string  `json:"id"`
	CountryCode   string  `json:"countryCode"`
	CountryName   string  `json:"countryName"`
	PhoneNumber   string  `json:"phoneNumber"`
	PhoneNumberID *string `json:"phoneNumberId,omitempty"`
	DisplayName   string  `json:"displayName"`
	Active        bool    `json:"active"`
}

type platformWhatsAppChannelInput struct {
	CountryCode   string  `json:"countryCode"`
	PhoneNumber   string  `json:"phoneNumber"`
	PhoneNumberID *string `json:"phoneNumberId"`
	DisplayName   string  `json:"displayName"`
	Active        bool    `json:"active"`
}

func cleanOptional(value *string) *string {
	if value == nil {
		return nil
	}
	v := strings.TrimSpace(*value)
	if v == "" {
		return nil
	}
	return &v
}

func (a *API) listPlatformWhatsAppChannels(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(r.Context(), `
		SELECT w.id,c.code,c.name,w.phone_number,w.phone_number_id,COALESCE(w.display_name,''),w.active
		FROM platform_whatsapp_channels w
		JOIN platform_countries c ON c.code=w.country_code
		ORDER BY c.name,w.created_at,w.id`)
	if err != nil {
		fail(w, 503, "channels_unavailable", "No pudimos cargar los canales de WhatsApp.")
		return
	}
	defer rows.Close()
	items := []platformWhatsAppChannel{}
	for rows.Next() {
		var item platformWhatsAppChannel
		if err := rows.Scan(&item.ID, &item.CountryCode, &item.CountryName, &item.PhoneNumber, &item.PhoneNumberID, &item.DisplayName, &item.Active); err != nil {
			fail(w, 503, "channels_unavailable", "No pudimos cargar los canales de WhatsApp.")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		fail(w, 503, "channels_unavailable", "No pudimos cargar los canales de WhatsApp.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func validatePlatformWhatsAppChannel(in *platformWhatsAppChannelInput) (string, string) {
	in.CountryCode = strings.ToUpper(strings.TrimSpace(in.CountryCode))
	in.PhoneNumber = strings.TrimSpace(in.PhoneNumber)
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.PhoneNumberID = cleanOptional(in.PhoneNumberID)
	if len(in.CountryCode) != 2 {
		return "invalid_country", "Selecciona un país válido."
	}
	if in.PhoneNumber == "" || len(in.PhoneNumber) > 40 {
		return "invalid_phone", "Ingresa un número de WhatsApp válido."
	}
	if in.Active && in.PhoneNumberID == nil {
		return "channel_incomplete", "Para activar el canal debes configurar el phone_number_id."
	}
	return "", ""
}

func (a *API) platformCountryExists(ctx context.Context, code string) (bool, error) {
	var exists bool
	err := a.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM platform_countries WHERE code=$1 AND active)`, code).Scan(&exists)
	return exists, err
}

func platformChannelDBError(w http.ResponseWriter, err error) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		fail(w, http.StatusConflict, "channel_already_exists", "El número o phone_number_id ya está registrado en otro canal.")
		return
	}
	fail(w, http.StatusInternalServerError, "channel_save_failed", "No pudimos guardar el canal de WhatsApp.")
}

func (a *API) createPlatformWhatsAppChannel(w http.ResponseWriter, r *http.Request) {
	var in platformWhatsAppChannelInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		fail(w, 400, "invalid_json", "El cuerpo de la solicitud no es válido.")
		return
	}
	if code, msg := validatePlatformWhatsAppChannel(&in); code != "" {
		fail(w, 400, code, msg)
		return
	}
	countryExists, err := a.platformCountryExists(r.Context(), in.CountryCode)
	if err != nil {
		fail(w, 503, "catalogs_unavailable", "No pudimos validar el país seleccionado.")
		return
	}
	if !countryExists {
		fail(w, 400, "invalid_country", "El país no existe o está deshabilitado.")
		return
	}
	var item platformWhatsAppChannel
	err = a.db.QueryRow(r.Context(), `
		INSERT INTO platform_whatsapp_channels(country_code,phone_number,phone_number_id,display_name,active)
		VALUES($1,$2,$3,$4,$5)
		RETURNING id,country_code,(SELECT name FROM platform_countries WHERE code=$1),phone_number,phone_number_id,COALESCE(display_name,''),active`,
		in.CountryCode, in.PhoneNumber, in.PhoneNumberID, in.DisplayName, in.Active).
		Scan(&item.ID, &item.CountryCode, &item.CountryName, &item.PhoneNumber, &item.PhoneNumberID, &item.DisplayName, &item.Active)
	if err != nil {
		platformChannelDBError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) updatePlatformWhatsAppChannel(w http.ResponseWriter, r *http.Request) {
	var in platformWhatsAppChannelInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		fail(w, 400, "invalid_json", "El cuerpo de la solicitud no es válido.")
		return
	}
	if code, msg := validatePlatformWhatsAppChannel(&in); code != "" {
		fail(w, 400, code, msg)
		return
	}
	countryExists, err := a.platformCountryExists(r.Context(), in.CountryCode)
	if err != nil {
		fail(w, 503, "catalogs_unavailable", "No pudimos validar el país seleccionado.")
		return
	}
	if !countryExists {
		fail(w, 400, "invalid_country", "El país no existe o está deshabilitado.")
		return
	}
	var item platformWhatsAppChannel
	err = a.db.QueryRow(r.Context(), `
		UPDATE platform_whatsapp_channels w SET
		  country_code=$2,phone_number=$3,phone_number_id=$4,display_name=$5,active=$6,updated_at=now()
		WHERE w.id=$1
		RETURNING w.id,w.country_code,(SELECT name FROM platform_countries WHERE code=w.country_code),w.phone_number,w.phone_number_id,COALESCE(w.display_name,''),w.active`,
		r.PathValue("id"), in.CountryCode, in.PhoneNumber, in.PhoneNumberID, in.DisplayName, in.Active).
		Scan(&item.ID, &item.CountryCode, &item.CountryName, &item.PhoneNumber, &item.PhoneNumberID, &item.DisplayName, &item.Active)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			fail(w, 404, "channel_not_found", "El canal de WhatsApp no existe.")
			return
		}
		platformChannelDBError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
