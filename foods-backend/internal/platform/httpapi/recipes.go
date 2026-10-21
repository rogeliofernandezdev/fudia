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

type moduleRowQuerier interface {
	QueryRow(context.Context,string,...any) pgx.Row
}

func moduleEnabled(ctx context.Context,q moduleRowQuerier,organizationID,key string)(bool,error){
	var active bool
	err:=q.QueryRow(ctx,`SELECT active FROM organization_modules WHERE organization_id=$1 AND module_key=$2`,organizationID,key).Scan(&active)
	if errors.Is(err,pgx.ErrNoRows){return true,nil}
	return active,err
}

func recipesRuntimeEnabled(ctx context.Context,q moduleRowQuerier,organizationID string)(bool,error){
	recipes,err:=moduleEnabled(ctx,q,organizationID,"recetas")
	if err!=nil||!recipes{return recipes,err}
	inventory,err:=moduleEnabled(ctx,q,organizationID,"inventario")
	if err!=nil{return false,err}
	return inventory,nil
}

type recipeItemView struct{
	ID string `json:"id"`
	InventoryItemID string `json:"inventoryItemId"`
	Name string `json:"name"`
	Unit string `json:"unit"`
	Quantity string `json:"quantity"`
	WastePercent string `json:"wastePercent"`
}
type recipeView struct{
	ID string `json:"id"`
	ProductID string `json:"productId"`
	ProductName string `json:"productName"`
	YieldQuantity string `json:"yieldQuantity"`
	Notes string `json:"notes"`
	Active bool `json:"active"`
	UpdatedAt time.Time `json:"updatedAt"`
	Items []recipeItemView `json:"items"`
}
type recipeInput struct{
	YieldQuantity float64 `json:"yieldQuantity"`
	Notes string `json:"notes"`
	Active bool `json:"active"`
	Items []struct{
		InventoryItemID string `json:"inventoryItemId"`
		Quantity float64 `json:"quantity"`
		WastePercent float64 `json:"wastePercent"`
	} `json:"items"`
}

func (a *API) listRecipes(w http.ResponseWriter,r *http.Request){
	s:=r.Context().Value(scopeKey{}).(scope)
	enabled,err:=recipesRuntimeEnabled(r.Context(),a.db,s.OrganizationID)
	if err!=nil{fail(w,503,"module_unavailable","No pudimos validar el módulo de recetas.");return}
	if !enabled{fail(w,403,"module_disabled","El módulo Recetas no está habilitado para esta empresa.");return}
	page,size:=pageParams(r)
	q:=strings.TrimSpace(r.URL.Query().Get("q"))
	var total int
	if err=a.db.QueryRow(r.Context(),`
		SELECT count(*) FROM product_recipes pr
		JOIN products p ON p.id=pr.product_id AND p.organization_id=pr.organization_id
		WHERE pr.organization_id=$1 AND ($2='' OR p.name ILIKE '%'||$2||'%')
	`,s.OrganizationID,q).Scan(&total);err!=nil{fail(w,503,"recipes_unavailable","No pudimos cargar las recetas.");return}
	rows,err:=a.db.Query(r.Context(),`
		SELECT pr.id,pr.product_id,p.name,pr.yield_quantity::text,pr.notes,pr.active,pr.updated_at,
		  (SELECT count(*) FROM product_recipe_items ri WHERE ri.recipe_id=pr.id AND ri.organization_id=pr.organization_id)
		FROM product_recipes pr
		JOIN products p ON p.id=pr.product_id AND p.organization_id=pr.organization_id
		WHERE pr.organization_id=$1 AND ($2='' OR p.name ILIKE '%'||$2||'%')
		ORDER BY pr.active DESC,p.name LIMIT $3 OFFSET $4
	`,s.OrganizationID,q,size,(page-1)*size)
	if err!=nil{fail(w,503,"recipes_unavailable","No pudimos cargar las recetas.");return}
	defer rows.Close();items:=[]map[string]any{}
	for rows.Next(){var id,pid,name,yield,notes string;var active bool;var updated time.Time;var count int;if rows.Scan(&id,&pid,&name,&yield,&notes,&active,&updated,&count)!=nil{continue};items=append(items,map[string]any{"id":id,"productId":pid,"productName":name,"yieldQuantity":yield,"notes":notes,"active":active,"updatedAt":updated,"itemCount":count})}
	writeJSON(w,200,map[string]any{"items":items,"total":total,"page":page,"pageSize":size})
}

