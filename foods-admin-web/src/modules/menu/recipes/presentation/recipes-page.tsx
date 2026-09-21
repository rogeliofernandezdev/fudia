"use client";
import {useMemo,useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,Icon,Input,PageHeader,Select,Status,Textarea} from "@/design-system";
import {useFeedback} from "@/providers";
import {getRecipe,listRecipeInventory,listRecipeProducts,listRecipes,saveRecipe,type RecipeDraft} from "../infrastructure/recipes-api";

const empty=(productId=""):RecipeDraft=>({productId,yieldQuantity:"1",notes:"",active:true,items:[]});

export function RecipesPage(){
 const qc=useQueryClient();const{notify}=useFeedback();
 const[q,setQ]=useState("");const[draft,setDraft]=useState<RecipeDraft|null>(null);const[loadingId,setLoadingId]=useState("");
 const recipes=useQuery({queryKey:["recipes",q],queryFn:()=>listRecipes(q)});
 const products=useQuery({queryKey:["recipe-products"],queryFn:listRecipeProducts});
 const inventory=useQuery({queryKey:["recipe-inventory"],queryFn:listRecipeInventory});
 const availableProducts=useMemo(()=>(products.data?.items??[]).filter(p=>p.quantityControl!=="inventory"),[products.data]);
 const save=useMutation({mutationFn:saveRecipe,onSuccess:r=>{setDraft(null);void qc.invalidateQueries({queryKey:["recipes"]});notify({tone:"success",title:"Receta guardada",message:r.productName+" ya puede controlar insumos cuando el módulo Recetas está activo."})},onError:e=>notify({tone:"danger",title:"No se pudo guardar",message:e.message})});
 async function edit(productId:string){
  setLoadingId(productId);
  try{const r=await getRecipe(productId);setDraft({productId:r.productId,yieldQuantity:r.yieldQuantity,notes:r.notes,active:r.active,items:r.items.map(i=>({inventoryItemId:i.inventoryItemId,quantity:i.quantity,wastePercent:i.wastePercent}))})}
  catch{setDraft(empty(productId))}
  finally{setLoadingId("")}
 }
 function addIngredient(){if(!draft)return;const first=inventory.data?.items.find(i=>!draft.items.some(x=>x.inventoryItemId===i.id));if(first)setDraft({...draft,items:[...draft.items,{inventoryItemId:first.id,quantity:"1",wastePercent:"0"}]})}
 function submit(){
  if(!draft||!draft.productId||Number(draft.yieldQuantity)<=0||!draft.items.length||draft.items.some(i=>!i.inventoryItemId||Number(i.quantity)<=0)){notify({tone:"danger",title:"Receta incompleta",message:"Selecciona producto, rendimiento e insumos con cantidades válidas."});return}
  save.mutate(draft)
 }
 return <>
  <PageHeader eyebrow="CARTA Y PRODUCCIÓN" title="Recetas y producción" description="Módulo opcional. Al desactivarlo, Productos, Inventario, Compras y Kárdex siguen funcionando sin consumo automático de insumos." action={<Button icon="plus" onClick={()=>setDraft(empty())}>Nueva receta</Button>}/>
  <section className="panel standardized-management">
   <div className="toolbar"><label className="ds-input-shell"><Icon name="search" size={17}/><Input value={q} onChange={e=>setQ(e.target.value)} placeholder="Buscar producto con receta..."/></label></div>
   {recipes.isLoading?<p>Cargando recetas…</p>:!recipes.data?.items.length?<div className="catalog-state"><Icon name="chefHat" size={24}/><b>Aún no hay recetas configuradas</b><p>El inventario puede operar normalmente sin recetas. Configúralas cuando quieras activar control de insumos.</p></div>:<div className="table-wrap"><table><thead><tr><th>PRODUCTO</th><th>RENDIMIENTO</th><th>INSUMOS</th><th>ESTADO</th><th>ACCIÓN</th></tr></thead><tbody>{recipes.data.items.map((r,i)=><tr className={i%2?"alternate":""} key={r.id}><td><b>{r.productName}</b></td><td>{r.yieldQuantity}</td><td>{r.itemCount}</td><td><Status tone={r.active?"green":"gray"}>{r.active?"Activa":"Inactiva"}</Status></td><td><Button kind="secondary" disabled={loadingId===r.productId} onClick={()=>void edit(r.productId)}>{loadingId===r.productId?"Cargando…":"Editar"}</Button></td></tr>)}</tbody></table></div>}
  </section>
  {draft&&<div className="modal-backdrop modal-overlay-in"><section className="crud-modal modal-panel-in" role="dialog" aria-modal="true">
   <header><span className="modal-title-icon"><Icon name="chefHat" size={18}/></span><div><small>RECETA OPCIONAL</small><h2>Configurar consumo</h2></div><button onClick={()=>setDraft(null)} aria-label="Cerrar"><Icon name="close"/></button></header>
   <div className="form-grid">
    <label className="span-2">Producto<Select value={draft.productId} onChange={e=>setDraft({...draft,productId:e.target.value})}><option value="">Selecciona producto</option>{availableProducts.map(p=><option key={p.id} value={p.id}>{p.name}</option>)}</Select></label>
    <label>Rendimiento<Input type="number" min="0.001" step="0.001" value={draft.yieldQuantity} onChange={e=>setDraft({...draft,yieldQuantity:e.target.value})}/></label>
    <label>Estado<Select value={draft.active?"active":"inactive"} onChange={e=>setDraft({...draft,active:e.target.value==="active"})}><option value="active">Activa</option><option value="inactive">Inactiva</option></Select></label>
    <label className="span-2">Notas<Textarea rows={2} maxLength={500} value={draft.notes} onChange={e=>setDraft({...draft,notes:e.target.value})}/></label>
   </div>
   <div className="table-wrap"><table><thead><tr><th>INSUMO</th><th>CANTIDAD</th><th>MERMA %</th><th></th></tr></thead><tbody>{draft.items.map((item,index)=><tr key={index}><td><Select value={item.inventoryItemId} onChange={e=>setDraft({...draft,items:draft.items.map((x,i)=>i===index?{...x,inventoryItemId:e.target.value}:x)})}>{inventory.data?.items.map(i=><option key={i.id} value={i.id}>{i.name} · {i.unit}</option>)}</Select></td><td><Input type="number" min="0.001" step="0.001" value={item.quantity} onChange={e=>setDraft({...draft,items:draft.items.map((x,i)=>i===index?{...x,quantity:e.target.value}:x)})}/></td><td><Input type="number" min="0" max="99.9999" step="0.01" value={item.wastePercent} onChange={e=>setDraft({...draft,items:draft.items.map((x,i)=>i===index?{...x,wastePercent:e.target.value}:x)})}/></td><td><Button kind="ghost" onClick={()=>setDraft({...draft,items:draft.items.filter((_,i)=>i!==index)})}>Quitar</Button></td></tr>)}</tbody></table></div>
   <footer><Button kind="secondary" icon="plus" onClick={addIngredient}>Agregar insumo</Button><span/><Button kind="ghost" onClick={()=>setDraft(null)}>Cancelar</Button><Button disabled={save.isPending} onClick={submit}>{save.isPending?"Guardando…":"Guardar receta"}</Button></footer>
  </section></div>}
 </>;
}
