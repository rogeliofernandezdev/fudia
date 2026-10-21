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

	// IDs from another organization cannot be attached to this product.
	otherScope := seedInventoryScope(t, pool)
	var otherProductID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO products(
		  organization_id,sku,name,description,price,active,product_type,quantity_control
		) VALUES($1,'MOD-OTHER-ORG','Producto aislado','',10,true,'prepared','none')
		RETURNING id
	`, otherScope.OrganizationID).Scan(&otherProductID); err != nil {
		t.Fatal(err)
	}
	otherReq := httptest.NewRequest("PUT", "/v1/admin/products/"+otherProductID+"/modifiers", bytes.NewReader([]byte(`{
	  "groups":[{"name":"Otro grupo","required":false,"minSelections":0,"maxSelections":1,
	    "options":[{"name":"Otra opción","surcharge":"0"}]}]
	}`)))
	otherReq.SetPathValue("id", otherProductID)
	otherReq = otherReq.WithContext(context.WithValue(otherReq.Context(), scopeKey{}, otherScope))
	otherRec := httptest.NewRecorder()
	api.saveProductModifiers(otherRec, otherReq)
	if otherRec.Code != 200 {
		t.Fatalf("save second organization's modifiers: %d %s", otherRec.Code, otherRec.Body.String())
	}
	var otherConfig modifierConfigView
	if err := json.Unmarshal(otherRec.Body.Bytes(), &otherConfig); err != nil {
		t.Fatal(err)
	}
	foreignGroup := saved
	foreignGroup.Groups = append([]modifierGroupView(nil), saved.Groups...)
	foreignGroup.Groups[0].ID = otherConfig.Groups[0].ID
	foreignBody, err := json.Marshal(foreignGroup)
	if err != nil {
		t.Fatal(err)
	}
	foreignReq := httptest.NewRequest("PUT", "/v1/admin/products/"+productID+"/modifiers", bytes.NewReader(foreignBody))
	foreignReq.SetPathValue("id", productID)
	foreignReq = foreignReq.WithContext(context.WithValue(foreignReq.Context(), scopeKey{}, s))
	foreignRec := httptest.NewRecorder()
	api.saveProductModifiers(foreignRec, foreignReq)
	if foreignRec.Code != 400 {
		t.Fatalf("foreign organization group ID should be rejected: %d %s", foreignRec.Code, foreignRec.Body.String())
	}
	foreignOption := saved
	foreignOption.Groups = append([]modifierGroupView(nil), saved.Groups...)
	foreignOption.Groups[0].Options = append([]modifierOptionView(nil), saved.Groups[0].Options...)
	foreignOption.Groups[0].Options[0].ID = otherConfig.Groups[0].Options[0].ID
	foreignOptionBody, err := json.Marshal(foreignOption)
	if err != nil {
		t.Fatal(err)
	}
	foreignOptionReq := httptest.NewRequest("PUT", "/v1/admin/products/"+productID+"/modifiers", bytes.NewReader(foreignOptionBody))
	foreignOptionReq.SetPathValue("id", productID)
	foreignOptionReq = foreignOptionReq.WithContext(context.WithValue(foreignOptionReq.Context(), scopeKey{}, s))
	foreignOptionRec := httptest.NewRecorder()
	api.saveProductModifiers(foreignOptionRec, foreignOptionReq)
	if foreignOptionRec.Code != 400 {
		t.Fatalf("foreign organization option ID should be rejected: %d %s", foreignOptionRec.Code, foreignOptionRec.Body.String())
	}

	// A second identical save must preserve every group and option identifier.
	identicalBody, err := json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}
	identicalReq := httptest.NewRequest("PUT", "/v1/admin/products/"+productID+"/modifiers", bytes.NewReader(identicalBody))
	identicalReq.SetPathValue("id", productID)
	identicalReq = identicalReq.WithContext(context.WithValue(identicalReq.Context(), scopeKey{}, s))
	identicalRec := httptest.NewRecorder()
	api.saveProductModifiers(identicalRec, identicalReq)
	if identicalRec.Code != 200 {
		t.Fatalf("save identical modifiers: %d %s", identicalRec.Code, identicalRec.Body.String())
	}
	var identical modifierConfigView
	if err := json.Unmarshal(identicalRec.Body.Bytes(), &identical); err != nil {
		t.Fatal(err)
	}
	if identical.Groups[0].ID != saved.Groups[0].ID ||
		identical.Groups[0].Options[0].ID != saved.Groups[0].Options[0].ID ||
		identical.Groups[1].ID != saved.Groups[1].ID ||
		identical.Groups[1].Options[0].ID != saved.Groups[1].Options[0].ID {
		t.Fatalf("identical save changed modifier IDs: before=%#v after=%#v", saved, identical)
	}

	// Rename, reprice, reorder, add, and remove options by carrying their IDs.
	edited := identical
	// Keep the original group option slices intact for the ID comparisons below.
	edited.Groups = append([]modifierGroupView(nil), identical.Groups...)
	edited.Groups[1].Options = []modifierOptionView{
		{ID: identical.Groups[1].Options[1].ID, Name: "Tocino", Surcharge: "4.50"},
		{ID: identical.Groups[1].Options[0].ID, Name: "Queso premium", Surcharge: "3.75"},
		{Name: "Palta", Surcharge: "2.00"},
	}
	editedBody, err := json.Marshal(edited)
	if err != nil {
		t.Fatal(err)
	}
	editedReq := httptest.NewRequest("PUT", "/v1/admin/products/"+productID+"/modifiers", bytes.NewReader(editedBody))
	editedReq.SetPathValue("id", productID)
	editedReq = editedReq.WithContext(context.WithValue(editedReq.Context(), scopeKey{}, s))
	editedRec := httptest.NewRecorder()
	api.saveProductModifiers(editedRec, editedReq)
	if editedRec.Code != 200 {
		t.Fatalf("save partially edited modifiers: %d %s", editedRec.Code, editedRec.Body.String())
	}
	var partiallyEdited modifierConfigView
	if err := json.Unmarshal(editedRec.Body.Bytes(), &partiallyEdited); err != nil {
		t.Fatal(err)
	}
	if partiallyEdited.Groups[1].ID != identical.Groups[1].ID ||
		partiallyEdited.Groups[1].Options[0].ID != identical.Groups[1].Options[1].ID ||
		partiallyEdited.Groups[1].Options[1].ID != identical.Groups[1].Options[0].ID ||
		partiallyEdited.Groups[1].Options[1].Name != "Queso premium" ||
		partiallyEdited.Groups[1].Options[1].Surcharge != "3.75" ||
		partiallyEdited.Groups[1].Options[2].ID == "" ||
		partiallyEdited.Groups[1].Options[2].ID == identical.Groups[1].Options[0].ID ||
		len(partiallyEdited.Groups[1].Options) != 3 {
		t.Fatalf("partial edit did not preserve/add the expected modifier identities: %#v", partiallyEdited)
	}

	// Remove one option and reorder the remaining ones without changing IDs.
	partiallyEdited.Groups[1].Options = []modifierOptionView{
		partiallyEdited.Groups[1].Options[2],
		partiallyEdited.Groups[1].Options[1],
	}
	removalBody, err := json.Marshal(partiallyEdited)
	if err != nil {
		t.Fatal(err)
	}
	removalReq := httptest.NewRequest("PUT", "/v1/admin/products/"+productID+"/modifiers", bytes.NewReader(removalBody))
	removalReq.SetPathValue("id", productID)
	removalReq = removalReq.WithContext(context.WithValue(removalReq.Context(), scopeKey{}, s))
	removalRec := httptest.NewRecorder()
	api.saveProductModifiers(removalRec, removalReq)
	if removalRec.Code != 200 {
		t.Fatalf("remove and reorder modifier options: %d %s", removalRec.Code, removalRec.Body.String())
	}
	var reordered modifierConfigView
	if err := json.Unmarshal(removalRec.Body.Bytes(), &reordered); err != nil {
		t.Fatal(err)
	}
	if len(reordered.Groups[1].Options) != 2 ||
		reordered.Groups[1].Options[0].ID != partiallyEdited.Groups[1].Options[0].ID ||
		reordered.Groups[1].Options[1].ID != partiallyEdited.Groups[1].Options[1].ID ||
		reordered.Groups[1].Options[0].Name != "Palta" ||
		reordered.Groups[1].Options[1].Name != "Queso premium" {
		t.Fatalf("removal/reorder changed retained option identities: %#v", reordered.Groups[1].Options)
	}

	// The retained IDs from the open form remain valid for a new order.
	orderGroupID := reordered.Groups[1].ID
	orderOptionID := reordered.Groups[1].Options[1].ID
	orderReq := httptest.NewRequest("POST", "/v1/admin/orders", bytes.NewReader([]byte(`{
	  "channel":"mostrador",
	  "items":[{"productId":"`+productID+`","qty":1,"unitPrice":0,"selections":[],
	    "modifiers":[
	      {"groupId":"`+reordered.Groups[0].ID+`","optionId":"`+reordered.Groups[0].Options[0].ID+`"},
	      {"groupId":"`+orderGroupID+`","optionId":"`+orderOptionID+`"}]}]
	}`)))
	orderReq = orderReq.WithContext(context.WithValue(orderReq.Context(), scopeKey{}, s))
	orderRec := httptest.NewRecorder()
	api.createOrder(orderRec, orderReq)
	if orderRec.Code != 201 {
		t.Fatalf("create order using retained modifier IDs: %d %s", orderRec.Code, orderRec.Body.String())
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


func TestOrderModifiersAreValidatedPricedAndSnapshotted(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()

	var productID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO products(
		  organization_id,sku,name,description,price,active,product_type,quantity_control
		)
		VALUES($1,'MOD-ORDER','Pizza configurable','',20,true,'prepared','none')
		RETURNING id
	`, s.OrganizationID).Scan(&productID); err != nil {
		t.Fatal(err)
	}

	configReq := httptest.NewRequest("PUT", "/v1/admin/products/"+productID+"/modifiers", bytes.NewReader([]byte(`{
	  "groups":[{
	    "name":"Tamaño","required":true,"minSelections":1,"maxSelections":1,
	    "options":[{"name":"Grande","surcharge":"5.00"},{"name":"Mediana","surcharge":"0"}]
	  }]
	}`)))
	configReq.SetPathValue("id", productID)
	configReq = configReq.WithContext(context.WithValue(configReq.Context(), scopeKey{}, s))
	configRec := httptest.NewRecorder()
	api.saveProductModifiers(configRec, configReq)
	if configRec.Code != 200 {
		t.Fatalf("save modifiers: %d %s", configRec.Code, configRec.Body.String())
	}
	var config modifierConfigView
	if err := json.Unmarshal(configRec.Body.Bytes(), &config); err != nil {
		t.Fatal(err)
	}
	groupID := config.Groups[0].ID
	optionID := config.Groups[0].Options[0].ID

	missingReq := httptest.NewRequest("POST", "/v1/admin/orders", bytes.NewReader([]byte(`{
	  "channel":"mostrador",
	  "items":[{"productId":"`+productID+`","qty":1,"unitPrice":0,"selections":[],"modifiers":[]}]
	}`)))
	missingReq = missingReq.WithContext(context.WithValue(missingReq.Context(), scopeKey{}, s))
	missingRec := httptest.NewRecorder()
	api.createOrder(missingRec, missingReq)
	if missingRec.Code != 400 {
		t.Fatalf("required modifier must be enforced: %d %s", missingRec.Code, missingRec.Body.String())
	}

	orderReq := httptest.NewRequest("POST", "/v1/admin/orders", bytes.NewReader([]byte(`{
	  "channel":"mostrador",
	  "items":[{
	    "productId":"`+productID+`","qty":2,"unitPrice":0.01,"selections":[],
	    "modifiers":[{"groupId":"`+groupID+`","optionId":"`+optionID+`"}]
	  }]
	}`)))
	orderReq = orderReq.WithContext(context.WithValue(orderReq.Context(), scopeKey{}, s))
	orderRec := httptest.NewRecorder()
	api.createOrder(orderRec, orderReq)
	if orderRec.Code != 201 {
		t.Fatalf("create modified order: %d %s", orderRec.Code, orderRec.Body.String())
	}
	var created order
	if err := json.Unmarshal(orderRec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Total != "50.00" {
		t.Fatalf("backend must price base + modifier surcharge for every unit, got %s", created.Total)
	}

	// Updating the live name/price must not rewrite the historical order snapshot.
	config.Groups[0].Options[0].Name = "Grande actualizado"
	config.Groups[0].Options[0].Surcharge = "7.00"
	historyBody, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	historyReq := httptest.NewRequest("PUT", "/v1/admin/products/"+productID+"/modifiers", bytes.NewReader(historyBody))
	historyReq.SetPathValue("id", productID)
	historyReq = historyReq.WithContext(context.WithValue(historyReq.Context(), scopeKey{}, s))
	historyRec := httptest.NewRecorder()
	api.saveProductModifiers(historyRec, historyReq)
	if historyRec.Code != 200 {
		t.Fatalf("update live modifier after order: %d %s", historyRec.Code, historyRec.Body.String())
	}

	getReq := httptest.NewRequest("GET", "/v1/admin/orders/"+created.ID, nil)
	getReq.SetPathValue("id", created.ID)
	getReq = getReq.WithContext(context.WithValue(getReq.Context(), scopeKey{}, s))
	getRec := httptest.NewRecorder()
	api.getOrder(getRec, getReq)
	if getRec.Code != 200 {
		t.Fatalf("get modified order: %d %s", getRec.Code, getRec.Body.String())
	}
	var loaded order
	if err := json.Unmarshal(getRec.Body.Bytes(), &loaded); err != nil {
		t.Fatal(err)
	}
	if len(loaded.Items) != 1 || loaded.Items[0].UnitPrice != "25.00" ||
		len(loaded.Items[0].Modifiers) != 1 ||
		loaded.Items[0].Modifiers[0].Name != "Grande" ||
		loaded.Items[0].Modifiers[0].Surcharge != "5.00" {
		t.Fatalf("modifier snapshot was not preserved: %#v", loaded.Items)
	}
}
