package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var errSubscriptionLimit = errors.New("subscription limit reached")

type subscriptionPlanView struct {
	ID           string   `json:"id"`
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Currency     string   `json:"currency"`
	MonthlyPrice string   `json:"monthlyPrice"`
	AnnualPrice  string   `json:"annualPrice"`
	TrialDays    int      `json:"trialDays"`
	MaxLocations *int     `json:"maxLocations"`
	MaxUsers     *int     `json:"maxUsers"`
	ModuleKeys   []string `json:"moduleKeys"`
	TermsVersion string   `json:"termsVersion"`
	Active       bool     `json:"active"`
}

type subscriptionPlanInput struct {
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Currency     string   `json:"currency"`
	MonthlyPrice string   `json:"monthlyPrice"`
	AnnualPrice  string   `json:"annualPrice"`
	TrialDays    int      `json:"trialDays"`
	MaxLocations *int     `json:"maxLocations"`
	MaxUsers     *int     `json:"maxUsers"`
	ModuleKeys   []string `json:"moduleKeys"`
	TermsVersion string   `json:"termsVersion"`
	Active       bool     `json:"active"`
}

type subscriptionPaymentView struct {
	ID                string  `json:"id"`
	Amount            string  `json:"amount"`
	Currency          string  `json:"currency"`
	Status            string  `json:"status"`
	Provider          string  `json:"provider"`
	ExternalReference *string `json:"externalReference"`
	PaidAt            *string `json:"paidAt"`
	CreatedAt         string  `json:"createdAt"`
}

type organizationSubscriptionView struct {
	ID                    string                    `json:"id"`
	Plan                  subscriptionPlanView      `json:"plan"`
	BillingCycle          string                    `json:"billingCycle"`
	PriceAmount           string                    `json:"priceAmount"`
	Currency              string                    `json:"currency"`
	Status                string                    `json:"status"`
	TrialStartsAt         *string                   `json:"trialStartsAt"`
	TrialEndsAt           *string                   `json:"trialEndsAt"`
	CurrentPeriodStartsAt *string                   `json:"currentPeriodStartsAt"`
	CurrentPeriodEndsAt   *string                   `json:"currentPeriodEndsAt"`
	RenewsAt              *string                   `json:"renewsAt"`
	AutoRenew             bool                      `json:"autoRenew"`
	TermsVersion          *string                   `json:"termsVersion"`
	TermsAcceptedAt       *string                   `json:"termsAcceptedAt"`
	Usage                 subscriptionUsage         `json:"usage"`
	Payments              []subscriptionPaymentView `json:"payments"`
}

type subscriptionUsage struct {
	Locations int `json:"locations"`
	Users     int `json:"users"`
}

type subscriptionUpdateInput struct {
	PlanID        string `json:"planId"`
	BillingCycle string `json:"billingCycle"`
	Status       string `json:"status"`
	AutoRenew    bool   `json:"autoRenew"`
	TermsAccepted bool  `json:"termsAccepted"`
}

type subscriptionPaymentInput struct {
	Amount            string `json:"amount"`
	Currency          string `json:"currency"`
	Status            string `json:"status"`
	Provider          string `json:"provider"`
	ExternalReference string `json:"externalReference"`
	PaidAt            string `json:"paidAt"`
}

var planCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,39}$`)

func normalizePlanInput(in *subscriptionPlanInput) bool {
	in.Code = strings.ToLower(strings.TrimSpace(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	in.MonthlyPrice = strings.TrimSpace(in.MonthlyPrice)
	in.AnnualPrice = strings.TrimSpace(in.AnnualPrice)
	in.TermsVersion = strings.TrimSpace(in.TermsVersion)
	if !planCodePattern.MatchString(in.Code) || in.Name == "" || len(in.Name) > 120 || len(in.Description) > 500 || in.TermsVersion == "" || len(in.TermsVersion) > 80 {
		return false
	}
	if _, ok := currencyByCode(in.Currency); !ok {
		return false
	}
	monthly, monthlyOK := new(big.Rat).SetString(in.MonthlyPrice)
	annual, annualOK := new(big.Rat).SetString(in.AnnualPrice)
	if !monthlyOK || !annualOK || monthly.Sign() < 0 || annual.Sign() < 0 || in.TrialDays < 0 || in.TrialDays > 365 {
		return false
	}
	if (in.MaxLocations != nil && *in.MaxLocations < 1) || (in.MaxUsers != nil && *in.MaxUsers < 1) {
		return false
	}
	seen := map[string]bool{}
	modules := make([]string, 0, len(in.ModuleKeys))
	for _, raw := range in.ModuleKeys {
		key := strings.TrimSpace(raw)
		if key == "" || seen[key] || !moduleCanActivate(key) {
			return false
		}
		seen[key] = true
		modules = append(modules, key)
	}
	if len(modules) == 0 {
		return false
	}
	in.ModuleKeys = modules
	return true
}

func scanSubscriptionPlan(row pgx.Row, out *subscriptionPlanView) error {
	return row.Scan(
		&out.ID, &out.Code, &out.Name, &out.Description, &out.Currency,
		&out.MonthlyPrice, &out.AnnualPrice, &out.TrialDays,
		&out.MaxLocations, &out.MaxUsers, &out.ModuleKeys, &out.TermsVersion, &out.Active,
	)
}

func readSubscriptionPlan(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, id string, activeOnly bool) (subscriptionPlanView, error) {
	var out subscriptionPlanView
	err := scanSubscriptionPlan(q.QueryRow(ctx, `
		SELECT id,code,name,description,currency,monthly_price::text,annual_price::text,
		       trial_days,max_locations,max_users,module_keys,terms_version,active
		FROM subscription_plans
		WHERE id=$1 AND (NOT $2 OR active)
	`, id, activeOnly), &out)
	return out, err
}

func (a *API) listSubscriptionPlans(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(r.Context(), `
		SELECT id,code,name,description,currency,monthly_price::text,annual_price::text,
		       trial_days,max_locations,max_users,module_keys,terms_version,active
		FROM subscription_plans
		ORDER BY active DESC,name
	`)
	if err != nil {
		fail(w, 503, "plans_unavailable", "No pudimos cargar los planes.")
		return
	}
	defer rows.Close()
	items := []subscriptionPlanView{}
	for rows.Next() {
		var item subscriptionPlanView
		if err = rows.Scan(&item.ID,&item.Code,&item.Name,&item.Description,&item.Currency,&item.MonthlyPrice,&item.AnnualPrice,&item.TrialDays,&item.MaxLocations,&item.MaxUsers,&item.ModuleKeys,&item.TermsVersion,&item.Active); err != nil {
			fail(w, 503, "plans_unavailable", "No pudimos cargar los planes.")
			return
		}
		items = append(items, item)
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

func (a *API) createSubscriptionPlan(w http.ResponseWriter, r *http.Request) {
	a.saveSubscriptionPlan(w, r, true)
}

func (a *API) updateSubscriptionPlan(w http.ResponseWriter, r *http.Request) {
	a.saveSubscriptionPlan(w, r, false)
}

func (a *API) saveSubscriptionPlan(w http.ResponseWriter, r *http.Request, create bool) {
	var in subscriptionPlanInput
	if json.NewDecoder(r.Body).Decode(&in) != nil || !normalizePlanInput(&in) {
		fail(w, 400, "invalid_plan", "Revisa código, precios, límites, módulos y versión de condiciones.")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "plans_unavailable", "No pudimos guardar el plan.")
		return
	}
	defer tx.Rollback(r.Context())

	var id string
	if create {
		err = tx.QueryRow(r.Context(), `
			INSERT INTO subscription_plans(code,name,description,currency,monthly_price,annual_price,trial_days,max_locations,max_users,module_keys,terms_version,active)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			RETURNING id
		`, in.Code,in.Name,in.Description,in.Currency,in.MonthlyPrice,in.AnnualPrice,in.TrialDays,in.MaxLocations,in.MaxUsers,in.ModuleKeys,in.TermsVersion,in.Active).Scan(&id)
	} else {
		id = r.PathValue("id")
		var exceeds bool
		err = tx.QueryRow(r.Context(), `
			SELECT EXISTS(
			  SELECT 1
			  FROM organization_subscriptions s
			  WHERE s.plan_id=$1
			    AND (
			      ($2::integer IS NOT NULL AND (SELECT count(*) FROM locations l WHERE l.organization_id=s.organization_id AND l.active)>$2)
			      OR
			      ($3::integer IS NOT NULL AND (SELECT count(*) FROM users u WHERE u.organization_id=s.organization_id AND u.active AND NOT u.platform_admin)>$3)
			    )
			)
		`, id,in.MaxLocations,in.MaxUsers).Scan(&exceeds)
		if err == nil && exceeds {
			fail(w, 409, "plan_limit_conflict", "El nuevo límite es menor que el uso actual de una empresa suscrita.")
			return
		}
		if err == nil {
			tag, updateErr := tx.Exec(r.Context(), `
				UPDATE subscription_plans
				SET code=$2,name=$3,description=$4,currency=$5,monthly_price=$6,annual_price=$7,
				    trial_days=$8,max_locations=$9,max_users=$10,module_keys=$11,terms_version=$12,active=$13,updated_at=now()
				WHERE id=$1 AND code<>'legacy'
			`, id,in.Code,in.Name,in.Description,in.Currency,in.MonthlyPrice,in.AnnualPrice,in.TrialDays,in.MaxLocations,in.MaxUsers,in.ModuleKeys,in.TermsVersion,in.Active)
			err = updateErr
			if err == nil && tag.RowsAffected() == 0 {
				err = pgx.ErrNoRows
			}
		}
	}
	if err != nil {
		fail(w, 409, "plan_conflict", "El código ya existe o el plan no puede modificarse.")
		return
	}

	if !create {
		rows, queryErr := tx.Query(r.Context(), `SELECT organization_id FROM organization_subscriptions WHERE plan_id=$1 AND status<>'cancelled'`, id)
		if queryErr != nil {
			fail(w, 503, "plans_unavailable", "No pudimos sincronizar el plan.")
			return
		}
		orgIDs := []string{}
		for rows.Next() {
			var orgID string
			if rows.Scan(&orgID) == nil {
				orgIDs = append(orgIDs, orgID)
			}
		}
		rows.Close()
		for _, orgID := range orgIDs {
			if err = syncOrganizationModulesForPlan(r.Context(), tx, orgID, in.ModuleKeys); err != nil {
				fail(w, 503, "plans_unavailable", "No pudimos sincronizar los módulos del plan.")
				return
			}
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "plans_unavailable", "No pudimos guardar el plan.")
		return
	}
	a.audit(r, map[bool]string{true:"subscription_plan.created",false:"subscription_plan.updated"}[create], "subscription_plan", id)
	plan, err := readSubscriptionPlan(r.Context(), a.db, id, false)
	if err != nil {
		fail(w, 503, "plans_unavailable", "El plan se guardó pero no pudimos volver a cargarlo.")
		return
	}
	writeJSON(w, map[bool]int{true:201,false:200}[create], plan)
}

func subscriptionPeriodEnd(start time.Time, cycle string) time.Time {
	if cycle == "annual" {
		return start.AddDate(1,0,0)
	}
	return start.AddDate(0,1,0)
}

func createOrganizationSubscription(ctx context.Context, tx pgx.Tx, organizationID, administratorID, planID, billingCycle string, termsAccepted bool) error {
	if billingCycle != "monthly" && billingCycle != "annual" {
		return errors.New("invalid billing cycle")
	}
	if !termsAccepted {
		return errors.New("terms not accepted")
	}
	plan, err := readSubscriptionPlan(ctx, tx, planID, true)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	status := "active"
	price := plan.MonthlyPrice
	var trialStarts, trialEnds *time.Time
	periodEnd := subscriptionPeriodEnd(now, billingCycle)
	if billingCycle == "annual" {
		price = plan.AnnualPrice
	}
	if plan.TrialDays > 0 {
		status = "trial"
		start := now
		end := now.AddDate(0,0,plan.TrialDays)
		trialStarts, trialEnds = &start, &end
		periodEnd = end
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO organization_subscriptions(
		  organization_id,plan_id,billing_cycle,price_amount,currency,status,
		  trial_starts_at,trial_ends_at,current_period_starts_at,current_period_ends_at,renews_at,
		  auto_renew,terms_version,terms_accepted_at,terms_accepted_by
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$10,true,$11,$9,$12)
	`, organizationID,plan.ID,billingCycle,price,plan.Currency,status,trialStarts,trialEnds,now,periodEnd,plan.TermsVersion,administratorID)
	if err != nil {
		return err
	}
	return syncOrganizationModulesForPlan(ctx, tx, organizationID, plan.ModuleKeys)
}

