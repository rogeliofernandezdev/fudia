package main

import (
	"log/slog"
	"os"

	audit "github.com/foods-platform/foods-backend/internal/modules/audit/application"
	einvoicingapp "github.com/foods-platform/foods-backend/internal/modules/einvoicing/application"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/infrastructure/dispatch"
	einvoicingpg "github.com/foods-platform/foods-backend/internal/modules/einvoicing/infrastructure/postgres"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/jurisdictions/pe"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/ports"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/providers/fake"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/providers/sunat"
	einvoicinghttp "github.com/foods-platform/foods-backend/internal/modules/einvoicing/transport/http"
	orgcontracts "github.com/foods-platform/foods-backend/internal/modules/organizations/contracts"
	"github.com/foods-platform/foods-backend/internal/platform/secretbox"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"github.com/jackc/pgx/v5/pgxpool"
)

// newEInvoicing composes electronic invoicing: country rules and providers are
// registered here; the simulated provider is test-only by contract. Tenant
// credentials are encrypted with FUDIA_EINVOICING_MASTER_KEY (Secret Manager);
// without it, credentials cannot be saved and SUNAT production cannot send.
func newEInvoicing(db *pgxpool.Pool, work unitofwork.Manager, audit audit.Repository, orders einvoicingapp.OrderSource, taxes orgcontracts.TaxResolver, access einvoicingapp.Entitlement) (*einvoicingapp.Documents, *einvoicinghttp.Handler) {
	registry := ports.NewRegistry().WithJurisdiction(pe.New()).WithProvider(sunat.New()).WithProvider(fake.New())
	repo := einvoicingpg.New(db)
	documents := einvoicingapp.NewDocuments(repo, audit, orders, taxes, access, registry)
	documents.WithDispatcher(dispatch.NewInProcess(documents.Process))
	settings := einvoicingapp.NewSettings(repo, work, audit, access, registry)
	if keyring, err := secretbox.FromEnv("FUDIA_EINVOICING"); err == nil {
		documents.WithCipher(keyring)
		settings.WithCipher(keyring)
	} else if err != secretbox.ErrNotConfigured {
		slog.Error("einvoicing_master_key_invalid", "error", err.Error())
	}
	return documents, einvoicinghttp.New(documents, settings, func() string { return os.Getenv("FUDIA_EINVOICING_WORKER_KEY") })
}
