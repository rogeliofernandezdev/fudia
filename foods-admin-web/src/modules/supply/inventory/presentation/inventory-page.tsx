"use client";
import "./inventory.css";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,Icon,Input,PageHeader,Pagination,RemoteModalSkeleton,Status} from "@/design-system";
import {useFeedback,useSettings} from "@/providers";
import {useSession} from "@/providers/session-context";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import {InventoryEntryDialog} from "./inventory-entry-dialog";
import {createInventoryEntry,listInventory,listInventoryProducts} from "../infrastructure/inventory-api";

const statusMeta={
 ok:{label:"Saludable",tone:"green" as const},
 low:{label:"Stock bajo",tone:"orange" as const},
 out:{label:"Sin stock",tone:"gray" as const},
};

export function InventoryPage(){
 const client=useQueryClient();
 const{notify}=useFeedback();
 const settings=useSettings();
 const{can,location}=useSession();
 const[search,setSearch]=useState("");
 const[page,setPage]=useState(1);
 const[size,setSize]=useState(10);
 const[entryOpen,setEntryOpen]=useState(false);

 const inventory=useQuery({queryKey:["inventory",search,page,size],queryFn:()=>listInventory({q:search,page,pageSize:size})});
 const products=useQuery({queryKey:["inventory-products"],queryFn:()=>listInventoryProducts(),enabled:entryOpen,staleTime:30000});
 const save=useMutation({
  mutationFn:createInventoryEntry,
  onSuccess:result=>{
   setEntryOpen(false);
   void client.invalidateQueries({queryKey:["inventory"]});
   void client.invalidateQueries({queryKey:["inventory-products"]});
   void client.invalidateQueries({queryKey:["products"]});
   void client.invalidateQueries({queryKey:["product-availability"]});
   void client.invalidateQueries({queryKey:["inventory-movements"]});
   notify({tone:"success",title:result.createdProduct?"Producto y entrada registrados":"Entrada registrada",message:`${result.name}: +${formatRegionalNumber(Number(result.stockQuantity),location?.country,{maximumFractionDigits:3})} ${result.unit}. Saldo ${formatRegionalNumber(Number(result.balance),location?.country,{maximumFractionDigits:3})} ${result.unit}.`});
  },
  onError:error=>notify({tone:"danger",title:"No se pudo registrar la entrada",message:error.message}),
 });
 const items=inventory.data?.items??[];

 return <>
  <PageHeader eyebrow="ABASTECIMIENTO" title="Inventario" description="Consulta la existencia física de los productos del local y registra cada ingreso de mercadería." action={can("inventory.manage")?<Button icon="plus" onClick={()=>setEntryOpen(true)}>Nueva entrada</Button>:undefined}/>
  <section className="panel standardized-management inventory-panel">
   <div className="inventory-toolbar">
    <label className="ds-input-shell"><Icon name="search" size={18}/><Input value={search} onChange={event=>{setSearch(event.target.value);setPage(1)}} placeholder="Buscar producto..."/></label>
    <p><Icon name="store" size={15}/>Existencia física del local activo. Los platos por porciones no aparecen aquí.</p>
   </div>
   {inventory.isLoading?<InventorySkeleton/>:inventory.isError?
    <div className="inventory-state"><Icon name="alert" size={24}/><b>No pudimos cargar el inventario</b><p>{inventory.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>inventory.refetch()}>Reintentar</Button></div>
   :!items.length?
    <div className="inventory-state"><Icon name="stock" size={24}/><b>{search?"Sin coincidencias":"Aún no hay productos con inventario"}</b><p>{search?"Prueba con otro nombre.":"Usa Nueva entrada para recibir un producto existente o crear una mercadería nueva."}</p>{can("inventory.manage")&&!search&&<Button icon="plus" onClick={()=>setEntryOpen(true)}>Nueva entrada</Button>}</div>
   :<div className="table-wrap hover-scroll inventory-table-wrap">
    <table>
     <thead><tr><th>PRODUCTO</th><th>UNIDAD</th><th>EXISTENCIA</th><th>STOCK MÍNIMO</th><th>ESTADO</th></tr></thead>
     <tbody>{items.map((item,index)=>{
      const meta=statusMeta[item.status];
      return <tr className={index%2?"alternate":""} key={item.productId}>
       <td className="inventory-product-cell"><span className={`row-icon r${index%3}`}><Icon name="stock" size={18}/></span><b>{item.name}</b></td>
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

  {entryOpen&&(products.isLoading?<RemoteModalSkeleton className="inventory-entry-modal" label="Cargando productos para inventario" rows={6} close={()=>setEntryOpen(false)}/>:<InventoryEntryDialog products={products.data?.items??[]} currencySymbol={settings.currencySymbol} busy={save.isPending} close={()=>setEntryOpen(false)} save={draft=>save.mutate(draft)}/>)}
 </>;
}

function InventorySkeleton(){
 return <div className="inventory-table-wrap"><table aria-label="Cargando inventario"><thead><tr><th>PRODUCTO</th><th>UNIDAD</th><th>EXISTENCIA</th><th>STOCK MÍNIMO</th><th>ESTADO</th></tr></thead><tbody>{Array.from({length:6},(_,index)=><tr className="inventory-skeleton" key={index}>{Array.from({length:5},(_,cell)=><td key={cell}><i/></td>)}</tr>)}</tbody></table></div>;
}
