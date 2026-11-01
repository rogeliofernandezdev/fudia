package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/application"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/domain"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/ports"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type scope = requestscope.Scope
type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool} }

var _ application.Repository = (*Repository)(nil)

func (r *Repository) db(ctx context.Context) database.Executor {
	return database.ExecutorFor(ctx, r.pool)
}

func classify(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return application.ErrNotFound
	case errors.As(err, &pgErr) && pgErr.Code == "22P02":
		return application.ErrNotFound
	case errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "fiscal_documents_one_per_order_idx":
		return application.ErrOrderHasDocument
	case errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "einvoicing_series_organization_id_series_key":
		return application.ErrSeriesTaken
	}
	return err
}

func (r *Repository) IssueContext(ctx context.Context, s scope) (application.IssueContext, error) {
	var out application.IssueContext
	err := r.db(ctx).QueryRow(ctx, `
		SELECT p.id,p.country_code,p.currency,(now() AT TIME ZONE l.timezone)::date::text,
		       COALESCE(e.enabled,false),COALESCE(e.provider,''),COALESCE(e.environment,'test'),COALESCE(ls.default_document,'none')
		FROM locations l
		JOIN organization_fiscal_profiles p ON p.id=l.fiscal_profile_id AND p.organization_id=l.organization_id
		LEFT JOIN einvoicing_profile_settings e ON e.fiscal_profile_id=p.id
		LEFT JOIN einvoicing_location_settings ls ON ls.location_id=l.id
		WHERE l.id=NULLIF($1,'')::uuid AND l.organization_id=$2`, s.LocationID, s.OrganizationID).
		Scan(&out.FiscalProfileID, &out.Country, &out.Currency, &out.BusinessDate, &out.Enabled, &out.Provider, &out.Environment, &out.DefaultDocument)
	return out, classify(err)
}

// NextNumber reserves the next correlative; the UPDATE locks the series row
// until the payment transaction ends, so numbers never repeat.
func (r *Repository) NextNumber(ctx context.Context, s scope, documentType string) (string, int, error) {
	var series string
	var number int
	err := r.db(ctx).QueryRow(ctx, `
		UPDATE einvoicing_series SET last_number=last_number+1,updated_at=now()
		WHERE organization_id=$1 AND location_id=$2 AND document_type=$3
		RETURNING series,last_number`, s.OrganizationID, s.LocationID, documentType).Scan(&series, &number)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", 0, application.ErrSeriesMissing
	}
	return series, number, err
}

