package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type inventoryBalanceLock struct {
	ProductID *string
	Name string
	Unit string
	Quantity float64
	AverageUnitCost float64
}

func lockInventoryBalance(ctx context.Context, tx pgx.Tx, s scope, inventoryItemID string) (inventoryBalanceLock,error) {
	var out inventoryBalanceLock
	if _,err:=tx.Exec(ctx,`
		INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity,average_unit_cost)
		SELECT $1,$2,ii.id,0,0
		FROM inventory_items ii
		WHERE ii.id=$3 AND ii.organization_id=$1 AND ii.active
		ON CONFLICT(location_id,inventory_item_id) DO NOTHING
	`,s.OrganizationID,s.LocationID,inventoryItemID);err!=nil{return out,err}
	err:=tx.QueryRow(ctx,`
		SELECT ii.product_id,COALESCE(p.name,ii.name),ii.unit,sb.quantity::float8,sb.average_unit_cost::float8
		FROM inventory_items ii
		LEFT JOIN products p ON p.id=ii.product_id AND p.organization_id=ii.organization_id
		JOIN stock_balances sb ON sb.organization_id=ii.organization_id AND sb.inventory_item_id=ii.id
		  AND sb.location_id=$2
		WHERE ii.id=$3 AND ii.organization_id=$1 AND ii.active
		FOR UPDATE OF sb,ii
	`,s.OrganizationID,s.LocationID,inventoryItemID).Scan(&out.ProductID,&out.Name,&out.Unit,&out.Quantity,&out.AverageUnitCost)
	return out,err
}

func nullableUserID(id string) any {\n\tid=strings.TrimSpace(id)\n\tif id==""{return nil}\n\treturn id\n}\n\nfunc insertValuedMovement(ctx context.Context,tx pgx.Tx,s scope,productID *string,inventoryItemID,movementType string,delta,balanceAfter,unitCost float64,sourceType,sourceID,note string) error {
	valueDelta:=math.Round(delta*unitCost*10000)/10000
	balanceValue:=math.Round(balanceAfter*unitCost*10000)/10000
	_,err:=tx.Exec(ctx,`
		INSERT INTO stock_movements(
		  organization_id,location_id,product_id,inventory_item_id,movement_type,
		  quantity_delta,balance_after,source_type,source_id,note,created_by,
		  unit_cost,value_delta,balance_value_after
		)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
	`,s.OrganizationID,s.LocationID,productID,inventoryItemID,movementType,delta,balanceAfter,sourceType,sourceID,note,nullableUserID(s.UserID),unitCost,valueDelta,balanceValue)
	return err
}

