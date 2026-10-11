package httpapi

type platformCountryOption struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	DefaultCurrency string `json:"defaultCurrency"`
	CallingCode     string `json:"callingCode"`
	DefaultTimezone string `json:"defaultTimezone"`
}
