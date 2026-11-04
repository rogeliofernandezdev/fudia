package application

import (
	"context"
	"errors"
	"math/big"
	"strings"

	audit "github.com/foods-platform/foods-backend/internal/modules/audit/application"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/domain"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/ports"
	ordercontracts "github.com/foods-platform/foods-backend/internal/modules/orders/contracts"
	orgcontracts "github.com/foods-platform/foods-backend/internal/modules/organizations/contracts"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

// OrderSource reads the charged order (items, delivery fee, total).
type OrderSource interface {
	PaymentOrder(ctx context.Context, s scope, id string) (ordercontracts.Order, error)
}

// Entitlement tells whether the company may use the facturacion module.
type Entitlement interface {
	ModuleEntitled(ctx context.Context, org, key string) (bool, error)
}

// Dispatcher sends a stored document to the asynchronous worker (in process
// locally; Cloud Tasks in production).
type Dispatcher interface {
	Dispatch(documentID string)
}

type Documents struct {
	repo       Repository
	audit      audit.Repository
	orders     OrderSource
	taxes      orgcontracts.TaxResolver
	access     Entitlement
	registry   *ports.Registry
	dispatcher Dispatcher
	cipher     Cipher
}

var _ contracts.Issuer = (*Documents)(nil)

func NewDocuments(repo Repository, audit audit.Repository, orders OrderSource, taxes orgcontracts.TaxResolver, access Entitlement, registry *ports.Registry) *Documents {
	return &Documents{repo: repo, audit: audit, orders: orders, taxes: taxes, access: access, registry: registry}
}
func (a *Documents) WithDispatcher(d Dispatcher) *Documents { a.dispatcher = d; return a }
func (a *Documents) WithCipher(c Cipher) *Documents         { a.cipher = c; return a }

func failure(k apperror.Kind, code, message string) error { return apperror.New(k, code, message) }

// Setup codes that mean "the module is active but this location still needs
// configuration"; checkout explains them instead of hiding the section.
var pendingSetupCodes = map[string]bool{
	"einvoicing_issuer_incomplete":   true,
	"einvoicing_provider_missing":    true,
	"einvoicing_credentials_missing": true,
	"einvoicing_series_missing":      true,
}

// issueSetup resolves what is needed to issue at the session location, or the
// reason it cannot. The facturacion module is the only switch: with it active,
// a location issues as soon as issuer, credentials and series are complete.
func (a *Documents) issueSetup(ctx context.Context, s scope) (IssueContext, ports.Jurisdiction, error) {
	entitled, err := a.access.ModuleEntitled(ctx, s.OrganizationID, domain.ModuleKey)
	if err != nil {
		return IssueContext{}, nil, failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos validar la facturación electrónica.")
	}
	if !entitled {
		return IssueContext{}, nil, failure(apperror.Conflict, "einvoicing_not_enabled", "La facturación electrónica no está activa para esta empresa.")
	}
	ic, err := a.repo.IssueContext(ctx, s)
	if errors.Is(err, ErrNotFound) {
		return ic, nil, failure(apperror.Conflict, "einvoicing_not_enabled", "El local no tiene un perfil fiscal para emitir comprobantes.")
	}
	if err != nil {
		return ic, nil, failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos cargar la facturación del local.")
	}
	j, ok := a.registry.Jurisdiction(ic.Country)
	if !ok {
		return ic, nil, failure(apperror.Conflict, "einvoicing_unsupported_country", "La facturación electrónica aún no está disponible para el país de este local.")
	}
	if ic.Provider == "" {
		if options := a.registry.ProviderOptions(); len(options) > 0 {
			ic.Provider, ic.Environment = options[0].Value, domain.EnvironmentTest
		}
	}
	return ic, j, a.ready(ctx, s, ic, j)
}

// ready checks the configuration a location needs before offering documents.
func (a *Documents) ready(ctx context.Context, s scope, ic IssueContext, j ports.Jurisdiction) error {
	issuer, err := a.repo.Issuer(ctx, s)
	if err != nil {
		return failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos validar los datos del emisor.")
	}
	if issue := j.ValidateIssuer(issuer); issue != nil {
		return failure(apperror.Conflict, "einvoicing_issuer_incomplete", issue.Message)
	}
	provider, ok := a.registry.Provider(ic.Provider)
	if !ok {
		return failure(apperror.Conflict, "einvoicing_provider_missing", "Elige el proveedor de facturación en Configuración › Facturación.")
	}
	if err := verifyCredentials(ctx, a.repo, a.cipher, s, ic.FiscalProfileID, provider, ic.Environment); err != nil {
		return err
	}
	series, err := a.repo.LocationSeries(ctx, s)
	if err != nil {
		return failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos leer las series del local.")
	}
	for _, option := range j.DocumentTypes() {
		if series[option.Value] == "" {
			return failure(apperror.Conflict, "einvoicing_series_missing", "Configura las series de este local en Configuración › Facturación.")
		}
	}
	return nil
}

// Request implements contracts.Issuer inside the payment transaction.
func (a *Documents) Request(ctx context.Context, s scope, orderID string, request contracts.FiscalRequest) (contracts.DocumentRef, error) {
	var ref contracts.DocumentRef
	request.Normalize()
	if !request.Wanted() {
		return ref, nil
	}
	ic, j, err := a.issueSetup(ctx, s)
	if err != nil {
		return ref, err
	}
	if j.DocumentLabel(request.DocumentType) == "" || !offered(j, request.DocumentType) {
		return ref, failure(apperror.Invalid, "invalid_fiscal_document", "Elige un comprobante disponible para este local.")
	}
	order, err := a.orders.PaymentOrder(ctx, s, orderID)
	if err != nil {
		return ref, err
	}
	charged, ok := domain.ParseAmount(order.Total)
	if !ok {
		return ref, failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos leer el total del pedido.")
	}
	if issue := j.ValidateIssue(request.DocumentType, charged, request.Customer); issue != nil {
		return ref, failure(apperror.Invalid, "invalid_fiscal_customer", issue.Message)
	}
	tax, err := a.taxes.EffectiveTax(ctx, s, ic.FiscalProfileID, ic.BusinessDate)
	if err != nil {
		return ref, err
	}
	lines, err := sourceLines(order)
	if err != nil {
		return ref, failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos leer el detalle del pedido.")
	}
	components := make([]domain.TaxComponent, len(tax.Components))
	for i, c := range tax.Components {
		components[i] = domain.TaxComponent{Code: c.Code, Name: c.Name, Rate: c.Rate}
	}
	breakdown, err := domain.Compute(lines, components, charged)
	if err != nil {
		return ref, failure(apperror.Conflict, "fiscal_total_mismatch", "El detalle del pedido no coincide con el total cobrado.")
	}
	series, number, err := a.repo.NextNumber(ctx, s, request.DocumentType)
	if errors.Is(err, ErrSeriesMissing) {
		return ref, failure(apperror.Conflict, "einvoicing_series_missing", "Configura la serie de "+strings.ToLower(j.DocumentLabel(request.DocumentType))+" de este local en Facturación.")
	}
	if err != nil {
		return ref, failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos reservar el número del comprobante.")
	}
	id, err := a.repo.InsertDocument(ctx, s, NewDocument{
		FiscalProfileID: ic.FiscalProfileID, OrderID: orderID, Country: ic.Country, DocumentType: request.DocumentType,
		Series: series, Number: number, IssueDate: ic.BusinessDate, Currency: ic.Currency, Provider: ic.Provider,
		Environment: ic.Environment, Customer: request.Customer, Breakdown: breakdown,
	})
	if errors.Is(err, ErrOrderHasDocument) {
		return ref, failure(apperror.Conflict, "order_has_fiscal_document", "Este pedido ya tiene un comprobante.")
	}
	if err != nil {
		return ref, failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos registrar el comprobante.")
	}
	_ = a.audit.Record(ctx, s, "fiscal_document.requested", "fiscal_document", id)
	return contracts.DocumentRef{ID: id, DocumentType: request.DocumentType, Label: j.DocumentLabel(request.DocumentType), Number: j.FormatNumber(series, number), Status: domain.StatusPending}, nil
}

func offered(j ports.Jurisdiction, documentType string) bool {
	for _, option := range j.DocumentTypes() {
		if option.Value == documentType {
			return true
		}
	}
	return false
}

func sourceLines(order ordercontracts.Order) ([]domain.SourceLine, error) {
	lines := make([]domain.SourceLine, 0, len(order.Items)+1)
	for _, item := range order.Items {
		qty, okQty := domain.ParseAmount(item.Qty)
		price, okPrice := domain.ParseAmount(item.UnitPrice)
		if !okQty || !okPrice {
			return nil, errors.New("invalid order item amount")
		}
		lines = append(lines, domain.SourceLine{Description: item.Name, UnitCode: "NIU", Quantity: qty, UnitPrice: price})
	}
	if fee, ok := domain.ParseAmount(order.DeliveryFee); ok && fee.Sign() > 0 {
		lines = append(lines, domain.SourceLine{Description: "Delivery", UnitCode: "ZZ", Quantity: big.NewRat(1, 1), UnitPrice: fee})
	}
	return lines, nil
}

func (a *Documents) Dispatch(documentID string) {
	if a.dispatcher != nil && documentID != "" {
		a.dispatcher.Dispatch(documentID)
	}
}

// Process sends one document. It is safe to call repeatedly: only a PENDING
// or ERROR document (or a SENDING one abandoned for 5 minutes) is claimed.
func (a *Documents) Process(ctx context.Context, id string) error {
	claimed, ok, err := a.repo.ClaimDocument(ctx, id)
	if err != nil || !ok {
		return err
	}
	doc := &claimed.Document
	if j, ok := a.registry.Jurisdiction(doc.Country); ok {
		doc.FiscalCode, doc.FormattedNumber = j.FiscalCode(doc.DocumentType), j.FormatNumber(doc.Series, doc.Number)
		doc.CustomerFiscalCode = j.CustomerFiscalCode(doc.Customer.DocumentType)
	}
	retry := domain.RetryDelayMinutes(claimed.Attempts)
	provider, found := a.registry.Provider(claimed.Provider)
	if !found {
		return a.repo.FailDocument(ctx, id, "Proveedor de facturación no configurado.", retry)
	}
	credentials, problem, err := a.documentCredentials(ctx, claimed, provider)
	if err != nil {
		return err
	}
	if problem != "" {
		return a.repo.FailDocument(ctx, id, problem, retry)
	}
	doc.Credentials = credentials
	result, err := provider.Issue(ctx, *doc)
	// The signed XML and its hash are kept even when the send fails: the
	// printed representation only needs the signature, not the answer.
	if len(result.Files) > 0 || result.DigestValue != "" {
		if saveErr := a.repo.SaveFiles(ctx, claimed.OrganizationID, id, result.DigestValue, result.Files); saveErr != nil {
			return a.repo.FailDocument(ctx, id, "No se pudo guardar el XML o la constancia de SUNAT.", retry)
		}
	}
	if err != nil {
		return a.repo.FailDocument(ctx, id, err.Error(), retry)
	}
	if result.Status != domain.StatusAccepted && result.Status != domain.StatusObserved {
		result.Status = domain.StatusRejected
	}
	return a.repo.CompleteDocument(ctx, id, result)
}

// documentCredentials decrypts the profile's credentials only for this send.
// A missing or unreadable credential leaves the document retryable with a
// message the administrator can act on.
func (a *Documents) documentCredentials(ctx context.Context, claimed ClaimedDocument, provider ports.Provider) (map[string]string, string, error) {
	sealed, found, err := a.repo.DocumentCredentials(ctx, claimed.OrganizationID, claimed.FiscalProfileID, provider.Key())
	if err != nil {
		return nil, "", err
	}
	if !found {
		if provider.RequiresCredentials(claimed.Document.Environment) {
			return nil, "Configura las credenciales de " + provider.Label() + " en Facturación.", nil
		}
		return map[string]string{}, "", nil
	}
	values, err := openCredentials(a.cipher, sealed, claimed.OrganizationID, claimed.FiscalProfileID, provider.Key())
	if err != nil {
		return nil, "No se pudieron descifrar las credenciales de facturación.", nil
	}
	return values, "", nil
}

// File returns a stored artifact (signed XML or CDR) of the session location.
func (a *Documents) File(ctx context.Context, s scope, id, kind string) (ports.File, error) {
	file, err := a.repo.File(ctx, s, id, kind)
	if errors.Is(err, ErrNotFound) {
		return file, failure(apperror.NotFound, "fiscal_document_file_not_found", "El archivo no está disponible para este comprobante.")
	}
	if err != nil {
		return file, failure(apperror.Unavailable, "fiscal_documents_unavailable", "No pudimos leer el archivo.")
	}
	return file, nil
}

// Sweep processes due documents; Cloud Scheduler calls it as a safety net.
func (a *Documents) Sweep(ctx context.Context, limit int) (int, error) {
	ids, err := a.repo.DueDocuments(ctx, limit)
	if err != nil {
		return 0, err
	}
	for _, id := range ids {
		if err := a.Process(ctx, id); err != nil {
			return 0, err
		}
	}
	return len(ids), nil
}

type DocumentSummary struct {
	ID               string  `json:"id"`
	DocumentType     string  `json:"documentType"`
	Label            string  `json:"label"`
	Number           string  `json:"number"`
	IssueDate        string  `json:"issueDate"`
	CustomerName     string  `json:"customerName"`
	CustomerDocument string  `json:"customerDocument"`
	Currency         string  `json:"currency"`
	Total            string  `json:"total"`
	Status           string  `json:"status"`
	StatusMessage    string  `json:"statusMessage"`
	OrderID          string  `json:"orderId"`
	OrderCode        string  `json:"orderCode"`
	CreatedAt        string  `json:"createdAt"`
	AcceptedAt       *string `json:"acceptedAt"`
	CanRetry         bool    `json:"canRetry"`
}
type DocumentDetail struct {
	DocumentSummary
	Customer          domain.Customer  `json:"customer"`
	Breakdown         domain.Breakdown `json:"breakdown"`
	ProviderLabel     string           `json:"providerLabel"`
	Environment       string           `json:"environment"`
	ProviderReference string           `json:"providerReference"`
	PDFURL            string           `json:"pdfUrl"`
	XMLURL            string           `json:"xmlUrl"`
	CDRURL            string           `json:"cdrUrl"`
	DigestValue       string           `json:"digestValue"`
	ProviderCode      string           `json:"providerCode"`
	Files             []string         `json:"files"`
	Attempts          int              `json:"attempts"`
	NextAttemptAt     *string          `json:"nextAttemptAt"`
	// Printout is null until the document is signed, and for rejected or
	// cancelled documents, which have no fiscal value to print.
	Printout *ports.Printout `json:"printout"`
}

const testEnvironmentNotice = "Ambiente de pruebas: sin validez tributaria."

// printout builds the printed representation once the document is signed.
func (a *Documents) printout(ctx context.Context, s scope, row DocumentRow, breakdown domain.Breakdown) (*ports.Printout, error) {
	if row.DigestValue == "" || row.Status == domain.StatusRejected || row.Status == domain.StatusCancelled {
		return nil, nil
	}
	j, ok := a.registry.Jurisdiction(row.Country)
	if !ok {
		return nil, nil
	}
	issuer, err := a.repo.Issuer(ctx, s)
	if err != nil {
		return nil, err
	}
	out := j.Printout(ports.PrintSource{
		DocumentType: row.DocumentType, Series: row.Series, Number: row.Number, IssueDate: row.IssueDate, Currency: row.Currency,
		Issuer: issuer, Customer: row.Customer, TaxTotal: breakdown.TaxTotal, Total: breakdown.Total, DigestValue: row.DigestValue,
	})
	if row.Environment == domain.EnvironmentTest {
		out.Notice = testEnvironmentNotice
	}
	return &out, nil
}

type DocumentsPage struct {
	Items         []DocumentSummary `json:"items"`
	Total         int               `json:"total"`
	Page          int               `json:"page"`
	PageSize      int               `json:"pageSize"`
	StatusOptions []ports.Option    `json:"statusOptions"`
}

var statusOptions = []ports.Option{
	{Value: domain.StatusPending, Label: "Emitiendo"}, {Value: domain.StatusAccepted, Label: "Aceptado"},
	{Value: domain.StatusObserved, Label: "Observado"}, {Value: domain.StatusRejected, Label: "Rechazado"},
	{Value: domain.StatusError, Label: "Con error"},
}

func (a *Documents) summary(row DocumentRow) DocumentSummary {
	label, number := row.DocumentType, row.Series
	if j, ok := a.registry.Jurisdiction(row.Country); ok {
		label, number = j.DocumentLabel(row.DocumentType), j.FormatNumber(row.Series, row.Number)
	}
	customerDocument := ""
	if row.Customer.DocumentType != "" {
		customerDocument = row.Customer.DocumentType + " " + row.Customer.DocumentNumber
	}
	return DocumentSummary{
		ID: row.ID, DocumentType: row.DocumentType, Label: label, Number: number, IssueDate: row.IssueDate,
		CustomerName: row.Customer.Name, CustomerDocument: customerDocument, Currency: row.Currency, Total: row.Total,
		Status: row.Status, StatusMessage: row.ProviderMessage, OrderID: row.OrderID, OrderCode: row.OrderCode,
		CreatedAt: row.CreatedAt, AcceptedAt: row.AcceptedAt, CanRetry: row.Status == domain.StatusError,
	}
}

func (a *Documents) List(ctx context.Context, s scope, f DocumentFilter) (DocumentsPage, error) {
	out := DocumentsPage{Items: []DocumentSummary{}, Page: f.Page, PageSize: f.Size, StatusOptions: statusOptions}
	if f.Status == domain.StatusPending {
		f.Status = "PENDING,SENDING"
	}
	rows, total, err := a.repo.Documents(ctx, s, f)
	if err != nil {
		return out, failure(apperror.Unavailable, "fiscal_documents_unavailable", "No pudimos cargar los comprobantes.")
	}
	for _, row := range rows {
		out.Items = append(out.Items, a.summary(row))
	}
	out.Total = total
	return out, nil
}

func (a *Documents) Get(ctx context.Context, s scope, id string) (DocumentDetail, error) {
	row, breakdown, err := a.repo.Document(ctx, s, id)
	if errors.Is(err, ErrNotFound) {
		return DocumentDetail{}, failure(apperror.NotFound, "fiscal_document_not_found", "No encontramos el comprobante.")
	}
	if err != nil {
		return DocumentDetail{}, failure(apperror.Unavailable, "fiscal_documents_unavailable", "No pudimos cargar el comprobante.")
	}
	providerLabel := row.Provider
	if p, ok := a.registry.Provider(row.Provider); ok {
		providerLabel = p.Label()
	}
	printout, err := a.printout(ctx, s, row, breakdown)
	if err != nil {
		return DocumentDetail{}, failure(apperror.Unavailable, "fiscal_documents_unavailable", "No pudimos preparar la representación impresa.")
	}
	return DocumentDetail{
		DocumentSummary: a.summary(row), Customer: row.Customer, Breakdown: breakdown, ProviderLabel: providerLabel,
		Environment: row.Environment, ProviderReference: row.ProviderReference, PDFURL: row.PDFURL, XMLURL: row.XMLURL,
		CDRURL: row.CDRURL, DigestValue: row.DigestValue, ProviderCode: row.ProviderCode, Files: row.Files, Attempts: row.Attempts, NextAttemptAt: row.NextAttemptAt,
		Printout: printout,
	}, nil
}

// OrderDocument implements contracts.Issuer for the sale detail: reprints
// show the document delivered at checkout instead of the payment ticket.
func (a *Documents) OrderDocument(ctx context.Context, s scope, orderID string) (*contracts.DocumentRef, error) {
	row, err := a.repo.OrderDocument(ctx, s, orderID)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, failure(apperror.Unavailable, "fiscal_documents_unavailable", "No pudimos leer el comprobante del pedido.")
	}
	summary := a.summary(row)
	return &contracts.DocumentRef{ID: summary.ID, DocumentType: summary.DocumentType, Label: summary.Label, Number: summary.Number, Status: summary.Status}, nil
}