func (a *API) getOrganizationSubscription(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	_, _ = a.db.Exec(r.Context(), `
		UPDATE organization_subscriptions
		SET status='past_due',updated_at=now()
		WHERE organization_id=$1 AND status IN ('trial','active') AND renews_at IS NOT NULL AND renews_at<now()
	`, s.OrganizationID)

	var out organizationSubscriptionView
	var plan subscriptionPlanView
	err := a.db.QueryRow(r.Context(), `
		SELECT s.id,
		       p.id,p.code,p.name,p.description,p.currency,p.monthly_price::text,p.annual_price::text,p.trial_days,p.max_locations,p.max_users,p.module_keys,p.terms_version,p.active,
		       s.billing_cycle,s.price_amount::text,s.currency,s.status,
		       CASE WHEN s.trial_starts_at IS NULL THEN NULL ELSE to_char(s.trial_starts_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') END,
		       CASE WHEN s.trial_ends_at IS NULL THEN NULL ELSE to_char(s.trial_ends_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') END,
		       CASE WHEN s.current_period_starts_at IS NULL THEN NULL ELSE to_char(s.current_period_starts_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') END,
		       CASE WHEN s.current_period_ends_at IS NULL THEN NULL ELSE to_char(s.current_period_ends_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') END,
		       CASE WHEN s.renews_at IS NULL THEN NULL ELSE to_char(s.renews_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') END,
		       s.auto_renew,s.terms_version,
		       CASE WHEN s.terms_accepted_at IS NULL THEN NULL ELSE to_char(s.terms_accepted_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') END
		FROM organization_subscriptions s
		JOIN subscription_plans p ON p.id=s.plan_id
		WHERE s.organization_id=$1
	`, s.OrganizationID).Scan(
		&out.ID,
		&plan.ID,&plan.Code,&plan.Name,&plan.Description,&plan.Currency,&plan.MonthlyPrice,&plan.AnnualPrice,&plan.TrialDays,&plan.MaxLocations,&plan.MaxUsers,&plan.ModuleKeys,&plan.TermsVersion,&plan.Active,
		&out.BillingCycle,&out.PriceAmount,&out.Currency,&out.Status,
		&out.TrialStartsAt,&out.TrialEndsAt,&out.CurrentPeriodStartsAt,&out.CurrentPeriodEndsAt,&out.RenewsAt,
		&out.AutoRenew,&out.TermsVersion,&out.TermsAcceptedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "subscription_not_found", "La empresa todavía no tiene una suscripción registrada.")
		return
	}
	if err != nil {
		fail(w, 503, "subscription_unavailable", "No pudimos cargar la suscripción.")
		return
	}
	out.Plan = plan
	_ = a.db.QueryRow(r.Context(), `SELECT count(*) FROM locations WHERE organization_id=$1 AND active`, s.OrganizationID).Scan(&out.Usage.Locations)
	_ = a.db.QueryRow(r.Context(), `SELECT count(*) FROM users WHERE organization_id=$1 AND active AND NOT platform_admin`, s.OrganizationID).Scan(&out.Usage.Users)

	out.Payments = []subscriptionPaymentView{}
	rows, paymentErr := a.db.Query(r.Context(), `
		SELECT id,amount::text,currency,status,provider,external_reference,
		       CASE WHEN paid_at IS NULL THEN NULL ELSE to_char(paid_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') END,
		       to_char(created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM subscription_payments
		WHERE organization_id=$1
		ORDER BY created_at DESC
		LIMIT 10
	`, s.OrganizationID)
	if paymentErr == nil {
		defer rows.Close()
		for rows.Next() {
			var item subscriptionPaymentView
			if rows.Scan(&item.ID,&item.Amount,&item.Currency,&item.Status,&item.Provider,&item.ExternalReference,&item.PaidAt,&item.CreatedAt) == nil {
				out.Payments = append(out.Payments, item)
			}
		}
	}
	writeJSON(w, 200, out)
}

