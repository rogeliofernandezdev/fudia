package domain

import (
	"encoding/json"
	"testing"
)

// The batch SQL consumes this exact key; a changed tag can otherwise pass decoding tests.
func TestTableModelsKeepZoneJSONContract(t *testing.T) {
	for _, value := range []any{Table{Zone: "Terraza"}, TableInput{Zone: "Terraza"}, TableBatchItem{Zone: "Terraza"}, TableQR{Zone: "Terraza"}} {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]json.RawMessage
		if err = json.Unmarshal(data, &fields); err != nil {
			t.Fatal(err)
		}
		if string(fields["zone"]) != `"Terraza"` {
			t.Fatalf("%T lost zone: %s", value, data)
		}
		if _, exists := fields["Zone"]; exists {
			t.Fatalf("unexpected key: %s", data)
		}
	}
}