func (a *API) getRecipe(w http.ResponseWriter,r *http.Request){
	s:=r.Context().Value(scopeKey{}).(scope)
	enabled,err:=recipesRuntimeEnabled(r.Context(),a.db,s.OrganizationID)
	if err!=nil{fail(w,503,"module_unavailable","No pudimos validar el módulo de recetas.");return}
	if !enabled{fail(w,403,"module_disabled","El módulo Recetas no está habilitado para esta empresa.");return}
	var out recipeView
	err=a.db.QueryRow(r.Context(),`
		SELECT pr.id,pr.product_id,p.name,pr.yield_quantity::text,pr.notes,pr.active,pr.updated_at
		FROM product_recipes pr JOIN products p ON p.id=pr.product_id AND p.organization_id=pr.organization_id
		WHERE pr.product_id=$1 AND pr.organization_id=$2
	`,r.PathValue("productId"),s.OrganizationID).Scan(&out.ID,&out.ProductID,&out.ProductName,&out.YieldQuantity,&out.Notes,&out.Active,&out.UpdatedAt)
	if errors.Is(err,pgx.ErrNoRows){fail(w,404,"recipe_not_found","El producto todavía no tiene receta.");return}
	if err!=nil{fail(w,503,"recipe_unavailable","No pudimos cargar la receta.");return}
	rows,err:=a.db.Query(r.Context(),`
		SELECT ri.id,ri.inventory_item_id,COALESCE(p.name,ii.name),ii.unit,ri.quantity::text,ri.waste_percent::text
		FROM product_recipe_items ri
		JOIN inventory_items ii ON ii.id=ri.inventory_item_id AND ii.organization_id=ri.organization_id
		LEFT JOIN products p ON p.id=ii.product_id AND p.organization_id=ii.organization_id
		WHERE ri.recipe_id=$1 AND ri.organization_id=$2 ORDER BY COALESCE(p.name,ii.name)
	`,out.ID,s.OrganizationID)
	if err!=nil{fail(w,503,"recipe_unavailable","No pudimos cargar los ingredientes.");return}
	defer rows.Close();out.Items=[]recipeItemView{}
	for rows.Next(){var x recipeItemView;if rows.Scan(&x.ID,&x.InventoryItemID,&x.Name,&x.Unit,&x.Quantity,&x.WastePercent)!=nil{fail(w,503,"recipe_unavailable","No pudimos leer los ingredientes.");return};out.Items=append(out.Items,x)}
	writeJSON(w,200,out)
}