func ensureSubscriptionCapacity(ctx context.Context, tx pgx.Tx, organizationID, resource string) error {
	var max *int
	switch resource {
	case "locations":
		if err := tx.QueryRow(ctx, `
			SELECT p.max_locations
			FROM organization_subscriptions s JOIN subscription_plans p ON p.id=s.plan_id
			WHERE s.organization_id=$1
			FOR UPDATE OF s
		`, organizationID).Scan(&max); err != nil {
			return err
		}
		if max == nil {
			return nil
		}
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM locations WHERE organization_id=$1 AND active`, organizationID).Scan(&count); err != nil {
			return err
		}
		if count >= *max {
			return errSubscriptionLimit
		}
	case "users":
		if err := tx.QueryRow(ctx, `
			SELECT p.max_users
			FROM organization_subscriptions s JOIN subscription_plans p ON p.id=s.plan_id
			WHERE s.organization_id=$1
			FOR UPDATE OF s
		`, organizationID).Scan(&max); err != nil {
			return err
		}
		if max == nil {
			return nil
		}
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM users WHERE organization_id=$1 AND active AND NOT platform_admin`, organizationID).Scan(&count); err != nil {
			return err
		}
		if count >= *max {
			return errSubscriptionLimit
		}
	default:
		return errors.New("unknown subscription resource")
	}
	return nil
}

