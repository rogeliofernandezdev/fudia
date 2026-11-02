package domain

import "testing"

func TestCounterOriginAndTableRequirement(t *testing.T) {
	if Origin(ChannelCounter) != OriginCounter || Origin("recojo") != OriginStandard || Origin("whatsapp") != OriginStandard {
		t.Fatal("only the counter channel has the counter origin")
	}
	if ServiceChannel(ChannelCounter) != "recojo" {
		t.Fatal("a counter sale is served as pickup in Pedidos and Cocina")
	}
	if ManualChannels[0]["value"] != ChannelCounter || ManualChannels[0]["label"] != "Mostrador" {
		t.Fatalf("manual counter option: %v", ManualChannels)
	}
	cases := []struct{ channel, table, kind, code string }{
		{"salon", "", "", "table_required"},
		{"salon", "t1", "table", ""},
		{"salon", "t1", "counter", "counter_not_table"},
		{ChannelCounter, "", "", ""},
		{ChannelCounter, "t1", "", "table_not_allowed"},
		{"recojo", "t1", "", "table_not_allowed"},
		{"delivery", "", "", ""},
	}
	for _, tc := range cases {
		if code, _ := TableRequirement(tc.channel, tc.table, tc.kind); code != tc.code {
			t.Fatalf("%+v: got %q", tc, code)
		}
	}
}
