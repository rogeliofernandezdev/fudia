package domain

import "testing"

func intPtr(v int) *int { return &v }

func TestEffectiveLimitPrefersCompanyException(t *testing.T) {
	if got := EffectiveLimit(intPtr(500), false, nil); got.Source != LimitSourcePlan || *got.Max != 500 {
		t.Fatalf("plan cap: %+v", got)
	}
	if got := EffectiveLimit(nil, false, nil); got.Source != LimitSourcePlan || got.Max != nil {
		t.Fatalf("unlimited plan: %+v", got)
	}
	if got := EffectiveLimit(intPtr(500), true, intPtr(800)); got.Source != LimitSourceCustom || *got.Max != 800 {
		t.Fatalf("custom exception: %+v", got)
	}
	if got := EffectiveLimit(intPtr(500), true, nil); got.Source != LimitSourceUnlimited || got.Max != nil {
		t.Fatalf("unlimited exception: %+v", got)
	}
}

func TestBlocksOnlyNewConversationsWhenEnforcingAtCap(t *testing.T) {
	capped := Limit{Source: LimitSourcePlan, Max: intPtr(10)}
	cases := []struct {
		mode  string
		limit Limit
		count int
		want  bool
	}{
		{UsageModeMeasure, capped, 50, false},
		{UsageModeWarn, capped, 10, false},
		{UsageModeEnforce, capped, 9, false},
		{UsageModeEnforce, capped, 10, true},
		{UsageModeEnforce, capped, 11, true},
		{UsageModeEnforce, Limit{Source: LimitSourceUnlimited}, 1000, false},
	}
	for _, c := range cases {
		if got := Blocks(c.mode, c.limit, c.count); got != c.want {
			t.Fatalf("Blocks(%s,%v,%d)=%v", c.mode, c.limit.Max, c.count, got)
		}
	}
}

func TestEvaluateWarnsThePlatformOnlyOutsideMeasuring(t *testing.T) {
	capped := Limit{Source: LimitSourcePlan, Max: intPtr(500)}
	measure := UsageSettings{Mode: UsageModeMeasure, WarnPercent: 80}
	warn := UsageSettings{Mode: UsageModeWarn, WarnPercent: 80}
	enforce := UsageSettings{Mode: UsageModeEnforce, WarnPercent: 80}
	cases := []struct {
		settings UsageSettings
		count    int
		percent  int
		alert    string
	}{
		{measure, 600, 120, UsageAlertNone},
		{warn, 399, 79, UsageAlertNone},
		{warn, 400, 80, UsageAlertWarning},
		{warn, 500, 100, UsageAlertReached},
		{enforce, 450, 90, UsageAlertWarning},
		{enforce, 520, 104, UsageAlertReached},
	}
	for _, c := range cases {
		percent, alert := Evaluate(c.settings, capped, c.count)
		if percent == nil || *percent != c.percent || alert != c.alert {
			t.Fatalf("Evaluate(%s,%d)=%v,%s want %d,%s", c.settings.Mode, c.count, percent, alert, c.percent, c.alert)
		}
	}
	if percent, alert := Evaluate(enforce, Limit{Source: LimitSourceUnlimited}, 9999); percent != nil || alert != UsageAlertNone {
		t.Fatalf("unlimited has no percentage or alert: %v %s", percent, alert)
	}
}
