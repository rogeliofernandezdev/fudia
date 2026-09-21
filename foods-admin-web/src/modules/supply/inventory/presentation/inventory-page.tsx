"use client";
import "./inventory.css";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,Icon,Input,PageHeader,Pagination,RemoteModalSkeleton,Status} from "@/design-system";
import {useFeedback} from "@/providers";
import {useSession} from "@/providers/session-context";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import {InventoryAdjustmentDialog} from "./inventory-adjustment-dialog";
import {createInventoryAdjustment,listInventory,listInventoryProducts} from "../infrastructure/inventory-api";

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
 const[page,setPage]=useState(1);
 const[size,setSize]=useState(10);
 const[adjustmentOpen,setAdjustmentOpen]=useState(false);

 const inventory=useQuery({queryKey:["inventory",search,page,size],queryFn:()=>listInventory({q:search,page,pageSize:size})});
 const products=useQuery({queryKey:["inventory-products"],queryFn:()=>listInventoryProducts(),enabled:adjustmentOpen,staleTime:10000});
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
   action={can("inventory.manage")?<Button icon="edit" disabled={noInventory} onClick={()=>setAdjustmentOpen(true)}>Registrar ajuste</Button>:undefined}
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
     <thead><tr><th>ARTÍCULO</th><th>TIPO</th><th>UNIDAD</th><th>EXISTENCIA</th><th>STOCK MÍNIMO</th><th>ESTADO</th></tr></thead>
     <tbody>{items.map((item,index)=>{
      const meta=statusMeta[item.status];
      return <tr className={index%2?"alternate":""} key={item.inventoryItemId}>
       <td className="inventory-product-cell"><span className={"row-icon r"+index%3}><Icon name="stock" size={18}/></span><b>{item.name}</b></td>
       <td>{item.kind==="ingredient"?"Insumo":"Producto"}</td>
       <td>{item.unit}</td>
       <td><b className="inventory-quantity">{formatRegionalNumber(Number(item.quantity),location?.country,{maximumFractionDigits:3})}</b></td>
       <td>{formatRegionalNumber(Number(item.minimumStock),location?.country,{maximumFractionDigits:3})}</td>
       <td><Status tone={meta.tone}>{meta.label}</Status></td>
      </tr>;
     })}</tbody>
    </table>
   </div>}
   {!inventory.isLoading&&!inventory.isError&&<Pagination page={page} size={size} total={inventory.data?.total??0} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>}
  </section>

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
 return <div className="inventory-table-wrap"><table aria-label="Cargando inventario"><thead><tr><th>ARTÍCULO</th><th>TIPO</th><th>UNIDAD</th><th>EXISTENCIA</th><th>STOCK MÍNIMO</th><th>ESTADO</th></tr></thead><tbody>{Array.from({length:6},(_,index)=><tr className="inventory-skeleton" key={index}>{Array.from({length:6},(_,cell)=><td key={cell}><i/></td>)}</tr>)}</tbody></table></div>;
}
