"use client";
import {DataTable} from "@/design-system/data-table";
import {Dialog} from "@/design-system/dialog";
import dynamic from "next/dynamic";
import "./inventory.css";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,EmptyState,ErrorState,Icon,InlineNotice,PageHeader,Pagination,RemoteModalSkeleton,RowActionButton,SearchField,Select,Skeleton,Status,TableSkeleton,ModalHeader} from "@/design-system";
import {useFeedback} from "@/providers";
import {useSession} from "@/providers/session-context";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import {useDebouncedValue} from "@/shared/hooks/use-debounced-value";
import {createInventoryAdjustment,createInventoryTransfer,deleteInventoryItem,getInventoryItem,listInventory,listInventoryUnitOptions,listTransferLocations,removeInventoryPresentation,setInventoryItemActive,updateInventorySettings} from "../infrastructure/inventory-api";
import type {InventoryItem,InventoryItemDetail,InventoryStatusFilter} from "../domain/types";

const InventoryAdjustmentDialog=dynamic(()=>import("./inventory-adjustment-dialog").then(module=>module.InventoryAdjustmentDialog),{ssr:false});
const InventorySettingsDialog=dynamic(()=>import("./inventory-settings-dialog").then(module=>module.InventorySettingsDialog),{ssr:false});
const InventoryTransferDialog=dynamic(()=>import("./inventory-transfer-dialog").then(module=>module.InventoryTransferDialog),{ssr:false});

const inventoryColumns=["Artículo","Tipo","Unidad","Existencia","Stock mínimo","Costo prom.","Valor","Estado","Acciones"];
const statusMeta={
 ok:{label:"Saludable",tone:"green" as const},
 low:{label:"Stock bajo",tone:"orange" as const},
 out:{label:"Sin stock",tone:"gray" as const},
};
// Un cambio de ficha o de estado se ve en Compras, Recetas, Carta y Kárdex.
const inventoryQueries=["inventory","inventory-item","inventory-products","inventory-movements","purchase-inventory","purchase-item-picker","purchase-assignable-items","purchase-linkable-products","products","product-availability","order-catalog","recipes","recipe-inventory-options"];

