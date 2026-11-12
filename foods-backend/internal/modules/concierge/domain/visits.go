package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

type CustomerIdentity struct {
	Provider      string `json:"provider"`
	Subject       string `json:"subject"`
	VerifiedPhone string `json:"verifiedPhone,omitempty"`
}

var verifiedPhonePattern = regexp.MustCompile(`^[1-9][0-9]{7,14}$`)

func (i CustomerIdentity) Valid() bool {
	return (i.Provider == "META_CLOUD_API" || i.Provider == "WHATSMEOW" || (i.Provider == "FUDIA_WEB" && i.VerifiedPhone == "")) && len(i.Subject) > 0 && len(i.Subject) <= 256 && !strings.ContainsAny(i.Subject, "\r\n\x00") && (i.VerifiedPhone == "" || verifiedPhonePattern.MatchString(i.VerifiedPhone)) && (i.Provider != "META_CLOUD_API" || (i.Subject == i.VerifiedPhone && i.VerifiedPhone != ""))
}
func IdentityHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func (i CustomerIdentity) SubjectKey(namespace string) string {
	return IdentityHash(i.Provider + ":" + namespace + ":" + i.Subject)
}
func (i CustomerIdentity) PhoneKey() string { return IdentityHash("verified-phone:" + i.VerifiedPhone) }

type RestaurantSearchInput struct {
	ChannelKey  string           `json:"channelKey"`
	Identity    CustomerIdentity `json:"identity"`
	OnlyVisited bool             `json:"onlyVisited"`
	Offset      int              `json:"offset"`
}
type RestaurantSelectionInput struct {
	ChannelKey string `json:"channelKey"`
	LocationID string `json:"locationId"`
}
type RestaurantView struct {
	OrganizationID   string `json:"organizationId"`
	LocationID       string `json:"locationId"`
	OrganizationName string `json:"organizationName"`
	LocationName     string `json:"locationName"`
	VisitCount       int64  `json:"visitCount"`
	LastVisitedAt    string `json:"lastVisitedAt,omitempty"`
	QRToken          string `json:"qrToken,omitempty"`
}
type RestaurantPage struct {
	Items   []RestaurantView `json:"items"`
	HasMore bool             `json:"hasMore"`
	Offset  int              `json:"offset"`
}
