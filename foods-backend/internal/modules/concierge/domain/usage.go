package domain

// Monthly Concierge conversation limit. Caps, mode, overrides and usage are
// platform-administrator data: none of these types is returned to companies.

const (
	UsageModeMeasure = "measure" // only counts
	UsageModeWarn    = "warn"    // counts and warns the platform administrator
	UsageModeEnforce = "enforce" // also rejects new conversations at 100 %

	LimitSourcePlan      = "plan"
	LimitSourceCustom    = "custom"
	LimitSourceUnlimited = "unlimited"

	UsageAlertNone    = "none"
	UsageAlertWarning = "warning"
	UsageAlertReached = "reached"

	ConversationKindTable    = "table"
	ConversationKindDelivery = "delivery"

	TierLite  = "lite"
	TierFlash = "flash"
)

type UsageSettings struct {
	Mode        string `json:"mode"`
	WarnPercent int    `json:"warnPercent"`
	UpdatedAt   string `json:"updatedAt"`
}
type UsageSettingsInput struct {
	Mode        string `json:"mode"`
	WarnPercent int    `json:"warnPercent"`
}

func ValidUsageMode(mode string) bool {
	return mode == UsageModeMeasure || mode == UsageModeWarn || mode == UsageModeEnforce
}

type ConversationInput struct {
	ConversationID string `json:"conversationId"`
	Kind           string `json:"kind"`
}
type ConversationView struct {
	ConversationID string `json:"conversationId"`
	Kind           string `json:"kind"`
	Month          string `json:"month"`
}
type TierTokens struct {
	Tier         string `json:"tier"`
	InputTokens  int64  `json:"inputTokens"`
	OutputTokens int64  `json:"outputTokens"`
}
type UsageReport struct {
	RequestID string       `json:"requestId"`
	Tiers     []TierTokens `json:"tiers"`
}
type LimitOverrideInput struct {
	LimitSource      string `json:"limitSource"`
	MaxConversations *int   `json:"maxConversations"`
}

// Limit is the effective cap of a company: its own exception when it has one,
// otherwise its plan's. Max nil is unlimited.
type Limit struct {
	Source string
	Max    *int
}

func EffectiveLimit(planMax *int, hasOverride bool, overrideMax *int) Limit {
	if !hasOverride {
		return Limit{Source: LimitSourcePlan, Max: planMax}
	}
	if overrideMax == nil {
		return Limit{Source: LimitSourceUnlimited}
	}
	return Limit{Source: LimitSourceCustom, Max: overrideMax}
}

// Blocks reports whether a NEW conversation must be rejected. Conversations
// already open are never evaluated, so they are never cut.
func Blocks(mode string, limit Limit, conversations int) bool {
	return mode == UsageModeEnforce && limit.Max != nil && conversations >= *limit.Max
}

// Evaluate returns the usage percentage (nil when unlimited) and the alert the
// platform administrator sees. Measuring alone never raises an alert.
func Evaluate(settings UsageSettings, limit Limit, conversations int) (*int, string) {
	if limit.Max == nil || *limit.Max <= 0 {
		return nil, UsageAlertNone
	}
	percent := conversations * 100 / *limit.Max
	if settings.Mode == UsageModeMeasure {
		return &percent, UsageAlertNone
	}
	if conversations >= *limit.Max {
		return &percent, UsageAlertReached
	}
	if percent >= settings.WarnPercent {
		return &percent, UsageAlertWarning
	}
	return &percent, UsageAlertNone
}

type UsageSummary struct {
	OrganizationID        string  `json:"organizationId"`
	OrganizationName      string  `json:"organizationName"`
	Month                 string  `json:"month"`
	PlanName              *string `json:"planName"`
	MaxConversations      *int    `json:"maxConversations"`
	LimitSource           string  `json:"limitSource"`
	Conversations         int     `json:"conversations"`
	RejectedConversations int     `json:"rejectedConversations"`
	UsagePercent          *int    `json:"usagePercent"`
	Alert                 string  `json:"alert"`
}

// Apply fills the effective cap, percentage and alert.
func (s *UsageSummary) Apply(settings UsageSettings, limit Limit) {
	s.LimitSource = limit.Source
	s.MaxConversations = limit.Max
	s.UsagePercent, s.Alert = Evaluate(settings, limit, s.Conversations)
}

type UsageAlertCount struct {
	Warning int `json:"warning"`
	Reached int `json:"reached"`
}
type UsagePage struct {
	Settings UsageSettings   `json:"settings"`
	Summary  UsageAlertCount `json:"summary"`
	Items    []UsageSummary  `json:"items"`
}
type UsageMonth struct {
	Month                 string `json:"month"`
	Conversations         int    `json:"conversations"`
	RejectedConversations int    `json:"rejectedConversations"`
	InputTokens           int64  `json:"inputTokens"`
	OutputTokens          int64  `json:"outputTokens"`
}
type UsageDetail struct {
	UsageSummary
	PlanMaxConversations *int          `json:"planMaxConversations"`
	Settings             UsageSettings `json:"settings"`
	Tokens               []TierTokens  `json:"tokens"`
	History              []UsageMonth  `json:"history"`
}