func (a *API) saveRecipe(w http.ResponseWriter,r *http.Request){
	s:=r.Context().Value(scopeKey{}).(scope)
	enabled,err:=recipesRuntimeEnabled(r.Context(),a.db,s.OrganizationID)
	if err!=nil{fail(w,503,"module_unavailable","No pudimos validar el módulo de recetas.");return}
	if !enabled{fail(w,403,"module_disabled","Habilita Recetas antes de configurarlas.");return}
	var in recipeInput
	if json.NewDecoder(r.Body).Decode(&in)!=nil||in.YieldQuantity<=0||len(in.Items)==0||len(in.Notes)>500{fail(w,400,"invalid_recipe","Indica rendimiento e insumos válidos.");return}
	in.YieldQuantity=math.Round(in.YieldQuantity*1000)/1000
	seen:=map[string]bool{}
	for i:=range in.Items{
		x:=&in.Items[i];x.InventoryItemID=strings.TrimSpace(x.InventoryItemID);x.Quantity=math.Round(x.Quantity*1000)/1000;x.WastePercent=math.Round(x.WastePercent*10000)/10000
		if x.InventoryItemID==""||x.Quantity<=0||x.WastePercent<0||x.WastePercent>=100||seen[x.InventoryItemID]{fail(w,400,"invalid_recipe","Los insumos de la receta no son válidos.");return};seen[x.InventoryItemID]=true
	}
	tx,err:=a.db.Begin(r.Context());if err!=nil{fail(w,503,"recipe_unavailable","No pudimos guardar la receta.");return};defer tx.Rollback(r.Context())
	var productName,control string
	err=tx.QueryRow(r.Context(),`SELECT name,quantity_control FROM products WHERE id=$1 AND organization_id=$2 AND active FOR UPDATE`,r.PathValue("productId"),s.OrganizationID).Scan(&productName,&control)
	if errors.Is(err,pgx.ErrNoRows){fail(w,404,"product_not_found","El producto no existe.");return}
	if err!=nil{fail(w,503,"recipe_unavailable","No pudimos validar el producto.");return}
	if control=="inventory"{fail(w,409,"recipe_inventory_conflict","Un producto controlado directamente por inventario no debe descontarse también por receta.");return}
	for _,x:=range in.Items{
		var ok bool
		if err=tx.QueryRow(r.Context(),`SELECT EXISTS(SELECT 1 FROM inventory_items WHERE id=$1 AND organization_id=$2 AND active)`,x.InventoryItemID,s.OrganizationID).Scan(&ok);err!=nil||!ok{fail(w,409,"recipe_item_invalid","Un insumo ya no está disponible.");return}
	}
	var recipeID string
	err=tx.QueryRow(r.Context(),`
		INSERT INTO product_recipes(organization_id,product_id,yield_quantity,notes,active,created_by,updated_by)
		VALUES($1,$2,$3,$4,$5,$6,$6)
		ON CONFLICT(organization_id,product_id)
		DO UPDATE SET yield_quantity=EXCLUDED.yield_quantity,notes=EXCLUDED.notes,active=EXCLUDED.active,updated_by=EXCLUDED.updated_by,updated_at=now()
		RETURNING id
	`,s.OrganizationID,r.PathValue("productId"),in.YieldQuantity,strings.TrimSpace(in.Notes),in.Active,s.UserID).Scan(&recipeID)
	if err!=nil{fail(w,503,"recipe_unavailable","No pudimos guardar la receta.");return}
	if _,err=tx.Exec(r.Context(),`DELETE FROM product_recipe_items WHERE recipe_id=$1 AND organization_id=$2`,recipeID,s.OrganizationID);err!=nil{fail(w,503,"recipe_unavailable","No pudimos actualizar los insumos.");return}
	for _,x:=range in.Items{
		if _,err=tx.Exec(r.Context(),`INSERT INTO product_recipe_items(organization_id,recipe_id,inventory_item_id,quantity,waste_percent) VALUES($1,$2,$3,$4,$5)`,s.OrganizationID,recipeID,x.InventoryItemID,x.Quantity,x.WastePercent);err!=nil{fail(w,503,"recipe_unavailable","No pudimos guardar un insumo.");return}
	}
	if err=tx.Commit(r.Context());err!=nil{fail(w,503,"recipe_unavailable","No pudimos confirmar la receta.");return}
	a.audit(r,"recipe.saved","product",r.PathValue("productId"))
	writeJSON(w,200,map[string]any{"id":recipeID,"productId":r.PathValue("productId"),"productName":productName})
}

func preparedRecipeProductUsage(items []preparedOrderItem) map[string]float64 {
	usage:=map[string]float64{}
	for _,item:=range items{
		if item.ProductID!=nil{usage[*item.ProductID]+=item.Qty}
		if item.ItemType=="combo"{for _,sel:=range item.Selections{usage[sel.ProductID]+=item.Qty}}
	}
	return usage
}

