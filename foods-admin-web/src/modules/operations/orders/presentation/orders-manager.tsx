"use client";
import {Dialog} from "@/design-system/dialog";
import "../../styles/orders.css";
import dynamic from "next/dynamic";
import {useState} from "react";
import {keepPreviousData,useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,DataTable,EmptyState,ErrorState,Icon,PageHeader,Pagination,RowActionButton,SearchField,Select,Skeleton,Status,TableSkeleton,Tabs,TabsBar} from "@/design-system";
import type {Order} from "../domain/types";
import {createManualOrder,getOrder,listOrders,updateOrderStatus} from "../infrastructure/orders-api";
import type {ManualOrderDraft} from "../domain/manual-order-schema";
const ManualOrderDialog=dynamic(()=>import("./manual-order-dialog").then(module=>module.ManualOrderDialog),{ssr:false});
import {useFeedback} from "@/providers/feedback-provider";
import {useSettings} from "@/providers/settings-context";
import {useMoney} from "@/providers";
import {useDebouncedValue} from "@/shared/hooks/use-debounced-value";
import {useSession} from "@/providers/session-context";
import {ComandaView} from "./comanda-view";
import type {Draft} from "../../salon/domain/types";
import {addOrderConsumption} from "../infrastructure/service-api";
import {OrderDetailContent,OrderDetailSkeleton,orderDetailTitleId} from "./order-detail-dialog";
import {channelIcons,orderStatus,registeredAt} from "./order-meta";

const columns=["PEDIDO","CANAL","ESTADO","REGISTRADO","TOTAL","ACCIONES"];

