package application

import (
	"context"
	"crypto/sha256"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/identity/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"testing"
	"time"
)

type memorySessions struct {
	candidates                                   []domain.Candidate
	created                                      *domain.Session
	locationError, sessionError, errorPermission error
	seenScope                                    requestscope.Scope
	seenEmail, seenPermission                    string
	allowed, admin                               bool
	revoked                                      []byte
}

func (r *memorySessions) Candidates(_ context.Context, email string) ([]domain.Candidate, error) {
	r.seenEmail = email
	return r.candidates, nil
}
func (r *memorySessions) FirstLocation(context.Context, domain.Candidate) (string, error) {
	return "location-a", r.locationError
}
func (r *memorySessions) CreateSession(_ context.Context, s domain.Session) error {
	r.created = &s
	return nil
}
func (r *memorySessions) RevokeSession(_ context.Context, hash []byte) error {
	r.revoked = hash
	return nil
}
func (r *memorySessions) SessionScope(context.Context, []byte) (requestscope.Scope, error) {
	return requestscope.Scope{UserID: "user-a", OrganizationID: "org-a", LocationID: "location-a"}, r.sessionError
}
func (r *memorySessions) IsPlatformAdmin(context.Context, string) (bool, error) { return r.admin, nil }
func (r *memorySessions) HasPermission(_ context.Context, s requestscope.Scope, p string) (bool, error) {
	r.seenScope = s
	r.seenPermission = p
	return r.allowed, r.errorPermission
}

type credentials struct{ err error }

func (credentials) Matches(hash, password string) bool { return hash == password }
func (c credentials) Token() (string, error)           { return "opaque-test-token", c.err }
func assertCode(t *testing.T, err error, code string) {
	t.Helper()
	var e *apperror.Error
	if !errors.As(err, &e) || e.Code != code {
		t.Fatalf("wanted %s, got %v", code, err)
	}
}
func TestLoginPreservesTenantAndStoresOnlyTokenHash(t *testing.T) {
	r := &memorySessions{candidates: []domain.Candidate{{UserID: "user-a", OrganizationID: "org-a", Name: "Ana", Hash: "password"}}}
	s := New(r, credentials{})
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	out, err := s.Login(context.Background(), " ANA@EXAMPLE.TEST ", "password")
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte(out.Token))
	if r.seenEmail != "ana@example.test" || r.created == nil || string(r.created.Hash) != string(hash[:]) || r.created.OrganizationID != "org-a" || r.created.LocationID != "location-a" || r.created.UserID != "user-a" || !r.created.ExpiresAt.Equal(now.Add(12*time.Hour)) {
		t.Fatalf("wrong persisted session: %#v", r.created)
	}
	if out.ExpiresIn != 43200 {
		t.Fatal(out)
	}
}
func TestLoginRejectsAmbiguousOrUnassignedAccountsWithoutSession(t *testing.T) {
	for _, tc := range []struct {
		name          string
		candidates    []domain.Candidate
		locationError error
		code          string
	}{{"invalid", nil, nil, "invalid_credentials"}, {"ambiguous", []domain.Candidate{{Hash: "p"}, {Hash: "p"}}, nil, "ambiguous_account"}, {"unassigned", []domain.Candidate{{Hash: "p"}}, ErrNotFound, "account_without_access"}} {
		t.Run(tc.name, func(t *testing.T) {
			r := &memorySessions{candidates: tc.candidates, locationError: tc.locationError}
			_, err := New(r, credentials{}).Login(context.Background(), "a@b.test", "p")
			assertCode(t, err, tc.code)
			if r.created != nil {
				t.Fatal("rejected login persisted a session")
			}
		})
	}
}
func TestSessionErrorsAndPermissionsKeepScope(t *testing.T) {
	r := &memorySessions{}
	s := New(r, credentials{})
	ctx := context.Background()
	_, err := s.Authenticate(ctx, "")
	assertCode(t, err, "unauthenticated")
	r.sessionError = ErrNotFound
	_, err = s.Authenticate(ctx, "expired")
	assertCode(t, err, "session_expired")
	r.sessionError = errors.New("database password must not leak")
	_, err = s.Authenticate(ctx, "token")
	assertCode(t, err, "session_unavailable")
	scope := requestscope.Scope{UserID: "u", OrganizationID: "o", LocationID: "l"}
	err = s.Authorize(ctx, scope, "customers.manage")
	assertCode(t, err, "forbidden")
	if r.seenScope != scope || r.seenPermission != "customers.manage" {
		t.Fatal("authorization lost tenant/local")
	}
	r.allowed = true
	if err = s.Authorize(ctx, scope, "customers.manage"); err != nil {
		t.Fatal(err)
	}
}
func TestLogoutHashesToken(t *testing.T) {
	r := &memorySessions{}
	s := New(r, credentials{})
	if err := s.Logout(context.Background(), "token"); err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256([]byte("token"))
	if string(r.revoked) != string(want[:]) {
		t.Fatal("raw token persisted")
	}
}
