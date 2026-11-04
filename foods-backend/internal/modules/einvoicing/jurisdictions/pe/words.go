package pe

import (
	"fmt"
	"math/big"
	"strings"
)

var (
	units = []string{"", "UNO", "DOS", "TRES", "CUATRO", "CINCO", "SEIS", "SIETE", "OCHO", "NUEVE", "DIEZ",
		"ONCE", "DOCE", "TRECE", "CATORCE", "QUINCE", "DIECISEIS", "DIECISIETE", "DIECIOCHO", "DIECINUEVE", "VEINTE",
		"VEINTIUNO", "VEINTIDOS", "VEINTITRES", "VEINTICUATRO", "VEINTICINCO", "VEINTISEIS", "VEINTISIETE", "VEINTIOCHO", "VEINTINUEVE"}
	tens     = []string{"", "", "", "TREINTA", "CUARENTA", "CINCUENTA", "SESENTA", "SETENTA", "OCHENTA", "NOVENTA"}
	hundreds = []string{"", "CIENTO", "DOSCIENTOS", "TRESCIENTOS", "CUATROCIENTOS", "QUINIENTOS", "SEISCIENTOS", "SETECIENTOS", "OCHOCIENTOS", "NOVECIENTOS"}
)

var currencyNames = map[string]string{"PEN": "SOLES", "USD": "DÓLARES AMERICANOS", "EUR": "EUROS"}

// AmountInWords renders the SUNAT legend 1000, e.g. "OCHENTA CON 30/100 SOLES".
func AmountInWords(amount string, currency string) string {
	value, ok := new(big.Rat).SetString(amount)
	if !ok {
		value = new(big.Rat)
	}
	cents := new(big.Int).Div(new(big.Int).Mul(value.Num(), big.NewInt(100)), value.Denom())
	whole, fraction := new(big.Int).DivMod(cents, big.NewInt(100), new(big.Int))
	name := currencyNames[currency]
	if name == "" {
		name = currency
	}
	return fmt.Sprintf("%s CON %02d/100 %s", numberWords(whole.Int64()), fraction.Int64(), name)
}

func numberWords(n int64) string {
	switch {
	case n == 0:
		return "CERO"
	case n < 30:
		return units[n]
	case n < 100:
		if n%10 == 0 {
			return tens[n/10]
		}
		return tens[n/10] + " Y " + units[n%10]
	case n == 100:
		return "CIEN"
	case n < 1000:
		return strings.TrimSpace(hundreds[n/100] + " " + wordsOrEmpty(n%100))
	case n < 1_000_000:
		thousands := "MIL"
		if n/1000 > 1 {
			thousands = shortNumber(n/1000) + " MIL"
		}
		return strings.TrimSpace(thousands + " " + wordsOrEmpty(n%1000))
	default:
		millions := "UN MILLON"
		if n/1_000_000 > 1 {
			millions = shortNumber(n/1_000_000) + " MILLONES"
		}
		return strings.TrimSpace(millions + " " + wordsOrEmpty(n%1_000_000))
	}
}

func wordsOrEmpty(n int64) string {
	if n == 0 {
		return ""
	}
	return numberWords(n)
}

// shortNumber uses the apocopated form before MIL/MILLONES (UN, VEINTIUN).
func shortNumber(n int64) string {
	words := numberWords(n)
	if strings.HasSuffix(words, "UNO") {
		return strings.TrimSuffix(words, "UNO") + "UN"
	}
	return words
}
