package httpapi

import "testing"

func TestCatalogPermissionLabelsMatchModule(t *testing.T) {
	want := map[string]string{"menu.read": "Ver carta y productos", "menu.manage": "Administrar carta y productos"}
	for _, group := range permissionCatalog {
		for _, item := range group["items"].([]map[string]string) {
			if label, ok := want[item["value"]]; ok {
				if item["label"] != label {
					t.Errorf("%s label = %q, want %q", item["value"], item["label"], label)
				}
				delete(want, item["value"])
			}
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing catalog permissions: %v", want)
	}
}
