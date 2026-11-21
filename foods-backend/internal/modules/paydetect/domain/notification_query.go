package domain

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// maxNotificationQuery limita la búsqueda de Pagos detectados: un remitente
// guardado nunca supera 80 caracteres.
const maxNotificationQuery = 80

// NotificationQuery limpia lo escrito en el buscador de Pagos detectados.
func NotificationQuery(raw string) string {
	q := strings.Join(strings.Fields(raw), " ")
	for utf8.RuneCountInString(q) > maxNotificationQuery {
		_, size := utf8.DecodeLastRuneInString(q)
		q = q[:len(q)-size]
	}
	return q
}

var amountQuery = regexp.MustCompile(`^\d{1,9}(\.\d{0,2})?$`)

// NotificationAmountPrefix devuelve lo escrito como prefijo de monto
// («S/ 25,5» → «25.5») o vacío si no es una cifra.
func NotificationAmountPrefix(q string) string {
	value := strings.TrimSpace(q)
	value = strings.TrimPrefix(strings.TrimPrefix(value, "S/"), "s/")
	value = strings.ReplaceAll(strings.TrimSpace(value), ",", ".")
	if !amountQuery.MatchString(value) {
		return ""
	}
	return value
}
