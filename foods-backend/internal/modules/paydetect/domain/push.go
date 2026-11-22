package domain

import (
	"strings"
	"time"
)

// PushTokenLimit acota el token de Firebase Cloud Messaging que declara la app
// (los reales rondan los 160 caracteres).
const PushTokenLimit = 4096

// WakeInterval es el mínimo entre dos avisos al mismo dispositivo: varias
// pestañas de Caja o instancias de la API comparten el límite en la base.
const WakeInterval = 10 * time.Second

// WakeTarget es un dispositivo activo del local con su token de push.
type WakeTarget struct {
	DeviceID string
	Token    string
}

// WakeResult resume un pedido de aviso: cuántos dispositivos del local pueden
// recibir push y a cuántos se envió ahora (el resto lo omitió el intervalo).
// Con Devices=0 Caja espacia sus pedidos: el local no usa Fudia Pay con push.
type WakeResult struct {
	Devices int `json:"devices"`
	Sent    int `json:"sent"`
}

// NormalizePushToken limpia el token declarado por la app. Vacío significa
// «sin push» (la app lo borra al desvincular); false si no es un token válido.
func NormalizePushToken(raw string) (string, bool) {
	token := strings.TrimSpace(raw)
	if token == "" {
		return "", true
	}
	if len(token) > PushTokenLimit || strings.ContainsAny(token, " \t\r\n") {
		return "", false
	}
	return token, true
}