// Retry sends again a document that failed for a temporary reason.
func (a *Documents) Retry(ctx context.Context, s scope, id string) (DocumentDetail, error) {
	command, err := a.repo.RetryDocument(ctx, s, id)
	if err != nil {
		return DocumentDetail{}, failure(apperror.Unavailable, "fiscal_documents_unavailable", "No pudimos reintentar el comprobante.")
	}
	if command == 0 {
		return DocumentDetail{}, failure(apperror.Conflict, "fiscal_document_not_retryable", "Solo se reintentan comprobantes con error de envío.")
	}
	_ = a.audit.Record(ctx, s, "fiscal_document.retried", "fiscal_document", id)
	a.Dispatch(id)
	return a.Get(ctx, s, id)
}

type CheckoutOptions struct {
	Enabled               bool                           `json:"enabled"`
	DefaultDocument       string                         `json:"defaultDocument"`
	DocumentTypes         []ports.Option                 `json:"documentTypes"`
	CustomerDocumentTypes []ports.CustomerDocumentOption `json:"customerDocumentTypes"`
	IdentityThreshold     string                         `json:"identityThreshold"`
	InvoiceCustomerType   string                         `json:"invoiceCustomerDocumentType"`
	Currency              string                         `json:"currency"`
	// Issue explains why an active module cannot issue yet at this location
	// (issuer, credentials or series); empty when not entitled or unsupported.
	Issue string `json:"issue,omitempty"`
}

// CheckoutOptions tells the POS whether to offer documents and which ones.
func (a *Documents) CheckoutOptions(ctx context.Context, s scope) (CheckoutOptions, error) {
	off := CheckoutOptions{DefaultDocument: domain.TypeNone, DocumentTypes: []ports.Option{}, CustomerDocumentTypes: []ports.CustomerDocumentOption{}}
	ic, j, err := a.issueSetup(ctx, s)
	var known *apperror.Error
	if errors.As(err, &known) && known.Kind == apperror.Conflict {
		if pendingSetupCodes[known.Code] {
			off.Issue = known.Message
		}
		return off, nil
	}
	if err != nil {
		return off, err
	}
	types := append([]ports.Option{{Value: domain.TypeNone, Label: "Sin comprobante"}}, j.DocumentTypes()...)
	return CheckoutOptions{Enabled: true, DefaultDocument: ic.DefaultDocument, DocumentTypes: types, CustomerDocumentTypes: j.CustomerDocumentTypes(), IdentityThreshold: j.IdentityThreshold(), InvoiceCustomerType: j.InvoiceCustomerDocumentType(), Currency: ic.Currency}, nil
}
