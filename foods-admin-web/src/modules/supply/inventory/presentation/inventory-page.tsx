"use client";
import dynamic from "next/dynamic";
import "./inventory.css";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,Icon,Input,PageHeader,Pagination,RemoteModalSkeleton,RowActionButton,Status} from "@/design-system";
import {useFeedback} from "@/providers";
import {useSession} from "@/providers/session-context";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import {useDebouncedValue} from "@/shared/hooks/use-debounced-value";
const InventoryAdjustmentDialog=dynamic(()=>import("./inventory-adjustment-dialog").then(module=>module.InventoryAdjustmentDialog),{ssr:false});
const InventorySettingsDialog=dynamic(()=>import("./inventory-settings-dialog").then(module=>module.InventorySettingsDialog),{ssr:false});
const InventoryTransferDialog=dynamic(()=>import("./inventory-transfer-dialog").then(module=>module.InventoryTransferDialog),{ssr:false});
import {createInventoryAdjustment,createInventoryTransfer,listInventory,listInventoryProducts,listTransferLocations,updateInventorySettings} from "../infrastructure/inventory-api";
import type {InventoryItem} from "../domain/types";

const statusMeta={
 ok:{label:"Saludable",tone:"green" as const},
 low:{label:"Stock bajo",tone:"orange" as const},
 out:{label:"Sin stock",tone:"gray" as const},
};

