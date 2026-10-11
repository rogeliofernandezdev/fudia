package domain

type Zone struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sortOrder"`
	Active    bool   `json:"active"`
}
type ZoneInput struct {
	Name      string `json:"name"`
	SortOrder *int   `json:"sortOrder"`
	Active    *bool  `json:"active"`
}
type Table struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Seats     int    `json:"seats"`
	Zone      string `json:"Zone"`
	Active    bool   `json:"active"`
	QrToken   string `json:"qrToken"`
	QrEnabled bool   `json:"qrEnabled"`
}
type TableInput struct {
	Name      string `json:"name"`
	Seats     *int   `json:"seats"`
	Zone      string `json:"Zone"`
	Active    *bool  `json:"active"`
	QrEnabled *bool  `json:"qrEnabled"`
}
type TableBatchInput struct {
	Items []TableInput `json:"items"`
}
type TableBatchItem struct {
	Name      string `json:"name"`
	Seats     int    `json:"seats"`
	Zone      string `json:"Zone"`
	QrEnabled bool   `json:"qrEnabled"`
}
type TableQR struct {
	Name             string `json:"name"`
	Seats            int    `json:"seats"`
	Zone             string `json:"zone"`
	OrgName          string `json:"organizationName"`
	LocName          string `json:"locationName"`
	ConciergeEnabled bool   `json:"conciergeEnabled"`
}
type TableBatchResult struct {
	Items []Table `json:"items"`
}
