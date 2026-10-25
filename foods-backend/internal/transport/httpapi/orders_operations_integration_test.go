package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestOperationalOrdersListReturnsRealWhatsAppOrdersAndCurrency(t *testing.T) {
	pool:=integrationPool(t)
	s:=seedInventoryScope(t,pool)
	api:=New(pool)

	var orderID string
	if err:=pool.QueryRow(context.Background(),
		"INSERT INTO orders(organization_id,location_id,channel,customer_name,customer_phone,subtotal,delivery_fee,total,status,created_by) VALUES($1,$2,'whatsapp','Cliente Concierge','51999999999',37.50,0,37.50,'confirmado',NULL) RETURNING id",
		s.OrganizationID,s.LocationID,
	).Scan(&orderID);err!=nil{
		t.Fatal(err)
	}

	req:=httptest.NewRequest("GET","/v1/operations/orders?status=abiertos&channel=salon&page=1&pageSize=20",nil)
	req=req.WithContext(context.WithValue(req.Context(),scopeKey{},s))
	rec:=httptest.NewRecorder()
	api.listOrders(rec,req)
	if rec.Code!=200{
		t.Fatalf("list operational orders: %d %s",rec.Code,rec.Body.String())
	}

	var out struct{
		Items []order `json:"items"`
		Total int `json:"total"`
		ChannelCounts map[string]int `json:"channelCounts"`
		ChannelOptions []map[string]string `json:"channelOptions"`
		CurrencySymbol string `json:"currencySymbol"`
	}
	if err:=json.Unmarshal(rec.Body.Bytes(),&out);err!=nil{
		t.Fatal(err)
	}
	found:=false
	for _,item:=range out.Items{
		if item.ID==orderID{
			found=true
			if item.Channel!="whatsapp"||item.Status!="confirmado"||item.Total!="37.50"{
				t.Fatalf("unexpected WhatsApp order: %#v",item)
			}
		}
	}
	if !found{
		t.Fatalf("real WhatsApp order %s not returned: %#v",orderID,out.Items)
	}
	// El pedido QR por WhatsApp conserva su canal, pero se filtra y cuenta como Salón.
	if out.Total<1||out.ChannelCounts["salon"]<1||out.ChannelCounts["whatsapp"]!=0{
		t.Fatalf("unexpected list counts: total=%d channels=%#v",out.Total,out.ChannelCounts)
	}
	for _,option:=range out.ChannelOptions{
		if option["value"]=="whatsapp"{
			t.Fatalf("WhatsApp is an origin, not a service tab: %#v",out.ChannelOptions)
		}
	}
	if out.CurrencySymbol==""{
		t.Fatal("currency symbol must come from the location fiscal profile")
	}
}