export function InventoryPage(){
 const client=useQueryClient();
 const{notify}=useFeedback();
 const{can,location}=useSession();
 const[search,setSearch]=useState("");
 const debouncedSearch=useDebouncedValue(search);
 const[page,setPage]=useState(1);
 const[size,setSize]=useState(10);
 const[adjustmentOpen,setAdjustmentOpen]=useState(false);
 const[settingsItem,setSettingsItem]=useState<InventoryItem|null>(null);
 const[transferOpen,setTransferOpen]=useState(false);

 const inventory=useQuery({queryKey:["inventory",debouncedSearch,page,size],queryFn:()=>listInventory({q:debouncedSearch,page,pageSize:size})});
 const products=useQuery({queryKey:["inventory-products"],queryFn:()=>listInventoryProducts(),enabled:adjustmentOpen||transferOpen,staleTime:10000});
 const locations=useQuery({queryKey:["transfer-locations"],queryFn:listTransferLocations,enabled:transferOpen,staleTime:30000});
 const settingsSave=useMutation({
  mutationFn:updateInventorySettings,
  onSuccess:()=>{setSettingsItem(null);void client.invalidateQueries({queryKey:["inventory"]});void client.invalidateQueries({queryKey:["inventory-products"]});notify({tone:"success",title:"Niveles actualizados",message:"Los niveles de reposición quedaron configurados para este local."})},
  onError:error=>notify({tone:"danger",title:"No se pudieron guardar los niveles",message:error.message}),
 });
 const transferSave=useMutation({
  mutationFn:createInventoryTransfer,
  onSuccess:result=>{setTransferOpen(false);void client.invalidateQueries({queryKey:["inventory"]});void client.invalidateQueries({queryKey:["inventory-products"]});void client.invalidateQueries({queryKey:["inventory-movements"]});notify({tone:"success",title:"Transferencia registrada",message:result.code+" movió el stock al local destino."})},
  onError:error=>notify({tone:"danger",title:"No se pudo transferir",message:error.message}),
 });

 const save=useMutation({
  mutationFn:createInventoryAdjustment,
  onSuccess:result=>{
   setAdjustmentOpen(false);
   void client.invalidateQueries({queryKey:["inventory"]});
   void client.invalidateQueries({queryKey:["inventory-products"]});
   void client.invalidateQueries({queryKey:["inventory-movements"]});
   const sign=result.movementType==="entry"?"+":"-";
   notify({
    tone:"success",
    title:"Ajuste registrado",
    message:result.name+": "+sign+formatRegionalNumber(result.quantity,location?.country,{maximumFractionDigits:3})+" "+result.unit+". Saldo "+formatRegionalNumber(result.stockAfter,location?.country,{maximumFractionDigits:3})+" "+result.unit+".",
   });
  },
  onError:error=>notify({tone:"danger",title:"No se pudo registrar el ajuste",message:error.message}),
 });
 const items=inventory.data?.items??[];
 const noInventory=!inventory.isLoading&&!inventory.isError&&(inventory.data?.total??0)===0;

 return <>
  <PageHeader
   eyebrow="ABASTECIMIENTO"
   title="Inventario"
   description="Consulta existencias del local y registra ajustes manuales con trazabilidad."
   action={(can("inventory.manage")||can("inventory.transfer"))?<div style={{display:"flex",gap:8}}>{can("inventory.transfer")&&<Button kind="secondary" icon="truck" disabled={noInventory} onClick={()=>setTransferOpen(true)}>Transferir</Button>}{can("inventory.manage")&&<Button icon="edit" disabled={noInventory} onClick={()=>setAdjustmentOpen(true)}>Registrar ajuste</Button>}</div>:undefined}
  />
  <section className="panel standardized-management inventory-panel">
   <div className="inventory-toolbar">
    <label className="ds-input-shell"><Icon name="search" size={18}/><Input value={search} onChange={event=>{setSearch(event.target.value);setPage(1)}} placeholder="Buscar artículo..."/></label>
    <p><Icon name="store" size={15}/>Existencia física del local activo en su unidad base.</p>
   </div>
   {inventory.isLoading?<InventorySkeleton/>:inventory.isError?
    <div className="inventory-state"><Icon name="alert" size={24}/><b>No pudimos cargar el inventario</b><p>{inventory.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>inventory.refetch()}>Reintentar</Button></div>
   :!items.length?
    <div className="inventory-state"><Icon name="stock" size={24}/><b>{search?"Sin coincidencias":"No hay artículos de inventario"}</b><p>{search?"Prueba con otro nombre.":"Inventario solo permite consultar y ajustar artículos existentes."}</p></div>
   :<div className="table-wrap hover-scroll inventory-table-wrap">
    <table>
     <thead><tr><th>ARTÍCULO</th><th>TIPO</th><th>UNIDAD</th><th>EXISTENCIA</th><th>MÍNIMO / REORDEN</th><th>COSTO PROM.</th><th>VALOR</th><th>ESTADO</th><th>ACCIÓN</th></tr></thead>
     <tbody>{items.map((item,index)=>{
      const meta=statusMeta[item.status];
      return <tr className={index%2?"alternate":""} key={item.inventoryItemId}>
       <td className="inventory-product-cell"><span className={"row-icon r"+index%3}><Icon name="stock" size={18}/></span><b>{item.name}</b></td>
       <td>{item.kind==="ingredient"?"Insumo":"Producto"}</td>
       <td>{item.unit}</td>
       <td><b className="inventory-quantity">{formatRegionalNumber(Number(item.quantity),location?.country,{maximumFractionDigits:3})}</b></td>
       <td><b>{formatRegionalNumber(Number(item.minimumStock),location?.country,{maximumFractionDigits:3})}</b><small> / {formatRegionalNumber(Number(item.reorderPoint),location?.country,{maximumFractionDigits:3})}</small></td>
       <td>{formatRegionalNumber(Number(item.averageUnitCost),location?.country,{minimumFractionDigits:2,maximumFractionDigits:4})}</td>
       <td><b>{formatRegionalNumber(Number(item.stockValue),location?.country,{minimumFractionDigits:2,maximumFractionDigits:2})}</b></td>
       <td><Status tone={meta.tone}>{meta.label}</Status></td>
       <td>{can("inventory.manage")?<RowActionButton action="edit" onClick={()=>setSettingsItem(item)}/>:null}</td>
      </tr>;
     })}</tbody>
    </table>
   </div>}
   {!inventory.isLoading&&!inventory.isError&&<Pagination page={page} size={size} total={inventory.data?.total??0} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>}
  </section>

  {settingsItem&&<InventorySettingsDialog item={settingsItem} busy={settingsSave.isPending} close={()=>setSettingsItem(null)} save={draft=>settingsSave.mutate(draft)}/>}
  {transferOpen&&(products.isLoading||locations.isLoading?
   <RemoteModalSkeleton className="inventory-adjustment-modal" label="Cargando transferencia" rows={5} close={()=>setTransferOpen(false)}/>
   :products.isError||locations.isError?
   <InventoryAdjustmentLoadError message={(products.error??locations.error)?.message??"No se pudo preparar la transferencia."} close={()=>setTransferOpen(false)} retry={()=>{void products.refetch();void locations.refetch()}}/>
   :<InventoryTransferDialog items={products.data?.items??[]} locations={locations.data?.items??[]} currentLocationId={location?.id??""} busy={transferSave.isPending} close={()=>setTransferOpen(false)} save={draft=>transferSave.mutate(draft)}/>
  )}

  {adjustmentOpen&&(products.isLoading?
   <RemoteModalSkeleton className="inventory-adjustment-modal" label="Cargando artículos de inventario" rows={5} close={()=>setAdjustmentOpen(false)}/>
   :products.isError?
   <InventoryAdjustmentLoadError message={products.error.message} close={()=>setAdjustmentOpen(false)} retry={()=>products.refetch()}/>
   :<InventoryAdjustmentDialog items={products.data?.items??[]} busy={save.isPending} close={()=>setAdjustmentOpen(false)} save={draft=>save.mutate(draft)}/>
  )}
 </>;
}

function InventoryAdjustmentLoadError({message,close,retry}:{message:string;close:()=>void;retry:()=>void}){
 return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal inventory-adjustment-modal modal-panel-in" role="dialog" aria-modal="true">
  <div className="modal-accent"/>
  <header><span className="modal-title-icon"><Icon name="alert" size={18}/></span><div><small>INVENTARIO</small><h2>No pudimos cargar los artículos</h2></div><button type="button" aria-label="Cerrar" onClick={close}><Icon name="close"/></button></header>
  <div className="inventory-state"><p>{message}</p><Button kind="secondary" icon="refresh" onClick={retry}>Reintentar</Button></div>
 </section></div>;
}

function InventorySkeleton(){
 return <div className="inventory-table-wrap"><table aria-label="Cargando inventario"><thead><tr><th>ARTÍCULO</th><th>TIPO</th><th>UNIDAD</th><th>EXISTENCIA</th><th>MÍNIMO / REORDEN</th><th>COSTO PROM.</th><th>VALOR</th><th>ESTADO</th><th>ACCIÓN</th></tr></thead><tbody>{Array.from({length:6},(_,index)=><tr className="inventory-skeleton" key={index}>{Array.from({length:9},(_,cell)=><td key={cell}><i/></td>)}</tr>)}</tbody></table></div>;
}