func (a *API) updateOrganizationSubscription(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in subscriptionUpdateInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_subscription", "Revisa los datos de la suscripción.")
		return
	}
	in.PlanID = strings.TrimSpace(in.PlanID)
	in.BillingCycle = strings.TrimSpace(in.BillingCycle)
	in.Status = strings.TrimSpace(in.Status)
	if in.PlanID == "" || (in.BillingCycle!="monthly" && in.BillingCycle!="annual") || (in.Status!="trial" && in.Status!="active" && in.Status!="past_due" && in.Status!="cancelled") {
		fail(w, 400, "invalid_subscription", "Plan, ciclo y estado son obligatorios.")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w,503,"subscription_unavailable","No pudimos actualizar la suscripción.")
		return
	}
	defer tx.Rollback(r.Context())

	plan, err := readSubscriptionPlan(r.Context(), tx, in.PlanID, true)
	if err != nil {
		fail(w,404,"plan_not_found","El plan seleccionado no está disponible.")
		return
	}
	if in.Status=="trial" && plan.TrialDays==0 {
		fail(w,409,"trial_unavailable","El plan seleccionado no incluye prueba gratuita.")
		return
	}
	var currentPlanID string
	var currentTerms *string
	if err = tx.QueryRow(r.Context(), `SELECT plan_id,terms_version FROM organization_subscriptions WHERE organization_id=$1 FOR UPDATE`, s.OrganizationID).Scan(&currentPlanID,&currentTerms); err != nil {
		fail(w,404,"subscription_not_found","La empresa todavía no tiene una suscripción.")
		return
	}
	var locationCount,userCount int
	_ = tx.QueryRow(r.Context(), `SELECT count(*) FROM locations WHERE organization_id=$1 AND active`, s.OrganizationID).Scan(&locationCount)
	_ = tx.QueryRow(r.Context(), `SELECT count(*) FROM users WHERE organization_id=$1 AND active AND NOT platform_admin`, s.OrganizationID).Scan(&userCount)
	if (plan.MaxLocations!=nil && locationCount>*plan.MaxLocations) || (plan.MaxUsers!=nil && userCount>*plan.MaxUsers) {
		fail(w,409,"plan_limit_conflict","La empresa supera los límites del plan seleccionado.")
		return
	}
	needsTerms := currentPlanID!=plan.ID || currentTerms==nil || *currentTerms!=plan.TermsVersion
	if needsTerms && !in.TermsAccepted {
		fail(w,409,"terms_required","La nueva versión de condiciones debe aceptarse antes de cambiar el plan.")
		return
	}

	now := time.Now().UTC()
	price := plan.MonthlyPrice
	if in.BillingCycle=="annual" { price=plan.AnnualPrice }
	var trialStarts,trialEnds,currentEnd,renews *time.Time
	if in.Status=="trial" {
		start:=now
		end:=now.AddDate(0,0,plan.TrialDays)
		trialStarts=&start;trialEnds=&end;currentEnd=&end;renews=&end
	} else if in.Status=="active" || in.Status=="past_due" {
		end:=subscriptionPeriodEnd(now,in.BillingCycle)
		currentEnd=&end;renews=&end
	}
	var cancelledAt *time.Time
	if in.Status=="cancelled" { cancelledAt=&now }
	termsVersion:=currentTerms
	var termsAcceptedAt any = nil
	var termsAcceptedBy any = nil
	if in.TermsAccepted {
		termsVersion=&plan.TermsVersion
		termsAcceptedAt=now
		termsAcceptedBy=s.UserID
	}
	_, err = tx.Exec(r.Context(), `
		UPDATE organization_subscriptions
		SET plan_id=$2,billing_cycle=$3,price_amount=$4,currency=$5,status=$6,
		    trial_starts_at=$7,trial_ends_at=$8,current_period_starts_at=$9,current_period_ends_at=$10,renews_at=$11,
		    auto_renew=$12,terms_version=COALESCE($13,terms_version),
		    terms_accepted_at=COALESCE($14,terms_accepted_at),terms_accepted_by=COALESCE($15,terms_accepted_by),
		    cancelled_at=$16,updated_at=now()
		WHERE organization_id=$1
	`, s.OrganizationID,plan.ID,in.BillingCycle,price,plan.Currency,in.Status,trialStarts,trialEnds,now,currentEnd,renews,in.AutoRenew,termsVersion,termsAcceptedAt,termsAcceptedBy,cancelledAt)
	if err != nil {
		fail(w,503,"subscription_unavailable","No pudimos actualizar la suscripción.")
		return
	}
	moduleKeys:=plan.ModuleKeys
	if in.Status=="cancelled" { moduleKeys=[]string{} }
	if err=syncOrganizationModulesForPlan(r.Context(),tx,s.OrganizationID,moduleKeys);err!=nil {
		fail(w,503,"subscription_unavailable","No pudimos aplicar los módulos del plan.")
		return
	}
	if err=tx.Commit(r.Context());err!=nil {
		fail(w,503,"subscription_unavailable","No pudimos confirmar el cambio de suscripción.")
		return
	}
	a.audit(r,"subscription.updated","organization",s.OrganizationID)
	a.getOrganizationSubscription(w,r)
}