func desiredRecipeInventoryUsage(ctx context.Context,tx pgx.Tx,s scope,items []preparedOrderItem)(map[string]float64,error){
	enabled,err:=recipesRuntimeEnabled(ctx,tx,s.OrganizationID);if err!=nil{return nil,err}
	out:=map[string]float64{};if !enabled{return out,nil}
	products:=preparedRecipeProductUsage(items)
	ids:=make([]string,0,len(products));for id:=range products{ids=append(ids,id)};sort.Strings(ids)
	for _,productID:=range ids{
		sold:=products[productID]
		var recipeID string;var yield float64
		err=tx.QueryRow(ctx,`SELECT id,yield_quantity::float8 FROM product_recipes WHERE organization_id=$1 AND product_id=$2 AND active`,s.OrganizationID,productID).Scan(&recipeID,&yield)
		if errors.Is(err,pgx.ErrNoRows){continue};if err!=nil{return nil,err}
		rows,err:=tx.Query(ctx,`SELECT inventory_item_id::text,quantity::float8,waste_percent::float8 FROM product_recipe_items WHERE recipe_id=$1 AND organization_id=$2 ORDER BY inventory_item_id`,recipeID,s.OrganizationID);if err!=nil{return nil,err}
		for rows.Next(){var itemID string;var qty,waste float64;if err=rows.Scan(&itemID,&qty,&waste);err!=nil{rows.Close();return nil,err};need:=sold/yield*qty*(1+waste/100);out[itemID]+=math.Round(need*1000)/1000};rows.Close();if err=rows.Err();err!=nil{return nil,err}
	}
	return out,nil
}

func loadNetRecipeUsage(ctx context.Context,tx pgx.Tx,s scope,orderID string)(map[string]float64,error){
	rows,err:=tx.Query(ctx,`
		SELECT inventory_item_id::text,COALESCE(sum(-quantity_delta),0)::float8
		FROM stock_movements
		WHERE organization_id=$1 AND location_id=$2 AND source_type='order' AND source_id=$3
		  AND movement_type IN ('recipe_consumption','recipe_reversal')
		GROUP BY inventory_item_id
	`,s.OrganizationID,s.LocationID,orderID)
	if err!=nil{return nil,err};defer rows.Close()
	out:=map[string]float64{};for rows.Next(){var id string;var qty float64;if err=rows.Scan(&id,&qty);err!=nil{return nil,err};if math.Abs(qty)>0.000001{out[id]=qty}}
	return out,rows.Err()
}

func recipeUsageDelta(previous,next map[string]float64)map[string]float64{
	ids:=map[string]bool{};for id:=range previous{ids[id]=true};for id:=range next{ids[id]=true}
	out:=map[string]float64{};for id:=range ids{d:=next[id]-previous[id];if math.Abs(d)>0.000001{out[id]=d}}
	return out
}

func applyRecipeUsageDelta(ctx context.Context,tx pgx.Tx,s scope,orderID string,delta map[string]float64)*orderPreparationError{
	ids:=make([]string,0,len(delta));for id:=range delta{ids=append(ids,id)};sort.Strings(ids)
	for _,itemID:=range ids{
		change:=math.Round(delta[itemID]*1000)/1000
		if math.Abs(change)<=0.000001{continue}
		bal,err:=lockInventoryBalance(ctx,tx,s,itemID)
		if errors.Is(err,pgx.ErrNoRows){return &orderPreparationError{Status:409,Code:"recipe_item_unavailable",Message:"Un insumo de receta ya no está disponible."}}
		if err!=nil{return &orderPreparationError{Status:503,Code:"recipe_inventory_unavailable",Message:"No pudimos validar el stock de receta."}}
		if change>bal.Quantity+0.000001{return &orderPreparationError{Status:409,Code:"insufficient_recipe_stock",Message:"No hay stock suficiente de "+bal.Name+" para preparar el pedido."}}
		after:=math.Round((bal.Quantity-change)*1000)/1000
		avg:=bal.AverageUnitCost
		if _,err=tx.Exec(ctx,`UPDATE stock_balances SET quantity=$4,average_unit_cost=$5,updated_at=now() WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`,s.OrganizationID,s.LocationID,itemID,after,avg);err!=nil{return &orderPreparationError{Status:503,Code:"recipe_inventory_unavailable",Message:"No pudimos actualizar el stock de receta."}}
		movement:="recipe_consumption";note:="Consumo automático por receta"
		if change<0{movement="recipe_reversal";note="Reversa de consumo por receta"}
		if err=insertValuedMovement(ctx,tx,s,bal.ProductID,itemID,movement,-change,after,bal.AverageUnitCost,"order",orderID,note);err!=nil{return &orderPreparationError{Status:503,Code:"recipe_inventory_unavailable",Message:"No pudimos registrar el Kárdex de receta."}}
	}
	return nil
}
