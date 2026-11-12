package domain

import "testing"

func TestOrderReadyForCustomerOnlyForPickupChannels(t *testing.T) {
	for channel, want := range map[string]string{"mostrador": FulfillmentCounter, "recojo": FulfillmentPickup, "delivery": "", "salon": "", "whatsapp": ""} {
		if got := FulfillmentFor(channel); got != want {
			t.Fatalf("FulfillmentFor(%q)=%q want %q", channel, got, want)
		}
		_, ok := OrderReadyForCustomer("o1", channel, "51911111111", "conv-1")
		if ok != (want != "") {
			t.Fatalf("channel %q notifies=%v", channel, ok)
		}
	}
	in, ok := OrderReadyForCustomer(" o1 ", "mostrador", " 51911111111 ", " conv-1 ")
	if !ok || in.Kind != CustomerOrderReady || in.OrderID != "o1" || in.CustomerPhone != "51911111111" || in.ConversationID != "conv-1" {
		t.Fatalf("trimmed input: %+v %v", in, ok)
	}
	for _, c := range []struct{ phone, conversation string }{{"5191", ""}, {"123456789012345678901234567890123", "c"}} {
		if _, ok := OrderReadyForCustomer("o1", "mostrador", c.phone, c.conversation); ok {
			t.Fatalf("invalid customer accepted: %+v", c)
		}
	}
}

func TestOrderOnTheWayForCustomerOnlyForDelivery(t *testing.T) {
	in, ok := OrderOnTheWayForCustomer(" o1 ", "delivery", " 51911111111 ", " conv-1 ")
	if !ok || in.Kind != CustomerOrderOnTheWay || in.OrderID != "o1" || in.CustomerPhone != "51911111111" || in.DedupeKey != "" {
		t.Fatalf("delivery notice: %+v %v", in, ok)
	}
	for _, channel := range []string{"recojo", "mostrador", "salon", "whatsapp"} {
		if _, ok := OrderOnTheWayForCustomer("o1", channel, "51911111111", "conv-1"); ok {
			t.Fatalf("channel %q must not notify", channel)
		}
	}
	if _, ok := OrderOnTheWayForCustomer("o1", "delivery", "51911111111", ""); ok {
		t.Fatal("order without conversation must not notify")
	}
	if ChannelFulfillment("delivery") != FulfillmentDelivery {
		t.Fatal("delivery fulfillment")
	}
}
