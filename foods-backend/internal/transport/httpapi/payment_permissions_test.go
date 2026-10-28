package httpapi

import (
	"context"
	"testing"
)

// Direct-handler tests still need an explicit actor with cashier authority.
func grantCashManagementForPaymentTest(t *testing.T, api *integrationAPI, s scope) {
	t.Helper()
	_, err := api.db.Exec(context.Background(), `WITH role AS (
	 INSERT INTO roles(organization_id,name,permissions,menu_access)
	 VALUES($1,'Cajero de prueba',ARRAY['cash.read','cash.manage'],ARRAY['caja','pos']) RETURNING id
	) INSERT INTO user_roles(user_id,role_id,location_id) SELECT $2,id,$3 FROM role`, s.OrganizationID, s.UserID, s.LocationID)
	if err != nil {
		t.Fatal(err)
	}
}