func (a *API) recordSubscriptionPayment(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in subscriptionPaymentInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w,400,"invalid_payment","Revisa los datos del pago.")
		return
	}
	in.Amount=strings.TrimSpace(in.Amount)
	in.Currency=strings.ToUpper(strings.TrimSpace(in.Currency))
	in.Status=strings.TrimSpace(in.Status)
	in.Provider=strings.TrimSpace(in.Provider)
	in.ExternalReference=strings.TrimSpace(in.ExternalReference)
	amount,amountOK:=new(big.Rat).SetString(in.Amount)
	if !amountOK || amount.Sign()<0 || (in.Status!="pending"&&in.Status!="paid"&&in.Status!="failed"&&in.Status!="refunded") || in.Provider=="" {
		fail(w,400,"invalid_payment","Monto, estado y proveedor son obligatorios.")
		return
	}
	tx,err:=a.db.Begin(r.Context())
	if err!=nil { fail(w,503,"payment_unavailable","No pudimos registrar el pago.");return }
	defer tx.Rollback(r.Context())
	var subscriptionID,billingCycle,currency string
	var moduleKeys []string
	if err=tx.QueryRow(r.Context(),`SELECT s.id,s.billing_cycle,s.currency,p.module_keys FROM organization_subscriptions s JOIN subscription_plans p ON p.id=s.plan_id WHERE s.organization_id=$1 FOR UPDATE OF s`,s.OrganizationID).Scan(&subscriptionID,&billingCycle,&currency,&moduleKeys);err!=nil {
		fail(w,404,"subscription_not_found","La empresa todavía no tiene una suscripción.");return
	}
	if in.Currency=="" { in.Currency=currency }
	if in.Currency!=currency {
		fail(w,409,"payment_currency_mismatch","El pago debe registrarse en la moneda de la suscripción.");return
	}
	var paidAt *time.Time
	if in.PaidAt!="" {
		parsed,parseErr:=time.Parse(time.RFC3339,in.PaidAt)
		if parseErr!=nil { fail(w,400,"invalid_payment","La fecha de pago no es válida.");return }
		paidAt=&parsed
	} else if in.Status=="paid" {
		now:=time.Now().UTC();paidAt=&now
	}
	now:=time.Now().UTC()
	periodStart:=now
	periodEnd:=subscriptionPeriodEnd(periodStart,billingCycle)
	var external any=nil
	if in.ExternalReference!="" { external=in.ExternalReference }
	var id string
	err=tx.QueryRow(r.Context(),`
		INSERT INTO subscription_payments(organization_id,subscription_id,amount,currency,status,provider,external_reference,period_starts_at,period_ends_at,paid_at,recorded_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id
	`,s.OrganizationID,subscriptionID,in.Amount,in.Currency,in.Status,in.Provider,external,periodStart,periodEnd,paidAt,s.UserID).Scan(&id)
	if err!=nil { fail(w,409,"payment_conflict","La referencia del pago ya fue registrada.");return }
	if in.Status=="paid" {
		_,err=tx.Exec(r.Context(),`
			UPDATE organization_subscriptions
			SET status='active',current_period_starts_at=$2,current_period_ends_at=$3,renews_at=$3,cancelled_at=NULL,updated_at=now()
			WHERE organization_id=$1
		`,s.OrganizationID,periodStart,periodEnd)
		if err==nil { err=syncOrganizationModulesForPlan(r.Context(),tx,s.OrganizationID,moduleKeys) }
	}
	if err!=nil || tx.Commit(r.Context())!=nil { fail(w,503,"payment_unavailable","No pudimos confirmar el pago.");return }
	a.audit(r,"subscription.payment_recorded","subscription_payment",id)
	writeJSON(w,201,map[string]string{"id":id})
}
