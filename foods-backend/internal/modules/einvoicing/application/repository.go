package application

import (
	"context"
	"errors"

	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/domain"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/ports"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

type scope = requestscope.Scope

var (
	ErrNotFound         = errors.New("einvoicing record not found")
	ErrSeriesMissing    = errors.New("einvoicing series not configured")
	ErrOrderHasDocument = errors.New("order already has an active fiscal document")
	ErrSeriesInUse      = errors.New("series already issued documents")
	ErrSeriesTaken      = errors.New("series used by another location")
)

// IssueContext is the session location's fiscal setup at issue time.
type IssueContext struct {
	FiscalProfileID string
	Country         string
	Currency        string
	BusinessDate    string
	Enabled         bool
	Provider        string
	Environment     string
	DefaultDocument string
}
type NewDocument struct {
	FiscalProfileID string
	OrderID         string
	Country         string
	DocumentType    string
	Series          string
	Number          int
	IssueDate       string
	Currency        string
	Provider        string
	Environment     string
	Customer        domain.Customer
	Breakdown       domain.Breakdown
}

// ClaimedDocument is a document taken by the worker (status SENDING).
type ClaimedDocument struct {
	Document        ports.Document
	OrganizationID  string
	FiscalProfileID string
	Provider        string
	Attempts        int
}

type DocumentRow struct {
	ID                string
	Country           string
	DocumentType      string
	Series            string
	Number            int
	IssueDate         string
	Currency          string
	Customer          domain.Customer
	NetTotal          string
	TaxTotal          string
	Total             string
	Status            string
	Provider          string
	Environment       string
	ProviderReference string
	ProviderCode      string
	ProviderMessage   string
	PDFURL            string
	XMLURL            string
	CDRURL            string
	Attempts          int
	NextAttemptAt     *string
	OrderID           string
	OrderCode         string
	CreatedAt         string
	AcceptedAt        *string
	DigestValue       string
	// Files lists the stored artifact kinds (xml, cdr).
	Files []string
}
type DocumentFilter struct {
	Status string
	Query  string
	Page   int
	Size   int
}

type ProfileSettingRow struct {
	FiscalProfileID string
	Country         string
	CountryName     string
	Currency        string
	Default         bool
	Enabled         bool
	Provider        string
	Environment     string
}
type LocationSettingRow struct {
	LocationID      string
	Name            string
	FiscalProfileID string
	Country         string
	Position        int
	DefaultDocument string
	Series          map[string]SeriesRow
}
type SeriesRow struct {
	Series     string
	LastNumber int
}

type Repository interface {
	IssueContext(ctx context.Context, s scope) (IssueContext, error)
	NextNumber(ctx context.Context, s scope, documentType string) (string, int, error)
	InsertDocument(ctx context.Context, s scope, in NewDocument) (string, error)
	ClaimDocument(ctx context.Context, id string) (ClaimedDocument, bool, error)
	CompleteDocument(ctx context.Context, id string, result ports.Result) error
	FailDocument(ctx context.Context, id, message string, retryMinutes int) error
	DueDocuments(ctx context.Context, limit int) ([]string, error)
	Documents(ctx context.Context, s scope, f DocumentFilter) ([]DocumentRow, int, error)
	Document(ctx context.Context, s scope, id string) (DocumentRow, domain.Breakdown, error)
	RetryDocument(ctx context.Context, s scope, id string) (int64, error)

	SaveFiles(ctx context.Context, organizationID, documentID, digest string, files []ports.File) error
	File(ctx context.Context, s scope, documentID, kind string) (ports.File, error)
	// DocumentCredentials is the worker read: no session scope, only the
	// document's own tenant and profile.
	DocumentCredentials(ctx context.Context, organizationID, profileID, provider string) (SealedCredentials, bool, error)

	Settings(ctx context.Context, s scope) ([]ProfileSettingRow, []LocationSettingRow, error)
	ProfileSetting(ctx context.Context, s scope, profileID string) (ProfileSettingRow, error)
	Issuer(ctx context.Context, s scope) (ports.Issuer, error)
	Credentials(ctx context.Context, s scope, profileID, provider string) (SealedCredentials, bool, error)
	SaveCredentials(ctx context.Context, s scope, profileID, provider string, sealed SealedCredentials) error
	ProfileCountry(ctx context.Context, s scope, profileID string) (string, error)
	SaveProfileSettings(ctx context.Context, s scope, profileID string, in ProfileSettingsInput) error
	LocationCountry(ctx context.Context, s scope, locationID string) (string, error)
	SaveLocationSettings(ctx context.Context, s scope, locationID string, in LocationSettingsInput) error
}