func (r *Repository) InsertDocument(ctx context.Context, s scope, in application.NewDocument) (string, error) {
	var id string
	b := in.Breakdown
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO fiscal_documents(organization_id,location_id,fiscal_profile_id,order_id,country_code,document_type,series,number,
		  issue_date,currency,customer_document_type,customer_document_number,customer_name,customer_address,
		  net_total,tax_total,total,provider,environment,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,NULLIF($20,'')::uuid)
		RETURNING id`,
		s.OrganizationID, s.LocationID, in.FiscalProfileID, in.OrderID, in.Country, in.DocumentType, in.Series, in.Number,
		in.IssueDate, in.Currency, in.Customer.DocumentType, in.Customer.DocumentNumber, in.Customer.Name, in.Customer.Address,
		b.NetTotal, b.TaxTotal, b.Total, in.Provider, in.Environment, s.UserID).Scan(&id)
	if err != nil {
		return "", classify(err)
	}
	descriptions, units, quantities, prices, nets, taxes, totals := []string{}, []string{}, []string{}, []string{}, []string{}, []string{}, []string{}
	for _, l := range b.Lines {
		descriptions, units, quantities, prices = append(descriptions, l.Description), append(units, l.UnitCode), append(quantities, l.Quantity), append(prices, l.UnitPrice)
		nets, taxes, totals = append(nets, l.NetAmount), append(taxes, l.TaxAmount), append(totals, l.Total)
	}
	if _, err = r.db(ctx).Exec(ctx, `
		INSERT INTO fiscal_document_lines(document_id,organization_id,line_no,description,unit_code,quantity,unit_price,net_amount,tax_amount,total)
		SELECT $1,$2,l.ord,l.description,l.unit,l.quantity::numeric,l.price::numeric,l.net::numeric,l.tax::numeric,l.total::numeric
		FROM unnest($3::text[],$4::text[],$5::text[],$6::text[],$7::text[],$8::text[],$9::text[]) WITH ORDINALITY AS l(description,unit,quantity,price,net,tax,total,ord)`,
		id, s.OrganizationID, descriptions, units, quantities, prices, nets, taxes, totals); err != nil {
		return "", err
	}
	codes, names, rates, amounts := []string{}, []string{}, []string{}, []string{}
	for _, t := range b.Taxes {
		codes, names, rates, amounts = append(codes, t.Code), append(names, t.Name), append(rates, t.Rate), append(amounts, t.Amount)
	}
	_, err = r.db(ctx).Exec(ctx, `
		INSERT INTO fiscal_document_taxes(document_id,organization_id,code,name,rate,amount,sort_order)
		SELECT $1,$2,t.code,t.name,t.rate::numeric,t.amount::numeric,(t.ord-1)::smallint
		FROM unnest($3::text[],$4::text[],$5::text[],$6::text[]) WITH ORDINALITY AS t(code,name,rate,amount,ord)`,
		id, s.OrganizationID, codes, names, rates, amounts)
	return id, err
}

const documentColumns = `d.id,d.country_code,d.document_type,d.series,d.number,d.issue_date::text,d.currency,
	d.customer_document_type,d.customer_document_number,d.customer_name,d.customer_address,
	d.net_total::text,d.tax_total::text,d.total::text,d.status,d.provider,d.environment,
	COALESCE(d.provider_reference,''),COALESCE(d.provider_code,''),COALESCE(d.provider_message,''),
	COALESCE(d.pdf_url,''),COALESCE(d.xml_url,''),COALESCE(d.cdr_url,''),d.attempts,
	CASE WHEN d.status='ERROR' THEN to_char(d.next_attempt_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') END,
	d.order_id::text,o.code,to_char(d.created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),
	to_char(d.accepted_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"')`

func scanDocument(row pgx.Row, d *application.DocumentRow) error {
	return row.Scan(documentTargets(d)...)
}

// ClaimDocument atomically moves a due document to SENDING for the worker.
func (r *Repository) ClaimDocument(ctx context.Context, id string) (application.ClaimedDocument, bool, error) {
	var out application.ClaimedDocument
	var row application.DocumentRow
	var organizationID, locationID, profileID string
	err := r.db(ctx).QueryRow(ctx, `
		WITH claimed AS (
		  UPDATE fiscal_documents SET status='SENDING',attempts=attempts+1,updated_at=now()
		  WHERE id=$1 AND (status IN ('PENDING','ERROR') OR (status='SENDING' AND updated_at<now()-interval '5 minutes'))
		  RETURNING *
		)
		SELECT d.organization_id::text,d.location_id::text,d.fiscal_profile_id::text,`+documentColumns+`
		FROM claimed d JOIN orders o ON o.id=d.order_id AND o.organization_id=d.organization_id`, id).
		Scan(append([]any{&organizationID, &locationID, &profileID}, documentTargets(&row)...)...)
	if err != nil {
		if errors.Is(classify(err), application.ErrNotFound) {
			return out, false, nil
		}
		return out, false, err
	}
	breakdown, err := r.breakdown(ctx, organizationID, row.ID, row)
	if err != nil {
		return out, false, err
	}
	var issuer ports.Issuer
	if err = r.db(ctx).QueryRow(ctx, `
		SELECT o.tax_id,o.legal_name,COALESCE(NULLIF(o.trade_name,''),o.legal_name),COALESCE(l.address,'')
		FROM organizations o JOIN locations l ON l.organization_id=o.id
		WHERE o.id=$1 AND l.id=$2`, organizationID, locationID).Scan(&issuer.TaxID, &issuer.LegalName, &issuer.TradeName, &issuer.Address); err != nil {
		return out, false, err
	}
	out.Provider, out.Attempts, out.OrganizationID, out.FiscalProfileID = row.Provider, row.Attempts, organizationID, profileID
	out.Document = ports.Document{ID: row.ID, Country: row.Country, DocumentType: row.DocumentType, Series: row.Series, Number: row.Number,
		IssueDate: row.IssueDate, Currency: row.Currency, Environment: row.Environment, Issuer: issuer, Customer: row.Customer, Breakdown: breakdown}
	return out, true, nil
}

func documentTargets(d *application.DocumentRow) []any {
	return []any{&d.ID, &d.Country, &d.DocumentType, &d.Series, &d.Number, &d.IssueDate, &d.Currency,
		&d.Customer.DocumentType, &d.Customer.DocumentNumber, &d.Customer.Name, &d.Customer.Address,
		&d.NetTotal, &d.TaxTotal, &d.Total, &d.Status, &d.Provider, &d.Environment,
		&d.ProviderReference, &d.ProviderCode, &d.ProviderMessage, &d.PDFURL, &d.XMLURL, &d.CDRURL, &d.Attempts,
		&d.NextAttemptAt, &d.OrderID, &d.OrderCode, &d.CreatedAt, &d.AcceptedAt}
}

func (r *Repository) breakdown(ctx context.Context, organizationID, id string, row application.DocumentRow) (domain.Breakdown, error) {
	out := domain.Breakdown{Lines: []domain.Line{}, Taxes: []domain.TaxAmount{}, NetTotal: row.NetTotal, TaxTotal: row.TaxTotal, Total: row.Total}
	rows, err := r.db(ctx).Query(ctx, `SELECT line_no,description,unit_code,quantity::text,unit_price::text,net_amount::text,tax_amount::text,total::text
		FROM fiscal_document_lines WHERE document_id=$1 AND organization_id=$2 ORDER BY line_no`, id, organizationID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var l domain.Line
		if err = rows.Scan(&l.No, &l.Description, &l.UnitCode, &l.Quantity, &l.UnitPrice, &l.NetAmount, &l.TaxAmount, &l.Total); err != nil {
			rows.Close()
			return out, err
		}
		out.Lines = append(out.Lines, l)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return out, err
	}
	rows, err = r.db(ctx).Query(ctx, `SELECT code,name,rate::text,amount::text FROM fiscal_document_taxes
		WHERE document_id=$1 AND organization_id=$2 ORDER BY sort_order`, id, organizationID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var t domain.TaxAmount
		if err = rows.Scan(&t.Code, &t.Name, &t.Rate, &t.Amount); err != nil {
			return out, err
		}
		out.Taxes = append(out.Taxes, t)
	}
	return out, rows.Err()
}

func (r *Repository) CompleteDocument(ctx context.Context, id string, result ports.Result) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE fiscal_documents SET status=$2,provider_reference=NULLIF($3,''),provider_code=NULLIF($4,''),provider_message=NULLIF($5,''),
		  pdf_url=NULLIF($6,''),xml_url=NULLIF($7,''),cdr_url=NULLIF($8,''),
		  accepted_at=CASE WHEN $2 IN ('ACCEPTED','OBSERVED') THEN now() END,updated_at=now()
		WHERE id=$1 AND status='SENDING'`, id, result.Status, result.Reference, result.Code, result.Message, result.PDFURL, result.XMLURL, result.CDRURL)
	return err
}
func (r *Repository) FailDocument(ctx context.Context, id, message string, retryMinutes int) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE fiscal_documents SET status='ERROR',provider_message=$2,next_attempt_at=now()+make_interval(mins=>$3),updated_at=now()
		WHERE id=$1 AND status='SENDING'`, id, message, retryMinutes)
	return err
}
func (r *Repository) DueDocuments(ctx context.Context, limit int) ([]string, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT id::text FROM fiscal_documents
		WHERE (status IN ('PENDING','ERROR') AND next_attempt_at<=now()) OR (status='SENDING' AND updated_at<now()-interval '5 minutes')
		ORDER BY next_attempt_at LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *Repository) Documents(ctx context.Context, s scope, f application.DocumentFilter) ([]application.DocumentRow, int, error) {
	statuses := []string{}
	for _, status := range strings.Split(f.Status, ",") {
		if status = strings.TrimSpace(status); status != "" {
			statuses = append(statuses, status)
		}
	}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT `+documentColumns+`,count(*) OVER()
		FROM fiscal_documents d JOIN orders o ON o.id=d.order_id AND o.organization_id=d.organization_id
		WHERE d.organization_id=$1 AND d.location_id=NULLIF($2,'')::uuid
		  AND (cardinality($3::text[])=0 OR d.status=ANY($3))
		  AND ($4='' OR d.series||'-'||d.number ILIKE '%'||$4||'%' OR d.customer_name ILIKE '%'||$4||'%'
		       OR d.customer_document_number ILIKE '%'||$4||'%' OR o.code ILIKE '%'||$4||'%')
		ORDER BY d.created_at DESC,d.id
		LIMIT $5 OFFSET $6`, s.OrganizationID, s.LocationID, statuses, strings.TrimSpace(f.Query), f.Size, (f.Page-1)*f.Size)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items, total := []application.DocumentRow{}, 0
	for rows.Next() {
		var d application.DocumentRow
		if err = rows.Scan(append(documentTargets(&d), &total)...); err != nil {
			return nil, 0, err
		}
		items = append(items, d)
	}
	return items, total, rows.Err()
}

func (r *Repository) Document(ctx context.Context, s scope, id string) (application.DocumentRow, domain.Breakdown, error) {
	var d application.DocumentRow
	err := scanDocument(r.db(ctx).QueryRow(ctx, `
		SELECT `+documentColumns+`
		FROM fiscal_documents d JOIN orders o ON o.id=d.order_id AND o.organization_id=d.organization_id
		WHERE d.id=$1 AND d.organization_id=$2 AND d.location_id=NULLIF($3,'')::uuid`, id, s.OrganizationID, s.LocationID), &d)
	if err != nil {
		return d, domain.Breakdown{}, classify(err)
	}
	if err := r.db(ctx).QueryRow(ctx, `
		SELECT COALESCE(d.digest_value,''),COALESCE(array_agg(f.kind ORDER BY f.kind) FILTER (WHERE f.kind IS NOT NULL),'{}')
		FROM fiscal_documents d LEFT JOIN fiscal_document_files f ON f.document_id=d.id AND f.organization_id=d.organization_id
		WHERE d.id=$1 AND d.organization_id=$2 GROUP BY d.id`, d.ID, s.OrganizationID).Scan(&d.DigestValue, &d.Files); err != nil {
		return d, domain.Breakdown{}, err
	}
	b, err := r.breakdown(ctx, s.OrganizationID, d.ID, d)
	return d, b, err
}

func (r *Repository) RetryDocument(ctx context.Context, s scope, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE fiscal_documents SET status='PENDING',next_attempt_at=now(),updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=NULLIF($3,'')::uuid AND status='ERROR'`, id, s.OrganizationID, s.LocationID)
	if err != nil {
		if errors.Is(classify(err), application.ErrNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *Repository) Settings(ctx context.Context, s scope) ([]application.ProfileSettingRow, []application.LocationSettingRow, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT p.id::text,p.country_code,COALESCE(c.name,p.country_code),p.currency,p.is_default,COALESCE(e.enabled,false),COALESCE(e.provider,''),COALESCE(e.environment,'test')
		FROM organization_fiscal_profiles p LEFT JOIN einvoicing_profile_settings e ON e.fiscal_profile_id=p.id
		LEFT JOIN platform_countries c ON c.code=p.country_code
		WHERE p.organization_id=$1 AND p.active ORDER BY p.is_default DESC,p.country_code`, s.OrganizationID)
	if err != nil {
		return nil, nil, err
	}
	profiles := []application.ProfileSettingRow{}
	for rows.Next() {
		var p application.ProfileSettingRow
		if err = rows.Scan(&p.FiscalProfileID, &p.Country, &p.CountryName, &p.Currency, &p.Default, &p.Enabled, &p.Provider, &p.Environment); err != nil {
			rows.Close()
			return nil, nil, err
		}
		profiles = append(profiles, p)
	}
	rows.Close()
	rows, err = r.db(ctx).Query(ctx, `
		SELECT l.id::text,l.name,l.fiscal_profile_id::text,p.country_code,COALESCE(ls.default_document,'none'),
		       COALESCE(s.document_type,''),COALESCE(s.series,''),COALESCE(s.last_number,0)
		FROM locations l
		JOIN organization_fiscal_profiles p ON p.id=l.fiscal_profile_id AND p.organization_id=l.organization_id
		LEFT JOIN einvoicing_location_settings ls ON ls.location_id=l.id
		LEFT JOIN einvoicing_series s ON s.location_id=l.id
		WHERE l.organization_id=$1 AND l.active
		ORDER BY l.created_at,l.id`, s.OrganizationID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	locations := []application.LocationSettingRow{}
	positions := map[string]int{}
	for rows.Next() {
		var id, name, profile, country, defaultDocument, documentType, series string
		var last int
		if err = rows.Scan(&id, &name, &profile, &country, &defaultDocument, &documentType, &series, &last); err != nil {
			return nil, nil, err
		}
		if len(locations) == 0 || locations[len(locations)-1].LocationID != id {
			positions[country]++
			locations = append(locations, application.LocationSettingRow{LocationID: id, Name: name, FiscalProfileID: profile, Country: country,
				Position: positions[country], DefaultDocument: defaultDocument, Series: map[string]application.SeriesRow{}})
		}
		if documentType != "" {
			locations[len(locations)-1].Series[documentType] = application.SeriesRow{Series: series, LastNumber: last}
		}
	}
	return profiles, locations, rows.Err()
}

func (r *Repository) ProfileCountry(ctx context.Context, s scope, profileID string) (string, error) {
	var country string
	err := r.db(ctx).QueryRow(ctx, `SELECT country_code FROM organization_fiscal_profiles WHERE id=$1 AND organization_id=$2 AND active`, profileID, s.OrganizationID).Scan(&country)
	return country, classify(err)
}
func (r *Repository) SaveProfileSettings(ctx context.Context, s scope, profileID string, in application.ProfileSettingsInput) error {
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO einvoicing_profile_settings(fiscal_profile_id,organization_id,enabled,provider,environment,updated_by)
		VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::uuid)
		ON CONFLICT (fiscal_profile_id) DO UPDATE SET enabled=excluded.enabled,provider=excluded.provider,
		  environment=excluded.environment,updated_by=excluded.updated_by,updated_at=now()`,
		profileID, s.OrganizationID, in.Enabled, in.Provider, in.Environment, s.UserID)
	return err
}
func (r *Repository) LocationCountry(ctx context.Context, s scope, locationID string) (string, error) {
	var country string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT p.country_code FROM locations l
		JOIN organization_fiscal_profiles p ON p.id=l.fiscal_profile_id AND p.organization_id=l.organization_id
		WHERE l.id=$1 AND l.organization_id=$2 AND l.active`, locationID, s.OrganizationID).Scan(&country)
	return country, classify(err)
}

// SaveLocationSettings runs inside a transaction: a series that already issued
// documents keeps its name, and a series name is unique within the company.
func (r *Repository) SaveLocationSettings(ctx context.Context, s scope, locationID string, in application.LocationSettingsInput) error {
	if _, err := r.db(ctx).Exec(ctx, `
		INSERT INTO einvoicing_location_settings(location_id,organization_id,default_document) VALUES($1,$2,$3)
		ON CONFLICT (location_id) DO UPDATE SET default_document=excluded.default_document,updated_at=now()`,
		locationID, s.OrganizationID, in.DefaultDocument); err != nil {
		return classify(err)
	}
	for documentType, series := range in.Series {
		var current string
		var last int
		err := r.db(ctx).QueryRow(ctx, `SELECT series,last_number FROM einvoicing_series
			WHERE location_id=$1 AND organization_id=$2 AND document_type=$3 FOR UPDATE`, locationID, s.OrganizationID, documentType).Scan(&current, &last)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			_, err = r.db(ctx).Exec(ctx, `INSERT INTO einvoicing_series(organization_id,location_id,document_type,series) VALUES($1,$2,$3,$4)`,
				s.OrganizationID, locationID, documentType, series)
		case err != nil:
		case current == series:
		case last > 0:
			return application.ErrSeriesInUse
		default:
			_, err = r.db(ctx).Exec(ctx, `UPDATE einvoicing_series SET series=$1,updated_at=now() WHERE location_id=$2 AND document_type=$3`, series, locationID, documentType)
		}
		if err != nil {
			return classify(err)
		}
	}
	return nil
}

func (r *Repository) SaveFiles(ctx context.Context, organizationID, documentID, digest string, files []ports.File) error {
	if _, err := r.db(ctx).Exec(ctx, `UPDATE fiscal_documents SET digest_value=NULLIF($3,''),updated_at=now() WHERE id=$1 AND organization_id=$2`, documentID, organizationID, digest); err != nil {
		return err
	}
	for _, f := range files {
		if _, err := r.db(ctx).Exec(ctx, `
			INSERT INTO fiscal_document_files(document_id,organization_id,kind,file_name,content_type,content) VALUES($1,$2,$3,$4,$5,$6)
			ON CONFLICT (document_id,kind) DO UPDATE SET file_name=excluded.file_name,content_type=excluded.content_type,content=excluded.content,created_at=now()`,
			documentID, organizationID, f.Kind, f.Name, f.ContentType, f.Content); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) File(ctx context.Context, s scope, documentID, kind string) (ports.File, error) {
	var f ports.File
	err := r.db(ctx).QueryRow(ctx, `
		SELECT f.kind,f.file_name,f.content_type,f.content FROM fiscal_document_files f
		JOIN fiscal_documents d ON d.id=f.document_id AND d.organization_id=f.organization_id
		WHERE f.document_id=$1 AND f.organization_id=$2 AND d.location_id=NULLIF($3,'')::uuid AND f.kind=$4`,
		documentID, s.OrganizationID, s.LocationID, kind).Scan(&f.Kind, &f.Name, &f.ContentType, &f.Content)
	return f, classify(err)
}

func scanCredentials(row pgx.Row) (application.SealedCredentials, bool, error) {
	var out application.SealedCredentials
	var metadata []byte
	err := row.Scan(&out.KeyVersion, &out.Nonce, &out.Ciphertext, &metadata, &out.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, false, nil
	}
	if err != nil {
		return out, false, classifyRead(err)
	}
	out.Metadata = map[string]string{}
	return out, true, json.Unmarshal(metadata, &out.Metadata)
}

// classifyRead turns a malformed id into "no rows" for optional reads.
func classifyRead(err error) error {
	if errors.Is(classify(err), application.ErrNotFound) {
		return nil
	}
	return err
}

const credentialColumns = `key_version,nonce,ciphertext,metadata,to_char(updated_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"')`

func (r *Repository) Credentials(ctx context.Context, s scope, profileID, provider string) (application.SealedCredentials, bool, error) {
	return scanCredentials(r.db(ctx).QueryRow(ctx, `SELECT `+credentialColumns+` FROM einvoicing_credentials
		WHERE scope='organization' AND organization_id=$1 AND fiscal_profile_id=$2 AND provider=$3`, s.OrganizationID, profileID, provider))
}
func (r *Repository) DocumentCredentials(ctx context.Context, organizationID, profileID, provider string) (application.SealedCredentials, bool, error) {
	return scanCredentials(r.db(ctx).QueryRow(ctx, `SELECT `+credentialColumns+` FROM einvoicing_credentials
		WHERE scope='organization' AND organization_id=$1 AND fiscal_profile_id=$2 AND provider=$3`, organizationID, profileID, provider))
}
func (r *Repository) SaveCredentials(ctx context.Context, s scope, profileID, provider string, sealed application.SealedCredentials) error {
	metadata, err := json.Marshal(sealed.Metadata)
	if err != nil {
		return err
	}
	_, err = r.db(ctx).Exec(ctx, `
		INSERT INTO einvoicing_credentials(scope,organization_id,fiscal_profile_id,provider,key_version,nonce,ciphertext,metadata,updated_by)
		VALUES('organization',$1,$2,$3,$4,$5,$6,$7,NULLIF($8,'')::uuid)
		ON CONFLICT (fiscal_profile_id,provider) WHERE scope='organization' DO UPDATE SET key_version=excluded.key_version,nonce=excluded.nonce,
		  ciphertext=excluded.ciphertext,metadata=excluded.metadata,updated_by=excluded.updated_by,updated_at=now()`,
		s.OrganizationID, profileID, provider, sealed.KeyVersion, sealed.Nonce, sealed.Ciphertext, metadata, s.UserID)
	return err
}

func (r *Repository) ProfileSetting(ctx context.Context, s scope, profileID string) (application.ProfileSettingRow, error) {
	var p application.ProfileSettingRow
	err := r.db(ctx).QueryRow(ctx, `
		SELECT p.id::text,p.country_code,p.currency,p.is_default,COALESCE(e.enabled,false),COALESCE(e.provider,''),COALESCE(e.environment,'test')
		FROM organization_fiscal_profiles p LEFT JOIN einvoicing_profile_settings e ON e.fiscal_profile_id=p.id
		WHERE p.id=$1 AND p.organization_id=$2 AND p.active`, profileID, s.OrganizationID).
		Scan(&p.FiscalProfileID, &p.Country, &p.Currency, &p.Default, &p.Enabled, &p.Provider, &p.Environment)
	return p, classify(err)
}

// Issuer reads the company identity and the session location address.
func (r *Repository) Issuer(ctx context.Context, s scope) (ports.Issuer, error) {
	var issuer ports.Issuer
	err := r.db(ctx).QueryRow(ctx, `
		SELECT o.tax_id,o.legal_name,COALESCE(NULLIF(o.trade_name,''),o.legal_name),
		       COALESCE((SELECT address FROM locations WHERE id=NULLIF($2,'')::uuid AND organization_id=o.id),'')
		FROM organizations o WHERE o.id=$1`, s.OrganizationID, s.LocationID).Scan(&issuer.TaxID, &issuer.LegalName, &issuer.TradeName, &issuer.Address)
	return issuer, classify(err)
}