export function OrdersManager(){
 const qc=useQueryClient();const{notify}=useFeedback();const settings=useSettings();const money=useMoney();const{can,location}=useSession();const canManage=can("orders.manage");
 const[q,setQ]=useState("");const debouncedQ=useDebouncedValue(q);const[channel,setChannel]=useState("");const[status,setStatus]=useState("abiertos");const[page,setPage]=useState(1);const[size,setSize]=useState(10);
 const[detailId,setDetailId]=useState<string|null>(null);const[cancelTarget,setCancelTarget]=useState<Order|null>(null);
 const[creating,setCreating]=useState(false);
 const[adding,setAdding]=useState<{order:Order;draft:Draft;requestKey:string}|null>(null);
 const list=useQuery({queryKey:["orders",debouncedQ,channel,status,page,size],queryFn:()=>listOrders({q:debouncedQ,channel,status,page,pageSize:size}),placeholderData:keepPreviousData});
 const detail=useQuery({queryKey:["order",detailId],queryFn:()=>getOrder(detailId!),enabled:Boolean(detailId),refetchInterval:query=>query.state.data?.completedAt?false:10000});
 const invalidate=()=>Promise.all(["orders","order","salon-floor","pos-orders","pos-order","dashboard","kitchen-tickets","order-catalog","order-combo","product-availability","products","sales"].map(key=>qc.invalidateQueries({queryKey:[key]})));
 const create=useMutation({mutationFn:(draft:ManualOrderDraft)=>createManualOrder(draft),onSuccess:order=>{setCreating(false);setChannel(order.channel);setStatus("abiertos");setQ("");setPage(1);invalidate();notify({tone:"success",title:"Pedido registrado",message:"El pedido fue enviado a Cocina. Puedes seguir su avance en Pedidos y registrar el cobro en Punto de venta."})},onError:error=>{void qc.invalidateQueries({queryKey:["order-catalog"]});void qc.invalidateQueries({queryKey:["order-combo"]});notify({tone:"danger",title:"No se pudo registrar el pedido",message:error.message})}});
 const advance=useMutation({mutationFn:(v:{id:string;status:string})=>updateOrderStatus(v.id,v.status),onSuccess:order=>{const refresh=invalidate();if(order.channel!=="salon"||order.status!=="entregado")notify({tone:"success",title:"Pedido actualizado",message:"El estado del pedido fue actualizado."});return refresh},onError:e=>notify({tone:"danger",title:"No se pudo actualizar el pedido",message:e.message})});
 const cancel=useMutation({mutationFn:(o:Order)=>updateOrderStatus(o.id,"cancelado"),onSuccess:()=>{setCancelTarget(null);invalidate();notify({tone:"success",title:"Pedido cancelado",message:"El pedido quedó marcado como cancelado."})},onError:e=>notify({tone:"danger",title:"No se pudo cancelar el pedido",message:e.message})});
 const append=useMutation({mutationFn:(draft:Draft)=>addOrderConsumption(adding!.order.id,draft,adding!.requestKey),onSuccess:order=>{setAdding(null);setDetailId(order.id);invalidate()},onError:error=>notify({tone:"danger",title:"No se pudieron agregar los productos",message:error.message})});
 const addProducts=(order:Order)=>{setAdding({order,requestKey:crypto.randomUUID(),draft:{channel:"salon",tableId:order.tableId,customerName:order.customerName,customerPhone:"",address:"",reference:"",notes:"",deliveryFee:"0",lines:[]}});setDetailId(null)};
 const items=list.data?.items??[];const counts=list.data?.channelCounts??{};const channelOptions=list.data?.channelOptions??[];
 const channelLabel=(v:string)=>channelOptions.find(o=>o.value===v)?.label??v;
 const showChannelCounts=status==="abiertos";
 const openTotal=Object.values(counts).reduce((a,b)=>a+b,0);
 const tabs=[{value:"",label:"Todos",icon:"receipt" as const,count:showChannelCounts?openTotal:undefined},...channelOptions.map(o=>({value:o.value,label:o.label,icon:channelIcons[o.value]??"receipt",count:showChannelCounts?counts[o.value]??0:undefined}))];
 const pickChannel=(v:string)=>{setChannel(v);setPage(1)};
 const hasActiveFilters=Boolean(q||channel||status!=="abiertos");
 const clearFilters=()=>{setQ("");setChannel("");setStatus("abiertos");setPage(1)};
 const loading=detail.isLoading;
 const closeDetail=()=>setDetailId(null);
 return <div className="orders-page-shell"><PageHeader title="Pedidos" description="Revisa el origen, estado y avance de cada pedido sin perder el contexto operativo." />
 <TabsBar sticky action={canManage?<Button icon="plus" disabled={list.isPending||list.isError||!(list.data?.channelOptions.length)} onClick={()=>setCreating(true)}>Nuevo pedido</Button>:undefined}>
  {list.isPending?<div className="orders-tabs-skeleton" role="status" aria-label="Cargando canales">{Array.from({length:4},(_,index)=><Skeleton key={index} width="var(--size-112)" height="calc(var(--control-height) - var(--size-8))" radius="var(--radius-control)"/>)}</div>
  :<Tabs items={tabs} value={channel} onChange={pickChannel} label="Filtrar pedidos por canal"/>}
 </TabsBar>
 <section className="panel management standardized-management orders-panel" aria-busy={list.isFetching&&list.isPlaceholderData}>
  <div className="toolbar">
   <SearchField label="Buscar pedidos" value={q} onChange={e=>{setQ(e.target.value);setPage(1)}} placeholder="Buscar cliente, mesa o teléfono"/>
   <Select aria-label="Filtrar por estado" value={status} onChange={e=>{setStatus(e.target.value);setPage(1)}}><option value="">Todos los estados</option><option value="abiertos">Abiertos</option>{(list.data?.statusOptions??[]).map(o=><option key={o.value} value={o.value}>{o.label}</option>)}</Select>
   {hasActiveFilters&&<Button kind="ghost" icon="close" onClick={clearFilters}>Limpiar filtros</Button>}
  </div>
  {list.isPending?<TableSkeleton columns={columns} label="Cargando pedidos"/>
  :list.isError?<ErrorState title="No pudimos cargar los pedidos" text={list.error.message} onRetry={()=>list.refetch()} retrying={list.isFetching}/>
  :!items.length?<EmptyState icon="receipt" title={hasActiveFilters?"Sin coincidencias":"Aún no hay pedidos abiertos"} text={hasActiveFilters?"Prueba con otro canal, estado o término de búsqueda.":"Los pedidos nuevos aparecerán aquí cuando ingresen."} action={hasActiveFilters?<Button kind="secondary" icon="close" onClick={clearFilters}>Limpiar filtros</Button>:undefined}/>
  :<DataTable label="Listado de pedidos" containerClassName="hover-scroll" className="orders-table">
     <thead><tr>{columns.map(column=><th key={column}>{column}</th>)}</tr></thead>
     <tbody>{items.map((o,index)=>{const meta=orderStatus(o.status);const subject=o.tableName||o.customerName||"Pedido";const secondary=o.tableName&&o.customerName?o.customerName:undefined;return <tr className={index%2?"alternate":""} key={o.id}>
      <td>
       <span className={"row-icon order-row-icon oc-"+o.channel}><Icon name={channelIcons[o.channel]??"receipt"} size={18}/></span>
       <b>{subject}</b>
       {secondary&&<small>{secondary}</small>}
       {o.channel==="salon"&&<small className="order-staff">Mozo: {o.waiterName||"Sin asignar"}</small>}
       {Boolean(o.collectedByNames?.length)&&<small className="order-staff">Cobrado por: {o.collectedByNames!.join(" · ")}</small>}
       <small className="order-mobile-registered"><time dateTime={o.createdAt}>{registeredAt(o.createdAt,location?.country,location?.timezone)}</time></small>
      </td>
      <td><span className="order-channel-cell">{channelLabel(o.channel)}</span></td>
      <td><Status tone={meta.tone}>{meta.label}</Status></td>
      <td><time className="order-registered" dateTime={o.createdAt}>{registeredAt(o.createdAt,location?.country,location?.timezone)}</time></td>
      <td><b className="order-table-total">{money(o.total)}</b></td>
      <td><div className="orders-table-actions"><RowActionButton action="view" label={`Ver detalle de ${subject}`} onClick={()=>setDetailId(o.id)}/></div></td>
     </tr>})}</tbody>
   </DataTable>}
  <Pagination page={page} size={size} total={list.data?.total??0} onPage={setPage} onSize={v=>{setSize(v);setPage(1)}}/>
 </section>
 {creating&&canManage&&<ManualOrderDialog channels={channelOptions} currencySymbol={settings.currencySymbol} busy={create.isPending} onClose={()=>{if(!create.isPending)setCreating(false)}} onSave={async draft=>{try{await create.mutateAsync(draft)}catch{/* Feedback conserva el error y el formulario permanece abierto. */}}}/>}
 {adding&&<ComandaView mode="append" initial={adding.draft} allTables={[{id:adding.order.tableId,name:adding.order.tableName,zone:"",seats:0,order:null}]} busy={append.isPending} close={()=>setAdding(null)} save={draft=>append.mutateAsync(draft).then(()=>{}).catch(()=>{})} notify={notify}/>}
 {detailId&&<div className="modal-backdrop modal-overlay-in">
  <Dialog onResponseClose={closeDetail} className="crud-modal order-detail modal-panel-in" role="dialog" aria-modal="true" aria-labelledby={orderDetailTitleId} aria-busy={loading}>
   {loading?<OrderDetailSkeleton variant="order" close={closeDetail}/>
   :<OrderDetailContent variant="order" order={detail.data} error={detail.error?.message} retry={()=>detail.refetch()} retrying={detail.isFetching} canManage={canManage} busy={advance.isPending} close={closeDetail} advance={st=>advance.mutate({id:detailId,status:st})} add={addProducts} cancel={o=>setCancelTarget(o)}/>}
  </Dialog>
 </div>}
 <ConfirmDialog open={Boolean(cancelTarget)} title="Cancelar pedido" description={`${cancelTarget?.tableName||cancelTarget?.customerName||"El pedido seleccionado"} quedará cancelado y no podrá reactivarse.`} tone="danger" confirmLabel="Cancelar pedido" pending={cancel.isPending} onCancel={()=>setCancelTarget(null)} onConfirm={()=>cancelTarget&&cancel.mutate(cancelTarget)}/></div>;
}