export function InventoryPage(){
 const client=useQueryClient();
 const{notify}=useFeedback();
 const{can,location}=useSession();
 const canManage=can("inventory.manage");
 const[search,setSearch]=useState("");
 const debouncedSearch=useDebouncedValue(search);
 const[status,setStatus]=useState<InventoryStatusFilter>("active");
 const[page,setPage]=useState(1);
 const[size,setSize]=useState(10);
 const[adjustmentOpen,setAdjustmentOpen]=useState(false);
 const[settingsItem,setSettingsItem]=useState<InventoryItem|null>(null);
 const[statusTarget,setStatusTarget]=useState<InventoryItem|null>(null);
 const[deleteTarget,setDeleteTarget]=useState<InventoryItemDetail|null>(null);
 const[transferOpen,setTransferOpen]=useState(false);
 const refresh=()=>{for(const key of inventoryQueries)void client.invalidateQueries({queryKey:[key]})};

 const inventory=useQuery({queryKey:["inventory",debouncedSearch,status,page,size],queryFn:()=>listInventory({q:debouncedSearch,status,page,pageSize:size})});
 const locations=useQuery({queryKey:["transfer-locations"],queryFn:listTransferLocations,enabled:transferOpen,staleTime:30000});
 const detail=useQuery({queryKey:["inventory-item",settingsItem?.inventoryItemId],queryFn:()=>getInventoryItem(settingsItem?.inventoryItemId??""),enabled:Boolean(settingsItem)});
 const units=useQuery({queryKey:["inventory-unit-options"],queryFn:listInventoryUnitOptions,enabled:Boolean(settingsItem),staleTime:60000});
 // Desactivar revisa antes las reglas del local para explicar qué falta resolver.
 const statusDetail=useQuery({queryKey:["inventory-item",statusTarget?.inventoryItemId],queryFn:()=>getInventoryItem(statusTarget?.inventoryItemId??""),enabled:Boolean(statusTarget?.active)});
 const settingsSave=useMutation({
  mutationFn:updateInventorySettings,
  onSuccess:()=>{setSettingsItem(null);refresh();notify({tone:"success",title:"Artículo actualizado",message:"La ficha y los niveles de stock quedaron guardados."})},
  onError:error=>{if("code" in error&&error.code==="inventory_item_name_taken")return;notify({tone:"danger",title:"No se pudo guardar el artículo",message:error.message})},
 });
 const presentationRemoval=useMutation({
  mutationFn:(presentationId:string)=>removeInventoryPresentation(settingsItem?.inventoryItemId??"",presentationId),
  onSuccess:result=>{refresh();notify({tone:"success",title:"Presentación quitada",message:result.result==="hidden"?"Ya no se ofrecerá en nuevas compras; las compras anteriores la conservan.":"La presentación se eliminó."})},
  onError:error=>notify({tone:"danger",title:"No se pudo quitar la presentación",message:error.message}),
 });
 const statusSave=useMutation({
  mutationFn:(item:InventoryItem)=>setInventoryItemActive(item.inventoryItemId,!item.active),
  onSuccess:(_,item)=>{setStatusTarget(null);refresh();notify({tone:"success",title:item.active?"Artículo desactivado":"Artículo activado",message:item.active?item.name+" ya no aparece en este local. Su kárdex se conserva.":item.name+" vuelve a la lista de este local."})},
  onError:error=>notify({tone:"danger",title:"No se pudo cambiar el estado",message:error.message}),
 });
 const removal=useMutation({
  mutationFn:(item:InventoryItemDetail)=>deleteInventoryItem(item.inventoryItemId),
  onSuccess:(_,item)=>{setDeleteTarget(null);setSettingsItem(null);refresh();notify({tone:"success",title:"Artículo eliminado",message:item.name+" se eliminó de forma permanente."})},
  onError:error=>notify({tone:"danger",title:"No se pudo eliminar el artículo",message:error.message}),
 });
 const transferSave=useMutation({
  mutationFn:createInventoryTransfer,
  onSuccess:result=>{setTransferOpen(false);void client.invalidateQueries({queryKey:["inventory"]});void client.invalidateQueries({queryKey:["inventory-movements"]});notify({tone:"success",title:"Transferencia registrada",message:"El stock se movió a "+result.toLocationName+"."})},
  onError:error=>notify({tone:"danger",title:"No se pudo transferir",message:error.message}),
 });

 const save=useMutation({
  mutationFn:createInventoryAdjustment,
  onSuccess:result=>{
   setAdjustmentOpen(false);
   void client.invalidateQueries({queryKey:["inventory"]});
   void client.invalidateQueries({queryKey:["inventory-movements"]});
   const sign=result.movementType==="entry"?"+":"-";
   notify({
    tone:"success",
    title:"Ajuste registrado",
    message:result.name+": "+sign+formatRegionalNumber(result.quantity,location?.country,{maximumFractionDigits:6})+" "+result.unit+". Saldo "+formatRegionalNumber(result.stockAfter,location?.country,{maximumFractionDigits:6})+" "+result.unit+".",
   });
  },
  onError:error=>notify({tone:"danger",title:"No se pudo registrar el ajuste",message:error.message}),
 });
 const items=inventory.data?.items??[];
 const filtered=Boolean(search)||status!=="active";
 const noInventory=!inventory.isLoading&&!inventory.isError&&!filtered&&(inventory.data?.total??0)===0;
 const blockers=statusDetail.data?.deactivateBlockers??[];
 const closeSettings=()=>{setSettingsItem(null);presentationRemoval.reset()};

 return <>
  <PageHeader
   title="Inventario"
   description="Consulta existencias del local y registra ajustes manuales con trazabilidad."
   action={(canManage||can("inventory.transfer"))?<>{can("inventory.transfer")&&<Button kind="secondary" icon="truck" disabled={noInventory} onClick={()=>setTransferOpen(true)}>Transferir</Button>}{canManage&&<Button icon="edit" disabled={noInventory} onClick={()=>setAdjustmentOpen(true)}>Registrar ajuste</Button>}</>:undefined}
  />
  <section className="panel standardized-management inventory-panel">
   <div className="inventory-toolbar">
    <SearchField value={search} onChange={event=>{setSearch(event.target.value);setPage(1)}} placeholder="Buscar artículo..."/>
    <Select chip aria-label="Filtrar por estado" value={status} onChange={event=>{setStatus(event.target.value as InventoryStatusFilter);setPage(1)}}><option value="active">Activos</option><option value="inactive">Inactivos</option><option value="all">Todos los estados</option></Select>
   </div>
   {inventory.isLoading?<TableSkeleton columns={inventoryColumns} label="Cargando inventario"/>:inventory.isError?
    <ErrorState title="No pudimos cargar el inventario" text={inventory.error.message} onRetry={()=>void inventory.refetch()}/>
   :!items.length?
    <EmptyState icon={search?"search":"stock"} title={search?"Sin coincidencias":status==="inactive"?"Sin artículos inactivos":"Aún no hay artículos"} text={search?"Prueba con otro nombre.":status==="inactive"?"Los artículos que desactives en este local aparecerán aquí.":"Inventario solo permite consultar y ajustar artículos existentes."}/>
   :<DataTable label="Inventario del local" containerClassName="hover-scroll inventory-table-wrap">
     <thead><tr>{inventoryColumns.map(column=><th key={column}>{column}</th>)}</tr></thead>
     <tbody>{items.map((item,index)=>{
      const meta=statusMeta[item.status];
      return <tr className={index%2?"alternate":""} key={item.inventoryItemId}>
       <td className="inventory-product-cell"><span className={"row-icon r"+index%3}><Icon name="stock" size={18}/></span><b>{item.name}</b></td>
       <td>{item.kind==="ingredient"?"Insumo":"Producto"}</td>
       <td>{item.unit}</td>
       <td><b className="inventory-quantity">{formatRegionalNumber(Number(item.quantity),location?.country,{maximumFractionDigits:6})}</b></td>
       <td><b>{formatRegionalNumber(Number(item.minimumStock),location?.country,{maximumFractionDigits:6})}</b></td>
       <td>{formatRegionalNumber(Number(item.averageUnitCost),location?.country,{minimumFractionDigits:2,maximumFractionDigits:4})}</td>
       <td><b>{formatRegionalNumber(Number(item.stockValue),location?.country,{minimumFractionDigits:2,maximumFractionDigits:2})}</b></td>
       <td>{item.active?<Status tone={meta.tone}>{meta.label}</Status>:<Status active={false}>Inactivo</Status>}</td>
       <td>{canManage?<div className="table-actions"><RowActionButton action="edit" label="Editar artículo" onClick={()=>setSettingsItem(item)}/><RowActionButton action={item.active?"deactivate":"activate"} stateLabel={`Estado de ${item.name} en este local`} onClick={()=>setStatusTarget(item)}/></div>:null}</td>
      </tr>;
     })}</tbody>
    </DataTable>}
   {!inventory.isLoading&&!inventory.isError&&<Pagination page={page} size={size} total={inventory.data?.total??0} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>}
  </section>

  {settingsItem&&(detail.isLoading||units.isLoading?
   <RemoteModalSkeleton className="crud-modal compact" label="Cargando artículo" rows={4} title="Editar artículo" close={closeSettings}/>
   :detail.isError||units.isError?
   <LoadError title="No pudimos cargar el artículo" stateTitle="Artículo no disponible" message={(detail.error??units.error)?.message??""} close={closeSettings} retry={()=>{void detail.refetch();void units.refetch()}}/>
   :detail.data&&<InventorySettingsDialog key={detail.data.inventoryItemId} item={detail.data} units={units.data?.items??[]} busy={settingsSave.isPending||removal.isPending} close={closeSettings} remove={()=>setDeleteTarget(detail.data)} removePresentation={id=>presentationRemoval.mutate(id)} removingPresentation={presentationRemoval.isPending?presentationRemoval.variables??"":""} save={async draft=>{try{await settingsSave.mutateAsync(draft);}catch(error){if(error&&typeof error==="object"&&"code" in error&&error.code==="inventory_item_name_taken")throw error;}}}/>
  )}
  {transferOpen&&(locations.isLoading?
   <RemoteModalSkeleton className="inventory-adjustment-modal" label="Cargando transferencia" rows={5} title="Transferir a otro local" close={()=>setTransferOpen(false)}/>
   :locations.isError?
   <LoadError title="No pudimos cargar los locales" stateTitle="Locales no disponibles" message={locations.error.message} close={()=>setTransferOpen(false)} retry={()=>void locations.refetch()}/>
   :<InventoryTransferDialog locations={locations.data?.items??[]} currentLocationId={location?.id??""} busy={transferSave.isPending} close={()=>setTransferOpen(false)} save={draft=>transferSave.mutate(draft)}/>
  )}

  {adjustmentOpen&&<InventoryAdjustmentDialog busy={save.isPending} close={()=>setAdjustmentOpen(false)} save={draft=>save.mutate(draft)}/>}

  <ConfirmDialog
   open={Boolean(statusTarget)}
   title={statusTarget?.active?"Desactivar artículo":"Activar artículo"}
   subject={statusTarget?{label:"Artículo",name:statusTarget.name}:undefined}
   description={statusTarget?.active?"Dejará de aparecer en Inventario y Compras de este local. Su kárdex se conserva y los demás locales no cambian.":"Volverá a la lista de este local con su historial."}
   tone={statusTarget?.active?"danger":"success"}
   confirmLabel={statusTarget?.active?"Desactivar":"Activar"}
   pending={statusSave.isPending}
   confirmDisabled={Boolean(statusTarget?.active&&(statusDetail.isLoading||statusDetail.isError||blockers.length>0))}
   onCancel={()=>setStatusTarget(null)}
   onConfirm={()=>statusTarget&&statusSave.mutate(statusTarget)}
  >
   {statusTarget?.active&&(statusDetail.isLoading?<Skeleton/>
   :statusDetail.isError?<InlineNotice tone="danger" title="No pudimos revisar el artículo">{statusDetail.error.message}</InlineNotice>
   :blockers.length>0?<InlineNotice tone="danger" title="Antes de desactivarlo"><>{blockers.map(blocker=><span className="inventory-blocker" key={blocker}>{blocker}</span>)}</></InlineNotice>
   :statusDetail.data?.productName?<p className="confirm-note">{statusDetail.data.productName} seguirá en la Carta y se mostrará agotado en este local.</p>:null)}
  </ConfirmDialog>
  <ConfirmDialog
   open={Boolean(deleteTarget)}
   title="Eliminar artículo"
   subject={deleteTarget?{label:"Artículo",name:deleteTarget.name}:undefined}
   description="Nunca tuvo compras, movimientos ni recetas. Se eliminará de forma permanente."
   note={deleteTarget?.productName?`${deleteTarget.productName} seguirá en la Carta, sin control de stock.`:undefined}
   confirmLabel="Eliminar"
   pending={removal.isPending}
   onCancel={()=>setDeleteTarget(null)}
   onConfirm={()=>deleteTarget&&removal.mutate(deleteTarget)}
  />
 </>;
}

function LoadError({title,stateTitle,message,close,retry}:{title:string;stateTitle:string;message:string;close:()=>void;retry:()=>void}){
 return <div className="modal-backdrop modal-overlay-in" role="presentation"><Dialog onResponseClose={close} className="crud-modal inventory-adjustment-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="inventory-load-error-title">

  <ModalHeader title={title} titleId="inventory-load-error-title" onClose={close} closeLabel="Cerrar"/>
  <ErrorState title={stateTitle} text={message} onRetry={retry}/>
 </Dialog></div>;
}
