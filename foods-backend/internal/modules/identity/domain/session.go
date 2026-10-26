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

// Access es lo que la sesión puede hacer en su local: se lee junto con la sesión
// para que cada petición valide su permiso sin otra consulta.
type Access struct {
	PlatformAdmin bool
	Permissions   []string
}

func (a Access) Allows(permission string) bool {
	if a.PlatformAdmin {
		return true
	}
	for _, granted := range a.Permissions {
		if granted == permission {
			return true
		}
	}
	return false
}
