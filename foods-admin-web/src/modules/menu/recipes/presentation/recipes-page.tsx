"use client";
import "./recipes.css";
import {useMemo,useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,FieldLabel,Icon,Input,PageHeader,RowActionButton,Select,Status,Textarea} from "@/design-system";
import {useFeedback,useSession} from "@/providers";
import {
 getRecipe,
 listRecipeInventory,
 listRecipeProducts,
 listRecipes,
 saveRecipe,
 type RecipeDraft,
 type RecipeSummary,
} from "../infrastructure/recipes-api";

const empty=(productId=""):RecipeDraft=>({productId,yieldQuantity:"1",notes:"",active:true,items:[]});

export function RecipesPage(){
 const qc=useQueryClient();
 const{notify}=useFeedback();
 const{can}=useSession();
 const[q,setQ]=useState("");
 const[draft,setDraft]=useState<RecipeDraft|null>(null);
 const[editingProductId,setEditingProductId]=useState("");
 const[loadingId,setLoadingId]=useState("");
 const[attempted,setAttempted]=useState(false);
 const[statusTarget,setStatusTarget]=useState<RecipeSummary|null>(null);
 const canManage=can("recipes.manage");

 const recipes=useQuery({queryKey:["recipes",q],queryFn:()=>listRecipes(q)});
 const products=useQuery({queryKey:["recipe-products"],queryFn:listRecipeProducts});
 const inventory=useQuery({queryKey:["recipe-inventory"],queryFn:listRecipeInventory});
 const availableProducts=useMemo(()=>(products.data?.items??[]).filter(p=>p.quantityControl!=="inventory"),[products.data]);

 const save=useMutation({
  mutationFn:saveRecipe,
  onSuccess:r=>{
   setDraft(null);
   setEditingProductId("");
   setAttempted(false);
   void qc.invalidateQueries({queryKey:["recipes"]});
   notify({tone:"success",title:"Receta guardada",message:r.productName+" ya usa la configuración de consumo indicada."});
  },
  onError:e=>notify({tone:"danger",title:"No se pudo guardar",message:e.message}),
 });

 const statusChange=useMutation({
  mutationFn:async({productId,active}:{productId:string;active:boolean})=>{
   const recipe=await getRecipe(productId);
   return saveRecipe({
    productId:recipe.productId,
    yieldQuantity:recipe.yieldQuantity,
    notes:recipe.notes,
    active,
    items:recipe.items.map(item=>({
     inventoryItemId:item.inventoryItemId,
     quantity:item.quantity,
     wastePercent:item.wastePercent,
    })),
   });
  },
  onSuccess:(_,variables)=>{
   setStatusTarget(null);
   void qc.invalidateQueries({queryKey:["recipes"]});
   notify({
    tone:"success",
    title:variables.active?"Receta activada":"Receta desactivada",
    message:variables.active?"El consumo automático vuelve a estar activo.":"La receta queda conservada sin consumir insumos automáticamente.",
   });
  },
  onError:e=>notify({tone:"danger",title:"No se pudo cambiar el estado",message:e.message}),
 });

 function openNew(){
  setEditingProductId("");
  setAttempted(false);
  setDraft(empty());
 }

 async function edit(productId:string){
  setLoadingId(productId);
  setAttempted(false);
  try{
   const r=await getRecipe(productId);
   setEditingProductId(productId);
   setDraft({
    productId:r.productId,
    yieldQuantity:r.yieldQuantity,
    notes:r.notes,
    active:r.active,
    items:r.items.map(i=>({inventoryItemId:i.inventoryItemId,quantity:i.quantity,wastePercent:i.wastePercent})),
   });
  }catch(e){
   notify({tone:"danger",title:"No se pudo cargar la receta",message:e instanceof Error?e.message:"Inténtalo nuevamente."});
  }finally{
   setLoadingId("");
  }
 }

 function closeRecipe(){
  if(save.isPending)return;
  setDraft(null);
  setEditingProductId("");
  setAttempted(false);
 }

 function addIngredient(){
  if(!draft)return;
  const first=inventory.data?.items.find(i=>!draft.items.some(x=>x.inventoryItemId===i.id));
  if(first)setDraft({...draft,items:[...draft.items,{inventoryItemId:first.id,quantity:"1",wastePercent:"0"}]});
 }

 function submit(event:React.FormEvent){
  event.preventDefault();
  if(!draft)return;
  setAttempted(true);
  const ids=draft.items.map(item=>item.inventoryItemId).filter(Boolean);
  const duplicateIngredient=new Set(ids).size!==ids.length;
  const invalidItems=draft.items.some(item=>{
   const quantity=Number(item.quantity);
   const waste=Number(item.wastePercent||0);
   return !item.inventoryItemId||!Number.isFinite(quantity)||quantity<=0||!Number.isFinite(waste)||waste<0||waste>=100;
  });
  if(!draft.productId||!Number.isFinite(Number(draft.yieldQuantity))||Number(draft.yieldQuantity)<=0||!draft.items.length||invalidItems||duplicateIngredient){
   notify({tone:"danger",title:"Receta incompleta",message:"Revisa producto, rendimiento e insumos antes de guardar."});
   return;
  }
  save.mutate(draft);
 }

 const items=recipes.data?.items??[];

 return <>
  <PageHeader
   eyebrow="CARTA Y PRODUCCIÓN"
   title="Recetas y producción"
   description="Define el consumo de insumos de los productos preparados. Inventario y Kárdex continúan operando aunque una receta esté desactivada."
   action={canManage?<Button icon="plus" onClick={openNew}>Nueva receta</Button>:undefined}
  />

  <section className="panel standardized-management">
   <div className="toolbar">
    <label className="ds-input-shell"><Icon name="search" size={17}/><Input value={q} onChange={e=>setQ(e.target.value)} placeholder="Buscar producto con receta..."/></label>
   </div>
   {recipes.isLoading?<RecipeListSkeleton/>:recipes.isError?
    <div className="catalog-state error"><Icon name="alert" size={24}/><b>No pudimos cargar las recetas</b><p>{recipes.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>recipes.refetch()}>Reintentar</Button></div>
   :!items.length?
    <div className="catalog-state"><Icon name="chefHat" size={24}/><b>Aún no hay recetas configuradas</b><p>Configura una receta cuando quieras descontar insumos automáticamente al vender un producto preparado.</p>{canManage&&<Button icon="plus" onClick={openNew}>Nueva receta</Button>}</div>
   :<div className="table-wrap hover-scroll" tabIndex={0}><table><thead><tr><th>PRODUCTO</th><th>RENDIMIENTO</th><th>INSUMOS</th><th>ESTADO</th><th>ACCIONES</th></tr></thead><tbody>{items.map((r,i)=><tr className={i%2?"alternate":""} key={r.id}><td><b>{r.productName}</b></td><td>{r.yieldQuantity}</td><td>{r.itemCount}</td><td><Status tone={r.active?"green":"gray"}>{r.active?"Activa":"Inactiva"}</Status></td><td>{canManage?<div className="table-actions"><RowActionButton action="edit" label={loadingId===r.productId?"Cargando receta":"Editar receta"} disabled={loadingId===r.productId||statusChange.isPending} onClick={()=>void edit(r.productId)}/><RowActionButton action={r.active?"deactivate":"activate"} label={r.active?"Desactivar receta":"Activar receta"} disabled={statusChange.isPending||loadingId===r.productId} onClick={()=>r.active?setStatusTarget(r):statusChange.mutate({productId:r.productId,active:true})}/></div>:<span className="recipe-read-only">Solo lectura</span>}</td></tr>)}</tbody></table></div>}
  </section>

  {draft&&<div className="modal-backdrop modal-overlay-in" role="presentation">
   <section className="crud-modal recipe-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="recipe-modal-title" aria-busy={save.isPending}>
    <header className="recipe-modal-header">
     <span className="modal-title-icon"><Icon name="chefHat" size={18}/></span>
     <div>
      <small>RECETA DE INSUMOS</small>
      <h2 id="recipe-modal-title">Configurar receta</h2>
      <p>Define cuánto produce la receta y qué insumos consume.</p>
     </div>
     <button className="recipe-modal-close" onClick={closeRecipe} disabled={save.isPending} aria-label="Cerrar"><Icon name="close" size={18}/></button>
    </header>

    <form className="recipe-form" onSubmit={submit} noValidate>
     <div className="recipe-form-body">
      <section className="recipe-main-fields" aria-label="Datos de la receta">
       <label className="recipe-product-field">
        <span>Producto</span>
        <Select value={draft.productId} disabled={Boolean(editingProductId)||products.isLoading||save.isPending} aria-invalid={attempted&&!draft.productId} onChange={e=>setDraft({...draft,productId:e.target.value})}>
         <option value="">{products.isLoading?"Cargando productos...":"Selecciona producto"}</option>
         {availableProducts.map(p=><option key={p.id} value={p.id}>{p.name}</option>)}
        </Select>
        {attempted&&!draft.productId&&<small className="recipe-field-error">Selecciona el producto de la receta.</small>}
        {products.isError&&<small className="recipe-field-error">No se pudieron cargar los productos.</small>}
       </label>

       <label className="recipe-yield-field">
        <FieldLabel hint="Cantidad de producto final que obtienes con los insumos indicados.">Rendimiento</FieldLabel>
        <Input type="number" min="0.001" step="0.001" inputMode="decimal" value={draft.yieldQuantity} aria-invalid={attempted&&Number(draft.yieldQuantity)<=0} onChange={e=>setDraft({...draft,yieldQuantity:e.target.value})}/>
        {attempted&&Number(draft.yieldQuantity)<=0&&<small className="recipe-field-error">Debe ser mayor que cero.</small>}
       </label>

       <label className="recipe-notes-field">
        <span>Notas <small>Opcional</small></span>
        <Textarea rows={2} maxLength={500} value={draft.notes} onChange={e=>setDraft({...draft,notes:e.target.value})} placeholder="Ej. Preparar por lote y conservar refrigerado."/>
       </label>
      </section>

      <section className="recipe-ingredients">
       <header className="recipe-ingredients-toolbar">
        <div>
         <span className="recipe-section-icon"><Icon name="stock" size={17}/></span>
         <span><b>Insumos de la receta</b><small>Indica la cantidad consumida para el rendimiento definido arriba.</small></span>
        </div>
        <Button kind="secondary" icon="plus" disabled={inventory.isLoading||save.isPending||draft.items.length>=(inventory.data?.items.length??0)} onClick={addIngredient}>Agregar insumo</Button>
       </header>

       {inventory.isError?<div className="recipe-inline-state error"><Icon name="alert" size={18}/><span><b>No pudimos cargar los insumos</b><small>Reintenta la carga antes de guardar la receta.</small></span><Button kind="secondary" icon="refresh" onClick={()=>inventory.refetch()}>Reintentar</Button></div>
       :draft.items.length===0?<div className={attempted?"recipe-inline-state warning":"recipe-inline-state"}><Icon name="stock" size={19}/><span><b>Agrega al menos un insumo</b><small>La receta necesita un insumo con cantidad válida para poder guardarse.</small></span></div>
       :<div className="recipe-ingredients-grid">
        <div className="recipe-ingredients-head" aria-hidden="true"><span>INSUMO</span><span>CANTIDAD</span><span>MERMA %</span><span>ACCIÓN</span></div>
        {draft.items.map((item,index)=>{
         const selected=inventory.data?.items.find(source=>source.id===item.inventoryItemId);
         const quantityInvalid=attempted&&Number(item.quantity)<=0;
         const waste=Number(item.wastePercent||0);
         const wasteInvalid=attempted&&(waste<0||waste>=100||!Number.isFinite(waste));
         return <div className="recipe-ingredient-row" key={index}>
          <label className="recipe-ingredient-source" data-label="INSUMO">
           <Select aria-label={`Insumo ${index+1}`} value={item.inventoryItemId} disabled={save.isPending} onChange={e=>setDraft({...draft,items:draft.items.map((x,i)=>i===index?{...x,inventoryItemId:e.target.value}:x)})}>
            {(inventory.data?.items??[]).map(source=><option key={source.id} value={source.id} disabled={source.id!==item.inventoryItemId&&draft.items.some(x=>x.inventoryItemId===source.id)}>{source.name} · {source.unit}</option>)}
           </Select>
          </label>
          <label className="recipe-ingredient-quantity" data-label="CANTIDAD">
           <div><Input aria-label={`Cantidad del insumo ${index+1}`} type="number" min="0.001" step="0.001" inputMode="decimal" disabled={save.isPending} aria-invalid={quantityInvalid} value={item.quantity} onChange={e=>setDraft({...draft,items:draft.items.map((x,i)=>i===index?{...x,quantity:e.target.value}:x)})}/>{selected&&<small>{selected.unit}</small>}</div>
           {quantityInvalid&&<small className="recipe-field-error">Mayor que 0.</small>}
          </label>
          <label className="recipe-ingredient-waste" data-label="MERMA %">
           <Input aria-label={`Merma del insumo ${index+1}`} type="number" min="0" max="99.9999" step="0.01" inputMode="decimal" disabled={save.isPending} aria-invalid={wasteInvalid} value={item.wastePercent} onChange={e=>setDraft({...draft,items:draft.items.map((x,i)=>i===index?{...x,wastePercent:e.target.value}:x)})}/>
           {wasteInvalid&&<small className="recipe-field-error">Entre 0 y 99.99.</small>}
          </label>
          <div className="recipe-ingredient-action" data-label="ACCIÓN"><RowActionButton action="remove" label={`Quitar ${selected?.name??"insumo"}`} disabled={save.isPending} onClick={()=>setDraft({...draft,items:draft.items.filter((_,i)=>i!==index)})}/></div>
         </div>;
        })}
       </div>}
      </section>
     </div>

     <footer className="recipe-form-footer">
      <Button kind="ghost" disabled={save.isPending} onClick={closeRecipe}>Cancelar</Button>
      <Button type="submit" icon="check" disabled={save.isPending}>{save.isPending?"Guardando…":"Guardar"}</Button>
     </footer>
    </form>
   </section>
  </div>}

  <ConfirmDialog
   open={Boolean(statusTarget)}
   title="Desactivar receta"
   description={statusTarget?`La receta de ${statusTarget.productName} quedará conservada, pero dejará de descontar insumos automáticamente.`:""}
   confirmLabel="Desactivar"
   pending={statusChange.isPending}
   onCancel={()=>setStatusTarget(null)}
   onConfirm={()=>statusTarget&&statusChange.mutate({productId:statusTarget.productId,active:false})}
  />
 </>;
}

function RecipeListSkeleton(){
 return <div className="recipe-list-skeleton" aria-label="Cargando recetas">
  <div className="recipe-list-skeleton-head">{Array.from({length:5},(_,i)=><i key={i}/>)}</div>
  {Array.from({length:5},(_,row)=><div className="recipe-list-skeleton-row" key={row}>{Array.from({length:5},(_,cell)=><i key={cell}/>)}</div>)}
 </div>;
}