func (a *API) updateInventoryLocationSettings(w http.ResponseWriter,r *http.Request){
	s:=r.Context().Value(scopeKey{}).(scope)
	var in struct{
		MinimumStock float64 `json:"minimumStock"`
		ReorderPoint float64 `json:"reorderPoint"`
		OptimalStock float64 `json:"optimalStock"`
	}
	if json.NewDecoder(r.Body).Decode(&in)!=nil||in.MinimumStock<0||in.ReorderPoint<0||in.OptimalStock<0{
		fail(w,400,"invalid_inventory_settings","Revisa los niveles de stock.")
		return
	}
	if in.ReorderPoint<in.MinimumStock{in.ReorderPoint=in.MinimumStock}
	if in.OptimalStock<in.ReorderPoint{in.OptimalStock=in.ReorderPoint}
	var exists bool
	if err:=a.db.QueryRow(r.Context(),`SELECT EXISTS(SELECT 1 FROM inventory_items WHERE id=$1 AND organization_id=$2 AND active)`,r.PathValue("id"),s.OrganizationID).Scan(&exists);err!=nil||!exists{
		fail(w,404,"inventory_item_not_found","El artículo de inventario no existe.")
		return
	}
	_,err:=a.db.Exec(r.Context(),`
		INSERT INTO inventory_location_settings(organization_id,location_id,inventory_item_id,minimum_stock,reorder_point,optimal_stock)
		VALUES($1,$2,$3,$4,$5,$6)
		ON CONFLICT(location_id,inventory_item_id)
		DO UPDATE SET minimum_stock=EXCLUDED.minimum_stock,reorder_point=EXCLUDED.reorder_point,
		  optimal_stock=EXCLUDED.optimal_stock,updated_at=now()
	`,s.OrganizationID,s.LocationID,r.PathValue("id"),in.MinimumStock,in.ReorderPoint,in.OptimalStock)
	if err!=nil{fail(w,503,"inventory_unavailable","No pudimos actualizar los niveles del artículo.");return}
	a.audit(r,"inventory.settings_updated","inventory_item",r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
}

type purchaseReceiptSummary struct{
	ID string `json:"id"`
	Code string `json:"code"`
	PurchaseOrderID string `json:"purchaseOrderId"`
	Number string `json:"number"`
	SupplierName string `json:"supplierName"`
	Notes string `json:"notes"`
	CreatedByName string `json:"createdByName"`
	CreatedAt time.Time `json:"createdAt"`
	ItemCount int `json:"itemCount"`
}
type purchaseReceiptItemView struct{
	ID string `json:"id"`
	PurchaseOrderItemID string `json:"purchaseOrderItemId"`
	InventoryItemID string `json:"inventoryItemId"`
	ItemName string `json:"itemName"`
	Quantity string `json:"quantity"`
	PresentationType string `json:"presentationType"`
	UnitsPerPresentation string `json:"unitsPerPresentation"`
	StockQuantity string `json:"stockQuantity"`
	UnitCost string `json:"unitCost"`
	ReturnedQuantity string `json:"returnedQuantity"`
	ReturnableQuantity string `json:"returnableQuantity"`
}
type purchaseReceiptDetail struct{
	purchaseReceiptSummary
	Items []purchaseReceiptItemView `json:"items"`
}

func (a *API) listPurchaseReceipts(w http.ResponseWriter,r *http.Request){
	s:=r.Context().Value(scopeKey{}).(scope)
	page,size:=pageParams(r)
	q:=strings.TrimSpace(r.URL.Query().Get("q"))
	from:=strings.TrimSpace(r.URL.Query().Get("from"))
	to:=strings.TrimSpace(r.URL.Query().Get("to"))
	where:=`pr.organization_id=$1 AND pr.location_id=$2
	  AND ($3='' OR pr.code ILIKE '%'||$3||'%' OR po.number ILIKE '%'||$3||'%' OR sp.name ILIKE '%'||$3||'%')
	  AND ($4='' OR pr.created_at >= $4::date)
	  AND ($5='' OR pr.created_at < ($5::date + interval '1 day'))`
	var total int
	if err:=a.db.QueryRow(r.Context(),`SELECT count(*) FROM purchase_receipts pr
		JOIN purchase_orders po ON po.id=pr.purchase_order_id AND po.organization_id=pr.organization_id
		JOIN suppliers sp ON sp.id=po.supplier_id AND sp.organization_id=po.organization_id
		WHERE `+where,s.OrganizationID,s.LocationID,q,from,to).Scan(&total);err!=nil{
		fail(w,503,"purchase_receipts_unavailable","No pudimos cargar las recepciones.");return
	}
	rows,err:=a.db.Query(r.Context(),`
		SELECT pr.id,pr.code,po.id,po.number,sp.name,pr.notes,COALESCE(u.full_name,'Usuario no disponible'),pr.created_at,
		  (SELECT count(*) FROM purchase_receipt_items pri WHERE pri.purchase_receipt_id=pr.id AND pri.organization_id=pr.organization_id)
		FROM purchase_receipts pr
		JOIN purchase_orders po ON po.id=pr.purchase_order_id AND po.organization_id=pr.organization_id
		JOIN suppliers sp ON sp.id=po.supplier_id AND sp.organization_id=po.organization_id
		LEFT JOIN users u ON u.id=pr.created_by
		WHERE `+where+` ORDER BY pr.created_at DESC LIMIT $6 OFFSET $7
	`,s.OrganizationID,s.LocationID,q,from,to,size,(page-1)*size)
	if err!=nil{fail(w,503,"purchase_receipts_unavailable","No pudimos cargar las recepciones.");return}
	defer rows.Close()
	items:=[]purchaseReceiptSummary{}
	for rows.Next(){var x purchaseReceiptSummary;if rows.Scan(&x.ID,&x.Code,&x.PurchaseOrderID,&x.Number,&x.SupplierName,&x.Notes,&x.CreatedByName,&x.CreatedAt,&x.ItemCount)!=nil{fail(w,503,"purchase_receipts_unavailable","No pudimos leer las recepciones.");return};items=append(items,x)}
	writeJSON(w,200,map[string]any{"items":items,"total":total,"page":page,"pageSize":size})
}

func (a *API) getPurchaseReceipt(w http.ResponseWriter,r *http.Request){
	s:=r.Context().Value(scopeKey{}).(scope)
	var out purchaseReceiptDetail
	err:=a.db.QueryRow(r.Context(),`
		SELECT pr.id,pr.code,po.id,po.number,sp.name,pr.notes,COALESCE(u.full_name,'Usuario no disponible'),pr.created_at,
		  (SELECT count(*) FROM purchase_receipt_items pri WHERE pri.purchase_receipt_id=pr.id AND pri.organization_id=pr.organization_id)
		FROM purchase_receipts pr
		JOIN purchase_orders po ON po.id=pr.purchase_order_id AND po.organization_id=pr.organization_id
		JOIN suppliers sp ON sp.id=po.supplier_id AND sp.organization_id=po.organization_id
		LEFT JOIN users u ON u.id=pr.created_by
		WHERE pr.id=$1 AND pr.organization_id=$2 AND pr.location_id=$3
	`,r.PathValue("id"),s.OrganizationID,s.LocationID).Scan(&out.ID,&out.Code,&out.PurchaseOrderID,&out.Number,&out.SupplierName,&out.Notes,&out.CreatedByName,&out.CreatedAt,&out.ItemCount)
	if errors.Is(err,pgx.ErrNoRows){fail(w,404,"purchase_receipt_not_found","La recepción no existe en este local.");return}
	if err!=nil{fail(w,503,"purchase_receipt_unavailable","No pudimos cargar la recepción.");return}
	rows,err:=a.db.Query(r.Context(),`
		SELECT pri.id,pri.purchase_order_item_id,pri.inventory_item_id,COALESCE(p.name,ii.name),
		  pri.quantity::text,pri.presentation_type,pri.units_per_presentation::text,pri.stock_quantity::text,pri.unit_cost::text,
		  COALESCE((SELECT sum(x.quantity) FROM purchase_return_items x WHERE x.purchase_receipt_item_id=pri.id AND x.organization_id=pri.organization_id),0)::text,
		  GREATEST(pri.quantity-COALESCE((SELECT sum(x.quantity) FROM purchase_return_items x WHERE x.purchase_receipt_item_id=pri.id AND x.organization_id=pri.organization_id),0),0)::text
		FROM purchase_receipt_items pri
		JOIN inventory_items ii ON ii.id=pri.inventory_item_id AND ii.organization_id=pri.organization_id
		LEFT JOIN products p ON p.id=ii.product_id AND p.organization_id=ii.organization_id
		WHERE pri.purchase_receipt_id=$1 AND pri.organization_id=$2 ORDER BY pri.created_at,pri.id
	`,out.ID,s.OrganizationID)
	if err!=nil{fail(w,503,"purchase_receipt_unavailable","No pudimos cargar el detalle de recepción.");return}
	defer rows.Close();out.Items=[]purchaseReceiptItemView{}
	for rows.Next(){var x purchaseReceiptItemView;if rows.Scan(&x.ID,&x.PurchaseOrderItemID,&x.InventoryItemID,&x.ItemName,&x.Quantity,&x.PresentationType,&x.UnitsPerPresentation,&x.StockQuantity,&x.UnitCost,&x.ReturnedQuantity,&x.ReturnableQuantity)!=nil{fail(w,503,"purchase_receipt_unavailable","No pudimos leer el detalle de recepción.");return};out.Items=append(out.Items,x)}
	writeJSON(w,200,out)
}

type purchaseReturnInput struct{
	IdempotencyKey string `json:"idempotencyKey"`
	Kind string `json:"kind"`
	Reason string `json:"reason"`
	Notes string `json:"notes"`
	Items []struct{
		PurchaseReceiptItemID string `json:"purchaseReceiptItemId"`
		Quantity float64 `json:"quantity"`
	} `json:"items"`
}

func (a *API) createPurchaseReturn(w http.ResponseWriter,r *http.Request){
	s:=r.Context().Value(scopeKey{}).(scope)
	var in purchaseReturnInput
	if json.NewDecoder(r.Body).Decode(&in)!=nil{fail(w,400,"invalid_purchase_return","Revisa la devolución.");return}
	in.IdempotencyKey=strings.TrimSpace(in.IdempotencyKey);in.Kind=strings.TrimSpace(in.Kind);in.Reason=strings.TrimSpace(in.Reason);in.Notes=strings.TrimSpace(in.Notes)
	if len(in.IdempotencyKey)>120||(in.Kind!="supplier_return"&&in.Kind!="receipt_correction")||in.Reason==""||len(in.Reason)>160||len(in.Notes)>500||len(in.Items)==0{
		fail(w,400,"invalid_purchase_return","Indica tipo, motivo y cantidades a devolver.");return
	}
	seen:=map[string]bool{}
	for i:=range in.Items{in.Items[i].PurchaseReceiptItemID=strings.TrimSpace(in.Items[i].PurchaseReceiptItemID);in.Items[i].Quantity=math.Round(in.Items[i].Quantity*1000)/1000;if in.Items[i].PurchaseReceiptItemID==""||in.Items[i].Quantity<=0||seen[in.Items[i].PurchaseReceiptItemID]{fail(w,400,"invalid_purchase_return","Las líneas de devolución no son válidas.");return};seen[in.Items[i].PurchaseReceiptItemID]=true}

	tx,err:=a.db.Begin(r.Context());if err!=nil{fail(w,503,"purchase_return_unavailable","No pudimos iniciar la devolución.");return};defer tx.Rollback(r.Context())
	var receiptID,orderID,supplierID,number string
	err=tx.QueryRow(r.Context(),`
		SELECT pr.id,po.id,po.supplier_id,po.number
		FROM purchase_receipts pr
		JOIN purchase_orders po ON po.id=pr.purchase_order_id AND po.organization_id=pr.organization_id AND po.location_id=pr.location_id
		WHERE pr.id=$1 AND pr.organization_id=$2 AND pr.location_id=$3
		FOR UPDATE OF pr,po
	`,r.PathValue("id"),s.OrganizationID,s.LocationID).Scan(&receiptID,&orderID,&supplierID,&number)
	if errors.Is(err,pgx.ErrNoRows){fail(w,404,"purchase_receipt_not_found","La recepción no existe en este local.");return}
	if err!=nil{fail(w,503,"purchase_return_unavailable","No pudimos validar la recepción.");return}
	if in.IdempotencyKey!=""{
		var existingID,existingCode,existingKind string
		err=tx.QueryRow(r.Context(),`
			SELECT id,code,kind FROM purchase_returns
			WHERE organization_id=$1 AND location_id=$2 AND purchase_receipt_id=$3 AND idempotency_key=$4
		`,s.OrganizationID,s.LocationID,receiptID,in.IdempotencyKey).Scan(&existingID,&existingCode,&existingKind)
		if err==nil{writeJSON(w,200,map[string]any{"id":existingID,"code":existingCode,"kind":existingKind,"purchaseOrderId":orderID,"number":number,"idempotent":true});return}
		if err!=nil&&!errors.Is(err,pgx.ErrNoRows){fail(w,503,"purchase_return_unavailable","No pudimos validar la devolución previa.");return}
	}

	var returnID,code string
	err=tx.QueryRow(r.Context(),`
		INSERT INTO purchase_returns(organization_id,location_id,supplier_id,purchase_order_id,purchase_receipt_id,kind,reason,notes,idempotency_key,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,''),$10) RETURNING id,code
	`,s.OrganizationID,s.LocationID,supplierID,orderID,receiptID,in.Kind,in.Reason,in.Notes,in.IdempotencyKey,s.UserID).Scan(&returnID,&code)
	if err!=nil{fail(w,503,"purchase_return_unavailable","No pudimos crear la devolución.");return}

	for _,requested:=range in.Items{
		var receiptItemID,orderItemID,itemID,presentationType string
		var receivedQty,units,unitCost,alreadyReturned float64
		err=tx.QueryRow(r.Context(),`
			SELECT pri.id,pri.purchase_order_item_id,pri.inventory_item_id,pri.quantity::float8,
			  pri.presentation_type,pri.units_per_presentation::float8,pri.unit_cost::float8,
			  COALESCE((SELECT sum(x.quantity) FROM purchase_return_items x WHERE x.purchase_receipt_item_id=pri.id AND x.organization_id=pri.organization_id),0)::float8
			FROM purchase_receipt_items pri
			WHERE pri.id=$1 AND pri.purchase_receipt_id=$2 AND pri.organization_id=$3
			FOR UPDATE
		`,requested.PurchaseReceiptItemID,receiptID,s.OrganizationID).Scan(&receiptItemID,&orderItemID,&itemID,&receivedQty,&presentationType,&units,&unitCost,&alreadyReturned)
		if errors.Is(err,pgx.ErrNoRows){fail(w,409,"purchase_return_item_invalid","Una línea no pertenece a esta recepción.");return}
		if err!=nil{fail(w,503,"purchase_return_unavailable","No pudimos validar una línea.");return}
		if requested.Quantity>receivedQty-alreadyReturned+0.000001{fail(w,409,"purchase_return_exceeds_received","La devolución supera lo recibido disponible.");return}
		if presentationType!="unit"&&math.Abs(requested.Quantity-math.Round(requested.Quantity))>0.000001{fail(w,400,"invalid_purchase_return_quantity","Paquetes y cajas deben devolverse en cantidades enteras.");return}
		stockQty:=math.Round(requested.Quantity*units*1000)/1000
		bal,err:=lockInventoryBalance(r.Context(),tx,s,itemID);if err!=nil{fail(w,503,"inventory_unavailable","No pudimos bloquear el stock.");return}
		if stockQty>bal.Quantity+0.000001{fail(w,409,"insufficient_stock","No hay stock suficiente para registrar la devolución.");return}
		after:=math.Round((bal.Quantity-stockQty)*1000)/1000
		avg:=bal.AverageUnitCost
		if _,err=tx.Exec(r.Context(),`UPDATE stock_balances SET quantity=$4,average_unit_cost=$5,updated_at=now() WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`,s.OrganizationID,s.LocationID,itemID,after,avg);err!=nil{fail(w,503,"inventory_unavailable","No pudimos actualizar el stock.");return}
		movement:=map[bool]string{true:"receipt_correction",false:"supplier_return"}[in.Kind=="receipt_correction"]
		if err=insertValuedMovement(r.Context(),tx,s,bal.ProductID,itemID,movement,-stockQty,after,bal.AverageUnitCost,"purchase_return",returnID,code+" · "+in.Reason);err!=nil{fail(w,503,"inventory_unavailable","No pudimos registrar el Kárdex.");return}
		if _,err=tx.Exec(r.Context(),`
			INSERT INTO purchase_return_items(organization_id,purchase_return_id,purchase_receipt_item_id,purchase_order_item_id,inventory_item_id,quantity,presentation_type,units_per_presentation,unit_cost)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
		`,s.OrganizationID,returnID,receiptItemID,orderItemID,itemID,requested.Quantity,presentationType,units,unitCost);err!=nil{fail(w,503,"purchase_return_unavailable","No pudimos guardar el detalle de devolución.");return}
		if in.Kind=="receipt_correction"{
			if _,err=tx.Exec(r.Context(),`UPDATE purchase_order_items SET received_quantity=GREATEST(received_quantity-$4,0) WHERE id=$1 AND purchase_order_id=$2 AND organization_id=$3`,orderItemID,orderID,s.OrganizationID,requested.Quantity);err!=nil{fail(w,503,"purchase_return_unavailable","No pudimos corregir la recepción.");return}
		}
	}
	if in.Kind=="receipt_correction"{
		var receivedLines,pendingLines int
		if err=tx.QueryRow(r.Context(),`SELECT count(*) FILTER(WHERE received_quantity>0),count(*) FILTER(WHERE received_quantity<quantity) FROM purchase_order_items WHERE purchase_order_id=$1 AND organization_id=$2`,orderID,s.OrganizationID).Scan(&receivedLines,&pendingLines);err!=nil{fail(w,503,"purchase_return_unavailable","No pudimos recalcular la orden.");return}
		next:="approved";if receivedLines>0{next="partially_received"};if pendingLines==0{next="received"}
		if _,err=tx.Exec(r.Context(),`UPDATE purchase_orders SET status=$4,received_at=CASE WHEN $4='received' THEN COALESCE(received_at,now()) ELSE NULL END,updated_at=now() WHERE id=$1 AND organization_id=$2 AND location_id=$3`,orderID,s.OrganizationID,s.LocationID,next);err!=nil{fail(w,503,"purchase_return_unavailable","No pudimos actualizar la orden.");return}
	}
	if err=tx.Commit(r.Context());err!=nil{fail(w,503,"purchase_return_unavailable","No pudimos confirmar la devolución.");return}
	a.audit(r,"purchase.return_created","purchase_order",orderID)
	writeJSON(w,201,map[string]any{"id":returnID,"code":code,"kind":in.Kind,"purchaseOrderId":orderID,"number":number})
}

type transferInput struct{
	IdempotencyKey string `json:"idempotencyKey"`
	ToLocationID string `json:"toLocationId"`
	Notes string `json:"notes"`
	Items []struct{InventoryItemID string `json:"inventoryItemId"`;Quantity float64 `json:"quantity"`} `json:"items"`
}

func (a *API) createInventoryTransfer(w http.ResponseWriter,r *http.Request){
	s:=r.Context().Value(scopeKey{}).(scope)
	var in transferInput
	if json.NewDecoder(r.Body).Decode(&in)!=nil{fail(w,400,"invalid_transfer","Revisa la transferencia.");return}
	in.IdempotencyKey=strings.TrimSpace(in.IdempotencyKey);in.ToLocationID=strings.TrimSpace(in.ToLocationID);in.Notes=strings.TrimSpace(in.Notes)
	if len(in.IdempotencyKey)>120||in.ToLocationID==""||in.ToLocationID==s.LocationID||len(in.Notes)>500||len(in.Items)==0{fail(w,400,"invalid_transfer","Selecciona otro local y al menos un artículo.");return}
	seen:=map[string]bool{}
	for i:=range in.Items{in.Items[i].InventoryItemID=strings.TrimSpace(in.Items[i].InventoryItemID);in.Items[i].Quantity=math.Round(in.Items[i].Quantity*1000)/1000;if in.Items[i].InventoryItemID==""||in.Items[i].Quantity<=0||seen[in.Items[i].InventoryItemID]{fail(w,400,"invalid_transfer","Las líneas de transferencia no son válidas.");return};seen[in.Items[i].InventoryItemID]=true}
	sort.Slice(in.Items,func(i,j int)bool{return in.Items[i].InventoryItemID<in.Items[j].InventoryItemID})

	tx,err:=a.db.Begin(r.Context());if err!=nil{fail(w,503,"transfer_unavailable","No pudimos iniciar la transferencia.");return};defer tx.Rollback(r.Context())
	var destination string
	if err=tx.QueryRow(r.Context(),`SELECT name FROM locations WHERE id=$1 AND organization_id=$2 AND active FOR SHARE`,in.ToLocationID,s.OrganizationID).Scan(&destination);errors.Is(err,pgx.ErrNoRows){fail(w,404,"destination_not_found","El local destino no existe o está inactivo.");return}else if err!=nil{fail(w,503,"transfer_unavailable","No pudimos validar el destino.");return}
	if in.IdempotencyKey!=""{
		var existingID,existingCode,existingTo string
		err=tx.QueryRow(r.Context(),`SELECT id,code,to_location_id::text FROM inventory_transfers WHERE organization_id=$1 AND from_location_id=$2 AND idempotency_key=$3`,s.OrganizationID,s.LocationID,in.IdempotencyKey).Scan(&existingID,&existingCode,&existingTo)
		if err==nil{writeJSON(w,200,map[string]any{"id":existingID,"code":existingCode,"toLocationId":existingTo,"idempotent":true});return}
		if err!=nil&&!errors.Is(err,pgx.ErrNoRows){fail(w,503,"transfer_unavailable","No pudimos validar la transferencia previa.");return}
	}
	var transferID,code string
	if err=tx.QueryRow(r.Context(),`INSERT INTO inventory_transfers(organization_id,from_location_id,to_location_id,notes,idempotency_key,created_by) VALUES($1,$2,$3,$4,NULLIF($5,''),$6) RETURNING id,code`,s.OrganizationID,s.LocationID,in.ToLocationID,in.Notes,in.IdempotencyKey,s.UserID).Scan(&transferID,&code);err!=nil{fail(w,503,"transfer_unavailable","No pudimos crear la transferencia.");return}

	for _,line:=range in.Items{
		var productID *string;var name string
		if err=tx.QueryRow(r.Context(),`SELECT product_id,COALESCE((SELECT name FROM products p WHERE p.id=inventory_items.product_id),name) FROM inventory_items WHERE id=$1 AND organization_id=$2 AND active FOR SHARE`,line.InventoryItemID,s.OrganizationID).Scan(&productID,&name);errors.Is(err,pgx.ErrNoRows){fail(w,404,"inventory_item_not_found","Un artículo ya no existe.");return}else if err!=nil{fail(w,503,"transfer_unavailable","No pudimos validar los artículos.");return}
		if _,err=tx.Exec(r.Context(),`
			INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity,average_unit_cost)
			VALUES($1,$2,$3,0,0),($1,$4,$3,0,0)
			ON CONFLICT(location_id,inventory_item_id) DO NOTHING
		`,s.OrganizationID,s.LocationID,line.InventoryItemID,in.ToLocationID);err!=nil{fail(w,503,"transfer_unavailable","No pudimos preparar los saldos.");return}
		rows,err:=tx.Query(r.Context(),`SELECT location_id,quantity::float8,average_unit_cost::float8 FROM stock_balances WHERE organization_id=$1 AND inventory_item_id=$2 AND location_id IN ($3,$4) ORDER BY location_id FOR UPDATE`,s.OrganizationID,line.InventoryItemID,s.LocationID,in.ToLocationID)
		if err!=nil{fail(w,503,"transfer_unavailable","No pudimos bloquear los saldos.");return}
		type qv struct{q,c float64};balances:=map[string]qv{}
		for rows.Next(){var loc string;var q,cost float64;if rows.Scan(&loc,&q,&cost)!=nil{rows.Close();fail(w,503,"transfer_unavailable","No pudimos leer los saldos.");return};balances[loc]=qv{q,cost}};rows.Close()
		src:=balances[s.LocationID];dst:=balances[in.ToLocationID]
		if line.Quantity>src.q+0.000001{fail(w,409,"insufficient_stock","No hay stock suficiente de "+name+" para transferir.");return}
		srcAfter:=math.Round((src.q-line.Quantity)*1000)/1000
		dstAfter:=math.Round((dst.q+line.Quantity)*1000)/1000
		dstAvg:=src.c
		if dstAfter>0{dstAvg=math.Round(((dst.q*dst.c)+(line.Quantity*src.c))/dstAfter*10000)/10000}
		srcAvg:=src.c
		if _,err=tx.Exec(r.Context(),`UPDATE stock_balances SET quantity=$4,average_unit_cost=$5,updated_at=now() WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`,s.OrganizationID,s.LocationID,line.InventoryItemID,srcAfter,srcAvg);err!=nil{fail(w,503,"transfer_unavailable","No pudimos descontar el origen.");return}
		if _,err=tx.Exec(r.Context(),`UPDATE stock_balances SET quantity=$4,average_unit_cost=$5,updated_at=now() WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`,s.OrganizationID,in.ToLocationID,line.InventoryItemID,dstAfter,dstAvg);err!=nil{fail(w,503,"transfer_unavailable","No pudimos ingresar el destino.");return}
		if _,err=tx.Exec(r.Context(),`INSERT INTO inventory_transfer_items(organization_id,transfer_id,inventory_item_id,quantity,unit_cost) VALUES($1,$2,$3,$4,$5)`,s.OrganizationID,transferID,line.InventoryItemID,line.Quantity,src.c);err!=nil{fail(w,503,"transfer_unavailable","No pudimos guardar el detalle.");return}
		if err=insertValuedMovement(r.Context(),tx,s,productID,line.InventoryItemID,"transfer_out",-line.Quantity,srcAfter,src.c,"inventory_transfer",transferID,code+" → "+destination);err!=nil{fail(w,503,"transfer_unavailable","No pudimos registrar el Kárdex de origen.");return}
		valueDelta:=math.Round(line.Quantity*src.c*10000)/10000
		balanceValue:=math.Round(dstAfter*dstAvg*10000)/10000
		if _,err=tx.Exec(r.Context(),`
			INSERT INTO stock_movements(organization_id,location_id,product_id,inventory_item_id,movement_type,quantity_delta,balance_after,source_type,source_id,note,created_by,unit_cost,value_delta,balance_value_after)
			VALUES($1,$2,$3,$4,'transfer_in',$5,$6,'inventory_transfer',$7,$8,$9,$10,$11,$12)
		`,s.OrganizationID,in.ToLocationID,productID,line.InventoryItemID,line.Quantity,dstAfter,transferID,code+" desde transferencia",s.UserID,src.c,valueDelta,balanceValue);err!=nil{fail(w,503,"transfer_unavailable","No pudimos registrar el Kárdex de destino.");return}
	}
	if err=tx.Commit(r.Context());err!=nil{fail(w,503,"transfer_unavailable","No pudimos confirmar la transferencia.");return}
	a.audit(r,"inventory.transfer_created","inventory_transfer",transferID)
	writeJSON(w,201,map[string]any{"id":transferID,"code":code,"toLocationId":in.ToLocationID,"toLocationName":destination})
}

func (a *API) listInventoryTransfers(w http.ResponseWriter,r *http.Request){
	s:=r.Context().Value(scopeKey{}).(scope)
	page,size:=pageParams(r)
	var total int
	_ = a.db.QueryRow(r.Context(),`SELECT count(*) FROM inventory_transfers WHERE organization_id=$1 AND (from_location_id=$2 OR to_location_id=$2)`,s.OrganizationID,s.LocationID).Scan(&total)
	rows,err:=a.db.Query(r.Context(),`
		SELECT t.id,t.code,fl.name,tl.name,t.notes,COALESCE(u.full_name,'Usuario no disponible'),t.created_at,
		  (SELECT count(*) FROM inventory_transfer_items x WHERE x.transfer_id=t.id AND x.organization_id=t.organization_id)
		FROM inventory_transfers t JOIN locations fl ON fl.id=t.from_location_id JOIN locations tl ON tl.id=t.to_location_id
		LEFT JOIN users u ON u.id=t.created_by
		WHERE t.organization_id=$1 AND (t.from_location_id=$2 OR t.to_location_id=$2)
		ORDER BY t.created_at DESC LIMIT $3 OFFSET $4
	`,s.OrganizationID,s.LocationID,size,(page-1)*size)
	if err!=nil{fail(w,503,"transfers_unavailable","No pudimos cargar las transferencias.");return}
	defer rows.Close();items:=[]map[string]any{}
	for rows.Next(){var id,code,from,to,note,user string;var at time.Time;var count int;if rows.Scan(&id,&code,&from,&to,&note,&user,&at,&count)!=nil{continue};items=append(items,map[string]any{"id":id,"code":code,"fromLocationName":from,"toLocationName":to,"notes":note,"createdByName":user,"createdAt":at,"itemCount":count})}
	writeJSON(w,200,map[string]any{"items":items,"total":total,"page":page,"pageSize":size})
}


func (a *API) approvePurchaseOrder(w http.ResponseWriter,r *http.Request){
	s:=r.Context().Value(scopeKey{}).(scope)
	tx,err:=a.db.Begin(r.Context());if err!=nil{fail(w,503,"purchase_unavailable","No pudimos aprobar la orden.");return};defer tx.Rollback(r.Context())
	var status string
	err=tx.QueryRow(r.Context(),`SELECT status FROM purchase_orders WHERE id=$1 AND organization_id=$2 AND location_id=$3 FOR UPDATE`,r.PathValue("id"),s.OrganizationID,s.LocationID).Scan(&status)
	if errors.Is(err,pgx.ErrNoRows){fail(w,404,"purchase_not_found","La orden de compra no existe.");return}
	if err!=nil{fail(w,503,"purchase_unavailable","No pudimos validar la orden.");return}
	if status!="pending_approval"{fail(w,409,"purchase_transition_invalid","Solo una orden por aprobar puede aprobarse.");return}
	if _,err=tx.Exec(r.Context(),`UPDATE purchase_orders SET status='approved',approved_at=now(),updated_at=now() WHERE id=$1 AND organization_id=$2 AND location_id=$3`,r.PathValue("id"),s.OrganizationID,s.LocationID);err!=nil{fail(w,503,"purchase_unavailable","No pudimos aprobar la orden.");return}
	if err=tx.Commit(r.Context());err!=nil{fail(w,503,"purchase_unavailable","No pudimos confirmar la aprobación.");return}
	a.audit(r,"purchase.approved","purchase_order",r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
}
