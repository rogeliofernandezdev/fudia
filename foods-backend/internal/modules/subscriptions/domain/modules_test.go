package domain

import "testing"

func TestModuleEnabledByAvailability(t *testing.T) {
	on := ModuleSwitch{On: true, InPlan: true}
	outsidePlan := ModuleSwitch{On: true}
	cases := []struct {
		key   string
		state ModuleSwitch
		want  bool
	}{
		{"reportes", on, true},
		{"reportes", outsidePlan, false},
		{"reportes", ModuleSwitch{InPlan: true}, false},
		// A pilot does not depend on the plan: plans cannot include it.
		{"facturacion", outsidePlan, true},
		{"facturacion", ModuleSwitch{InPlan: true}, false},
		// Planned modules stay closed even with stale data.
		{"crm", on, false},
	}
	for _, c := range cases {
		if got := ModuleEnabled(c.key, c.state); got != c.want {
			t.Errorf("%s %+v: got %v", c.key, c.state, got)
		}
	}
	if ModuleCanPilot("reportes") || !ModuleCanPilot("facturacion") || ModuleCanPilot("crm") {
		t.Fatal("only development modules can be piloted")
	}
}
