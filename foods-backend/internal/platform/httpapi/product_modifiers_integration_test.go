package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestProductModifierConfigurationLifecycle(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()

	var productID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO products(
		  organization_id,sku,name,description,price,active,product_type,quantity_control
		)
		VALUES($1,'MOD-TEST','Hamburguesa configurable','',25,true,'prepared','none')
		RETURNING id
	`, s.OrganizationID).Scan(&productID); err != nil {
		t.Fatal(err)
	}

	body := []byte(`{
	  "groups":[
	    {
	      "name":"Término",
	      "required":true,
	      "minSelections":1,
	      "maxSelections":1,
	      "options":[
	        {"name":"Medio","surcharge":"0"},
	        {"name":"Bien cocido","surcharge":"0"}
	      ]
	    },
	    {
	      "name":"Extras",
	      "required":false,
	      "minSelections":0,
	      "maxSelections":2,
	      "options":[
	        {"name":"Queso extra","surcharge":"3.00"},
	        {"name":"Tocino","surcharge":"4.50"}
	      ]
	    }
	  ]
	}`)
	req := httptest.NewRequest("PUT", "/v1/admin/products/"+productID+"/modifiers", bytes.NewReader(body))
	req.SetPathValue("id", productID)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.saveProductModifiers(rec, req)
	if rec.Code != 200 {
		t.Fatalf("save modifiers: %d %s", rec.Code, rec.Body.String())
	}

	var saved modifierConfigView
	if err := json.Unmarshal(rec.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.ProductID != productID || len(saved.Groups) != 2 ||
		len(saved.Groups[0].Options) != 2 || saved.Groups[1].Options[0].Surcharge != "3.00" {
		t.Fatalf("unexpected saved modifier config: %#v", saved)
	}

	getReq := httptest.NewRequest("GET", "/v1/admin/products/"+productID+"/modifiers", nil)
	getReq.SetPathValue("id", productID)
	getReq = getReq.WithContext(context.WithValue(getReq.Context(), scopeKey{}, s))
	getRec := httptest.NewRecorder()
	api.getProductModifiers(getRec, getReq)
	if getRec.Code != 200 {
		t.Fatalf("get modifiers: %d %s", getRec.Code, getRec.Body.String())
	}

	replaceBody := []byte(`{
	  "groups":[
	    {
	      "name":"Salsa",
	      "required":false,
	      "minSelections":0,
	      "maxSelections":1,
	      "options":[{"name":"Picante","surcharge":"1.50"}]
	    }
	  ]
	}`)
	replaceReq := httptest.NewRequest("PUT", "/v1/admin/products/"+productID+"/modifiers", bytes.NewReader(replaceBody))
	replaceReq.SetPathValue("id", productID)
	replaceReq = replaceReq.WithContext(context.WithValue(replaceReq.Context(), scopeKey{}, s))
	replaceRec := httptest.NewRecorder()
	api.saveProductModifiers(replaceRec, replaceReq)
	if replaceRec.Code != 200 {
		t.Fatalf("replace modifiers: %d %s", replaceRec.Code, replaceRec.Body.String())
	}
	var replaced modifierConfigView
	if err := json.Unmarshal(replaceRec.Body.Bytes(), &replaced); err != nil {
		t.Fatal(err)
	}
	if len(replaced.Groups) != 1 || replaced.Groups[0].Name != "Salsa" ||
		len(replaced.Groups[0].Options) != 1 {
		t.Fatalf("modifier replacement did not replace previous groups: %#v", replaced)
	}

	invalidReq := httptest.NewRequest(
		"PUT",
		"/v1/admin/products/"+productID+"/modifiers",
		bytes.NewReader([]byte(`{
		  "groups":[{
		    "name":"Obligatorio",
		    "required":true,
		    "minSelections":0,
		    "maxSelections":1,
		    "options":[{"name":"Uno","surcharge":"0"}]
		  }]
		}`)),
	)
	invalidReq.SetPathValue("id", productID)
	invalidReq = invalidReq.WithContext(context.WithValue(invalidReq.Context(), scopeKey{}, s))
	invalidRec := httptest.NewRecorder()
	api.saveProductModifiers(invalidRec, invalidReq)
	if invalidRec.Code != 400 {
		t.Fatalf("required modifier with min 0 must be rejected: %d %s", invalidRec.Code, invalidRec.Body.String())
	}
}
