package domain

import "time"

type Candidate struct {
	UserID, OrganizationID, Name, Hash string
	PlatformAdmin                      bool
}
type Session struct {
	Hash                               []byte
	UserID, OrganizationID, LocationID string
	ExpiresAt                          time.Time
}
type LoginResult struct {
	Token, UserID, Name string
	PlatformAdmin       bool
	ExpiresIn           int
}
