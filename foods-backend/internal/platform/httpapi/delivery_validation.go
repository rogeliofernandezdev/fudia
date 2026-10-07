package httpapi

import (
	"strings"
	"unicode/utf8"
)

func validateDeliveryContact(channel, name, phone, address, reference string) string {
	if channel != "delivery" {
		return ""
	}
	if strings.TrimSpace(name) == "" || utf8.RuneCountInString(strings.TrimSpace(name)) > 160 {
		return "Ingresa el nombre del cliente (máximo 160 caracteres)."
	}
	phone = strings.TrimSpace(phone)
	digits := 0
	for index, character := range phone {
		switch {
		case character >= '0' && character <= '9':
			digits++
		case character == '+' && index == 0:
		case character == ' ' || character == '\t' || character == '(' || character == ')' || character == '.' || character == '-':
		default:
			return "Ingresa un teléfono válido de 7 a 15 dígitos."
		}
	}
	if digits < 7 || digits > 15 || utf8.RuneCountInString(phone) > 30 {
		return "Ingresa un teléfono válido de 7 a 15 dígitos."
	}
	if strings.TrimSpace(address) == "" || utf8.RuneCountInString(strings.TrimSpace(address)) > 240 {
		return "Ingresa la dirección de entrega (máximo 240 caracteres)."
	}
	if utf8.RuneCountInString(strings.TrimSpace(reference)) > 240 {
		return "La referencia de entrega no puede superar 240 caracteres."
	}
	return ""
}
