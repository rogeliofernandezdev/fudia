package httpapi

import (
	"testing"
	"time"
)

func TestSubscriptionPeriodEndClampsToTargetMonthAndKeepsAnchor(t *testing.T) {
	tests := []struct {
		name      string
		start     string
		cycle     string
		anchorDay int
		want      string
	}{
		{"Jan 28 to Feb", "2026-01-28T10:15:30Z", "monthly", 28, "2026-02-28T10:15:30Z"},
		{"Jan 29 to Feb", "2026-01-29T10:15:30Z", "monthly", 29, "2026-02-28T10:15:30Z"},
		{"Jan 30 to Feb", "2026-01-30T10:15:30Z", "monthly", 30, "2026-02-28T10:15:30Z"},
		{"Jan 31 to Feb", "2026-01-31T10:15:30Z", "monthly", 31, "2026-02-28T10:15:30Z"},
		{"leap Feb 29 to Mar", "2028-02-29T10:15:30Z", "monthly", 29, "2028-03-29T10:15:30Z"},
		{"leap Feb 28 to Mar", "2028-02-28T10:15:30Z", "monthly", 28, "2028-03-28T10:15:30Z"},
		{"nonleap Feb 28 to Mar anchored 29", "2027-02-28T10:15:30Z", "monthly", 29, "2027-03-29T10:15:30Z"},
		{"March 28 to April", "2026-03-28T10:15:30Z", "monthly", 28, "2026-04-28T10:15:30Z"},
		{"March 29 to April", "2026-03-29T10:15:30Z", "monthly", 29, "2026-04-29T10:15:30Z"},
		{"March 30 to April", "2026-03-30T10:15:30Z", "monthly", 30, "2026-04-30T10:15:30Z"},
		{"March 31 to April", "2026-03-31T10:15:30Z", "monthly", 31, "2026-04-30T10:15:30Z"},
		{"December 28 to January", "2026-12-28T10:15:30Z", "monthly", 28, "2027-01-28T10:15:30Z"},
		{"December 29 to January", "2026-12-29T10:15:30Z", "monthly", 29, "2027-01-29T10:15:30Z"},
		{"December 30 to January", "2026-12-30T10:15:30Z", "monthly", 30, "2027-01-30T10:15:30Z"},
		{"December 31 to January", "2026-12-31T10:15:30Z", "monthly", 31, "2027-01-31T10:15:30Z"},
		{"leap annual Feb 29", "2028-02-29T10:15:30Z", "annual", 29, "2029-02-28T10:15:30Z"},
		{"annual Feb 29 through leap year", "2029-02-28T10:15:30Z", "annual", 29, "2030-02-29T10:15:30Z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, err := time.Parse(time.RFC3339, tt.start)
			if err != nil { t.Fatal(err) }
			got := subscriptionPeriodEnd(start, tt.cycle, tt.anchorDay).Format(time.RFC3339)
			if got != tt.want { t.Fatalf("subscriptionPeriodEnd() = %s, want %s", got, tt.want) }
		})
	}
}

func TestSubscriptionMonthlyRenewalsKeepContractAnchor(t *testing.T) {
	start, _ := time.Parse(time.RFC3339, "2026-01-30T00:00:00Z")
	anchorDay := start.Day()
	want := []string{"2026-02-28", "2026-03-30", "2026-04-30", "2026-05-30"}
	for i, expected := range want {
		end := subscriptionPeriodEnd(start, "monthly", anchorDay)
		if got := end.Format("2006-01-02"); got != expected {
			t.Fatalf("renewal %d end = %s, want %s", i+1, got, expected)
		}
		if !end.After(start) { t.Fatalf("renewal %d does not advance: start=%s end=%s", i+1, start, end) }
		start = end
	}
}

func TestSubscriptionMonthEndRenewalsDoNotOverlapOrSkip(t *testing.T) {
	start, _ := time.Parse(time.RFC3339, "2026-01-31T00:00:00Z")
	anchorDay := start.Day()
	previousEnd := start
	for i := 0; i < 48; i++ {
		end := subscriptionPeriodEnd(start, "monthly", anchorDay)
		if !end.After(start) { t.Fatalf("period %d does not advance", i+1) }
		if !start.Equal(previousEnd) { t.Fatalf("period %d has a gap or overlap: start=%s previous end=%s", i+1, start, previousEnd) }
		if end.Day() != 31 && end.Day() != 30 && end.Day() != 29 && end.Day() != 28 { t.Fatalf("unexpected end day: %s", end) }
		previousEnd = end
		start = end
	}
}
