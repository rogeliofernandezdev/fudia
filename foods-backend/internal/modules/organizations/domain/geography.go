package domain

// GeographicArea is a public catalog entry. Codes are INEI natural keys;
// persistent IDs are generated only by PostgreSQL.
type GeographicArea struct {
	Code       string `json:"code"`
	ParentCode string `json:"parentCode"`
	Name       string `json:"name"`
	Label      string `json:"label"`
}
