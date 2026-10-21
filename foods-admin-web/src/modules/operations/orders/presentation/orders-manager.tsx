"use client";
import {Dialog} from "@/design-system/dialog";
import "../../styles/orders.css";
import "../../styles/salon.css";
import Link from "next/link";
import dynamic from "next/dynamic";
import {useState,type KeyboardEvent} from "react";
import {keepPreviousData,useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,Icon,IconName,PageHeader,Pagination,RowActionButton,Status} from "@/design-system";
import type {Order} from "../domain/types";
import {createManualOrder,getOrder,listOrders,updateOrderStatus} from "../infrastructure/orders-api";
import type {ManualOrderDraft} from "../domain/manual-order-schema";
const ManualOrderDialog=dynamic(()=>import("./manual-order-dialog").then(module=>module.ManualOrderDialog),{ssr:false});
import {useFeedback} from "@/providers/feedback-provider";
import {useSettings} from "@/providers/settings-context";
import {useDebouncedValue} from "@/shared/hooks/use-debounced-value";
import {useSession} from "@/providers/session-context";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {pageRoutes} from "@/shared/routing/page-routes";


import {nextOrderAction,canManageOrderService,canCancelOrder} from "../domain/order-actions";
import {ComandaView} from "./comanda-view";
import type {Draft} from "../../salon/domain/types";
import {addOrderConsumption} from "../infrastructure/service-api";
import {canChargeAccount} from "../domain/service-flow";
import {OrderAccountActions,OrderItemService,accountLabel} from "./order-service-controls";
import {OrderAttribution,OrderAttributionSkeleton} from "./order-attribution";

const channelIcons:Record<string,IconName>={salon:"utensils",mostrador:"store",recojo:"box",delivery:"truck",whatsapp:"share"};
const statusMeta:Record<string,{label:string;tone:"green"|"blue"|"orange"|"gray"}>={nuevo:{label:"Nuevo",tone:"blue"},confirmado:{label:"Confirmado",tone:"blue"},preparando:{label:"Preparando",tone:"orange"},listo:{label:"Listo",tone:"green"},en_camino:{label:"En camino",tone:"orange"},entregado:{label:"Entregado",tone:"gray"},cancelado:{label:"Cancelado",tone:"gray"}};


function registeredAt(iso:string,country?:string,timeZone?:string){
  return formatRegionalDateTime(iso,{country,timeZone},{dateStyle:"medium",timeStyle:"short"});
}
const money=(v:string|number)=>Number(v).toFixed(2);

export function OrdersManager(){
 const qc=useQueryClient();const{notify}=useFeedback();const settings=useSettings();const{can,location}=useSession();const canManage=can("orders.manage");
 const[q,setQ]=useState("");const debouncedQ=useDebouncedValue(q);const[channel,setChannel]=useState("");const[status,setStatus]=useState("abiertos");const[page,setPage]=useState(1);const[size,setSize]=useState(12);
 const[detailId,setDetailId]=useState<string|null>(null);const[cancelTarget,setCancelTarget]=useState<Order|null>(null);
 const[creating,setCreating]=useState(false);
 const[adding,setAdding]=useState<{order:Order;draft:Draft;requestKey:string}|null>(null);
 const list=useQuery({queryKey:["orders",debouncedQ,channel,status,page,size],queryFn:()=>listOrders({q:debouncedQ,channel,status,page,pageSize:size}),placeholderData:keepPreviousData});
 const detail=useQuery({queryKey:["order",detailId],queryFn:()=>getOrder(detailId!),enabled:Boolean(detailId),refetchInterval:query=>query.state.data?.completedAt?false:10000});
 const invalidate=()=>Promise.all(["orders","order","salon-floor","pos-orders","pos-order","dashboard","kitchen-tickets","order-catalog","order-combo","product-availability","products","sales"].map(key=>qc.invalidateQueries({queryKey:[key]})));
 const create=useMutation({mutationFn:(draft:ManualOrderDraft)=>createManualOrder(draft),onSuccess:order=>{setCreating(false);setChannel(order.channel);setStatus("abiertos");setQ("");setPage(1);invalidate();notify({tone:"success",title:"Pedido registrado",message:"El pedido fue enviado a Cocina. Puedes seguir su avance en Pedidos y registrar el cobro en Punto de venta."})},onError:error=>{void qc.invalidateQueries({queryKey:["order-catalog"]});void qc.invalidateQueries({queryKey:["order-combo"]});notify({tone:"danger",title:"No se pudo registrar el pedido",message:error.message})}});
 const advance=useMutation({mutationFn:(v:{id:string;status:string})=>updateOrderStatus(v.id,v.status),onSuccess:order=>{const refresh=invalidate();if(order.channel!=="salon"||order.status!=="entregado")notify({tone:"success",title:"Pedido actualizado",message:"El estado del pedido fue actualizado."});return refresh},onError:e=>notify({tone:"danger",title:"No se pudo actualizar",message:e.message})});
 const cancel=useMutation({mutationFn:(o:Order)=>updateOrderStatus(o.id,"cancelado"),onSuccess:()=>{setCancelTarget(null);invalidate();notify({tone:"success",title:"Pedido cancelado",message:"El pedido quedó marcado como cancelado."})},onError:e=>notify({tone:"danger",title:"No se pudo cancelar",message:e.message})});
 const append=useMutation({mutationFn:(draft:Draft)=>addOrderConsumption(adding!.order.id,draft,adding!.requestKey),onSuccess:order=>{setAdding(null);setDetailId(order.id);invalidate()},onError:error=>notify({tone:"danger",title:"No se pudo agregar",message:error.message})});
 const addProducts=(order:Order)=>{setAdding({order,requestKey:crypto.randomUUID(),draft:{channel:"salon",tableId:order.tableId,customerName:order.customerName,customerPhone:"",address:"",reference:"",notes:"",deliveryFee:"0",lines:[]}});setDetailId(null)};
 const items=list.data?.items??[];const counts=list.data?.channelCounts??{};const channelOptions=list.data?.channelOptions??[];
 const channelLabel=(v:string)=>channelOptions.find(o=>o.value===v)?.label??v;
 const openTotal=Object.values(counts).reduce((a,b)=>a+b,0);
 const showChannelCounts=status==="abiertos";
 const tabs:{value:string;label:string;icon:IconName;count:number}[]=[{value:"",label:"Todos",icon:"receipt",count:openTotal},...channelOptions.map(o=>({value:o.value,label:o.label,icon:channelIcons[o.value]??"receipt",count:counts[o.value]??0}))];
 const pickChannel=(v:string)=>{setChannel(v);setPage(1)};
 const onTabKey=(e:KeyboardEvent<HTMLButtonElement>,i:number)=>{
  const target=e.key==="ArrowRight"?(i+1)%tabs.length:e.key==="ArrowLeft"?(i-1+tabs.length)%tabs.length:e.key==="Home"?0:e.key==="End"?tabs.length-1:-1;
  if(target<0||target===i)return;
  e.preventDefault();pickChannel(tabs[target].value);
  (e.currentTarget.parentElement?.children[target] as HTMLElement|undefined)?.focus();
 };
 const hasActiveFilters=Boolean(q||channel||status!=="abiertos");
 const emptyTitle=hasActiveFilters?"Sin coincidencias":"Sin pedidos abiertos";
 const emptyText=hasActiveFilters?"Prueba con otro canal, estado o término de búsqueda.":"Los pedidos nuevos aparecerán aquí cuando ingresen.";
 return <div className="orders-page-shell"><PageHeader eyebrow="OPERACIÓN OMNICANAL" title="Pedidos" description="Revisa el origen, estado y avance de cada pedido sin perder el contexto operativo." action={canManage?<Button icon="plus" disabled={list.isPending||list.isError||!(list.data?.channelOptions.length)} onClick={()=>setCreating(true)}>Nuevo pedido</Button>:undefined}/>
 <div className="orders-tabs-row">
  {list.isPending?<div className="orders-tabs orders-tabs-skeleton" aria-hidden="true">{Array.from({length:4},(_,i)=><span className={"orders-tab"+(i===0?" active":"")} key={i}><i className="orders-tab-icon"/><span className="orders-tab-text"><i/><i/></span></span>)}</div>:
  <div className={"orders-tabs"+(showChannelCounts?"":" no-counts")} role="tablist" aria-label="Filtrar pedidos por canal">
   {tabs.map((t,i)=><button type="button" role="tab" key={t.value||"all"} id={`orders-tab-${t.value||"all"}`} aria-selected={channel===t.value} aria-controls="orders-tabpanel" tabIndex={channel===t.value?0:-1} data-empty={showChannelCounts&&t.count===0} className={"orders-tab ch-"+(t.value||"all")+(channel===t.value?" active":"")} onClick={()=>pickChannel(t.value)} onKeyDown={e=>onTabKey(e,i)}>
    <i className="orders-tab-icon" aria-hidden="true"><Icon name={t.icon} size={17}/></i>
    <span className="orders-tab-text"><span className="orders-tab-label">{t.label}</span>{showChannelCounts&&<b className="orders-tab-count">{t.count}</b>}</span>
   </button>)}
  </div>}
 </div>
 <section className="panel management standardized-management orders-panel" id="orders-tabpanel" role="tabpanel" aria-labelledby={`orders-tab-${channel||"all"}`} aria-busy={list.isFetching&&list.isPlaceholderData}>
  <div className="toolbar">
   <label><Icon name="search" size={18}/><input aria-label="Buscar pedidos" value={q} onChange={e=>{setQ(e.target.value);setPage(1)}} placeholder="Buscar cliente, mesa o teléfono"/></label>
   <select aria-label="Filtrar por estado" value={status} onChange={e=>{setStatus(e.target.value);setPage(1)}}><option value="">Todos los estados</option><option value="abiertos">Abiertos</option>{(list.data?.statusOptions??[]).map(o=><option key={o.value} value={o.value}>{o.label}</option>)}</select>
   {hasActiveFilters&&<button type="button" className="orders-clear-filters" onClick={()=>{setQ("");setChannel("");setStatus("abiertos");setPage(1)}}><Icon name="close" size={15}/>Limpiar filtros</button>}
  </div>
  {list.isPending?<OrdersLoading/>:list.isError?<State icon="alert" title="No pudimos cargar los pedidos" text={list.error.message} action={()=>list.refetch()}/>:!items.length?<State icon="receipt" title={emptyTitle} text={emptyText}/>:<>
   <div className="table-wrap hover-scroll">
    <table className="orders-table">
     <thead><tr><th>PEDIDO</th><th>CANAL</th><th>ESTADO</th><th>REGISTRADO</th><th>TOTAL</th><th>ACCIONES</th></tr></thead>
     <tbody>{items.map((o,index)=>{const meta=statusMeta[o.status]??{label:o.status,tone:"gray" as const};const subject=o.tableName||o.customerName||"Pedido";const secondary=o.tableName&&o.customerName?o.customerName:undefined;return <tr className={index%2?"alternate":""} key={o.id}>
      <td>
       <span className={"row-icon order-row-icon oc-"+o.channel}><Icon name={channelIcons[o.channel]??"receipt"} size={17}/></span>
       <b>{subject}</b>
       {secondary&&<small>{secondary}</small>}
       {o.channel==="salon"&&<small className="order-staff">Mozo: {o.waiterName||"Sin asignar"}</small>}
       {Boolean(o.collectedByNames?.length)&&<small className="order-staff">Cobrado por: {o.collectedByNames!.join(" · ")}</small>}
       <small className="order-mobile-registered"><time dateTime={o.createdAt}>{registeredAt(o.createdAt,location?.country,location?.timezone)}</time></small>
      </td>
      <td><span className="order-channel-cell"><Icon name={channelIcons[o.channel]??"receipt"} size={13}/>{channelLabel(o.channel)}</span></td>
      <td><Status tone={meta.tone}>{meta.label}</Status></td>
      <td><time className="order-registered" dateTime={o.createdAt}>{registeredAt(o.createdAt,location?.country,location?.timezone)}</time></td>
      <td><b className="order-table-total">{settings.currencySymbol} {money(o.total)}</b></td>
      <td><div className="orders-table-actions"><RowActionButton action="view" onClick={()=>setDetailId(o.id)}/></div></td>
     </tr>})}</tbody>
    </table>
   </div>
  </>}
  <Pagination page={page} size={size} total={list.data?.total??0} onPage={setPage} onSize={v=>{setSize(v);setPage(1)}}/>
 </section>
 {creating&&canManage&&<ManualOrderDialog channels={channelOptions} currencySymbol={settings.currencySymbol} busy={create.isPending} onClose={()=>{if(!create.isPending)setCreating(false)}} onSave={async draft=>{try{await create.mutateAsync(draft)}catch{/* Feedback conserva el error y el formulario permanece abierto. */}}}/>}
 {adding&&<ComandaView mode="append" initial={adding.draft} allTables={[{id:adding.order.tableId,name:adding.order.tableName,zone:"",seats:0,order:null}]} busy={append.isPending} currencySymbol={settings.currencySymbol} close={()=>setAdding(null)} save={draft=>append.mutateAsync(draft).then(()=>{}).catch(()=>{})} notify={notify}/>}
 {detailId&&<OrderDetail loading={detail.isLoading} order={detail.data} error={detail.error?.message} currencySymbol={settings.currencySymbol} canManage={canManage} busy={advance.isPending} close={()=>setDetailId(null)} advance={st=>advance.mutate({id:detailId,status:st})} add={addProducts} cancel={o=>setCancelTarget(o)}/>}
 <ConfirmDialog open={Boolean(cancelTarget)} title="Cancelar pedido" description={`${cancelTarget?.tableName||cancelTarget?.customerName||"El pedido seleccionado"} quedará cancelado y no podrá reactivarse.`} tone="danger" confirmLabel="Cancelar pedido" pending={cancel.isPending} onCancel={()=>setCancelTarget(null)} onConfirm={()=>cancelTarget&&cancel.mutate(cancelTarget)}/></div>;
}

function OrderDetail({loading,order,error,currencySymbol,canManage:hasPermission,busy,close,advance,add,cancel}:{loading:boolean;order?:Order;error?:string;currencySymbol:string;canManage:boolean;busy:boolean;close:()=>void;advance:(st:string)=>void;add:(o:Order)=>void;cancel:(o:Order)=>void}){
 const{location,user,can}=useSession();
 const canManage=Boolean(hasPermission&&order&&canManageOrderService(order,user?.id));
 const canCharge=Boolean(can("cash.manage")&&order&&canChargeAccount(order)&&order.paymentStatus!=="paid");
 const meta=order?statusMeta[order.status]??{label:order.status,tone:"gray" as const}:null;
 const action=order?nextOrderAction(order):null;
 const itemCount=order?(order.items??[]).reduce((sum,it)=>sum+Number(it.qty||0),0):0;
 const subject=order?(order.tableName||order.customerName||"Pedido"):"Pedido";
 const subtitle=order?.tableName&&order.customerName?order.customerName:undefined;
 return <div className="modal-backdrop modal-overlay-in">
  <Dialog onResponseClose={close} className="crud-modal order-detail salon-order-detail modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="orders-preview-title" aria-busy={loading}>
   <div className="salon-order-detail-accent" aria-hidden="true"/>
   {loading?<OrderDetailSkeleton close={close}/>:<>
   <header className="salon-order-detail-head">
    <div className="salon-order-detail-identity">
     <span className="salon-order-detail-icon"><Icon name={order?channelIcons[order.channel]??"receipt":"receipt"} size={20}/></span>
     <div className="salon-order-detail-heading">
      <h2 id="orders-preview-title">{subject}</h2>
      {order&&<p>{subtitle&&<><span>{subtitle}</span><span aria-hidden="true"> · </span></>}<span className="salon-order-detail-opened"><Icon name="clock" size={14}/><time dateTime={order.createdAt}>{registeredAt(order.createdAt,location?.country,location?.timezone)}</time></span></p>}
     </div>
    </div>
    <div className="salon-order-detail-status">{order&&meta&&<Status tone={meta.tone}>{meta.label}</Status>}{order?.channel==="salon"&&<Status tone="blue">{accountLabel(order)}</Status>}</div>
    <button type="button" className="salon-order-detail-close" aria-label="Cerrar detalle" onClick={close}><Icon name="close" size={17}/></button>
   </header>

   {error?(
    <div className="order-detail-body"><div className="catalog-state error"><span><Icon name="alert" size={22}/></span><b>Error al cargar el pedido</b><p>{error}</p></div></div>
   ):order&&meta&&<>
    <div className="order-detail-body salon-order-detail-body">
     <div className="salon-order-detail-content">

     <OrderAttribution channel={order.channel} waiterName={order.waiterName} collectedByNames={order.collectedByNames}/>

     {(order.customerPhone||order.address||order.reference)&&<section className="order-detail-context" aria-label="Contacto y entrega">
      {order.customerPhone&&<div><span><Icon name="users" size={15}/></span><p><small>CONTACTO</small><b>{order.customerPhone}</b></p></div>}
      {(order.address||order.reference)&&<div className="order-detail-context-wide"><span><Icon name="truck" size={15}/></span><p><small>ENTREGA</small><b>{order.address||"Dirección no registrada"}</b>{order.reference&&<em>{order.reference}</em>}</p></div>}
     </section>}

     <section className="salon-order-detail-consumption">
      <header className="salon-order-detail-section-head"><div><h3>Productos del pedido</h3></div><span>{itemCount} unidad{itemCount===1?"":"es"}</span></header>
      <div className="order-detail-items salon-order-detail-items">
       {(order.items??[]).map(it=><div className="order-detail-line salon-order-detail-line" key={it.id}>
        <b className="salon-order-detail-qty">{Number(it.qty)}×</b>
        <div className="order-detail-line-info">
         <span>{it.name}</span>
         <small>{currencySymbol} {money(it.unitPrice)} c/u</small>
         {it.itemType==="combo"&&(it.selections??[]).length>0&&<div className="salon-order-detail-selections">{(it.selections??[]).map(sel=><small key={sel.groupId+sel.productId}><b>{sel.groupName}:</b> {sel.name}{Number(sel.surcharge)>0?` (+${currencySymbol} ${money(sel.surcharge)})`:""}</small>)}</div>}
         {it.note&&<em>{it.note}</em>}
         <OrderItemService order={order} itemId={it.id}/>
        </div>
        <strong className="salon-order-detail-line-total">{currencySymbol} {money(Number(it.qty)*Number(it.unitPrice))}</strong>
       </div>)}
       {!(order.items??[]).length&&<div className="salon-order-detail-empty"><Icon name="receipt" size={20}/><span>Sin ítems cargados aún.</span></div>}
      </div>

      {order.notes&&<div className="salon-order-detail-notes"><span><Icon name="edit" size={15}/></span><div><b>Notas generales</b><p>{order.notes}</p></div></div>}
     </section>
    <section className="salon-order-detail-totals" aria-label="Totales del pedido">
     {Number(order.deliveryFee)>0&&<div className="salon-order-detail-subtotal"><span>Subtotal</span><b>{currencySymbol} {money(order.subtotal)}</b></div>}
     {Number(order.deliveryFee)>0&&<div className="salon-order-detail-subtotal"><span>Delivery</span><b>{currencySymbol} {money(order.deliveryFee)}</b></div>}
     <div className="salon-order-detail-grand"><span>Total del pedido</span><strong>{currencySymbol} {money(order.total)}</strong></div>
    </section>
     </div>
    </div>

    {(canManage||canCharge)&&<footer className="order-detail-actions salon-order-detail-actions">
     {order.status==="entregado"&&<span className="order-detail-done"><Icon name="check" size={15}/>{order.channel==="salon"&&!order.completedAt?"Consumo entregado":"Pedido completado"}</span>}
     <div className="salon-order-detail-buttons" role="group" aria-label="Acciones del pedido">
      <OrderAccountActions order={order} onAdd={()=>add(order)} busy={busy}/>
      {canManage&&canCancelOrder(order)&&<Button icon="cancel" kind="ghost" className="order-detail-cancel" disabled={busy} onClick={()=>cancel(order)}>Cancelar pedido</Button>}
      {canManage&&action&&<Button icon={action.status==="entregado"?"availability":action.icon} className="order-detail-primary" disabled={busy} aria-busy={busy} onClick={()=>advance(action.status)}>{action.label}</Button>}
      {canCharge&&<Link href={`${pageRoutes.pos}?orderId=${order.id}`} className="button secondary salon-order-detail-pay" aria-disabled={busy} aria-busy={busy} tabIndex={busy?-1:undefined} onClick={event=>{if(busy)event.preventDefault();}}><Icon name="payment" size={18}/><span>Cobrar saldo</span></Link>}
     </div>
    </footer>}
   </>}
   </>}
  </Dialog>
 </div>;
}

function OrderDetailSkeleton({close}:{close:()=>void}){
 return <>
  <header className="salon-order-detail-head salon-order-detail-skeleton-head">
   <h2 id="orders-preview-title" className="sr-only">Cargando detalle del pedido</h2>
   <div className="salon-order-detail-identity" aria-hidden="true">
    <span className="salon-order-detail-skeleton-block salon-order-detail-skeleton-icon"/>
    <div className="salon-order-detail-skeleton-copy">
     <span className="salon-order-detail-skeleton-block title"/>
     <span className="salon-order-detail-skeleton-block medium"/>
    </div>
   </div>
   <span className="salon-order-detail-skeleton-block salon-order-detail-skeleton-status" aria-hidden="true"/>
   <button type="button" className="salon-order-detail-close" aria-label="Cerrar detalle" onClick={close}><Icon name="close" size={17}/></button>
  </header>
  <div className="order-detail-body salon-order-detail-body salon-order-detail-skeleton-body" aria-label="Cargando detalle del pedido">
   <div className="salon-order-detail-content" aria-hidden="true">
   <OrderAttributionSkeleton/>
   <section className="salon-order-detail-consumption salon-order-detail-skeleton-consumption">
    <div className="salon-order-detail-skeleton-section-head">
     <div><span className="salon-order-detail-skeleton-block heading"/></div>
     <span className="salon-order-detail-skeleton-block tiny"/>
    </div>
    <div className="order-detail-items salon-order-detail-items salon-order-detail-skeleton-items">
     {Array.from({length:3},(_,i)=><div className="salon-order-detail-skeleton-line" key={i}>
      <span className="salon-order-detail-skeleton-block salon-order-detail-skeleton-qty"/>
      <span className="salon-order-detail-skeleton-copy"><span className="salon-order-detail-skeleton-block line-title"/><span className="salon-order-detail-skeleton-block medium"/></span>
      <span className="salon-order-detail-skeleton-block price"/>
     </div>)}
    </div>
   </section>
  <section className="salon-order-detail-totals salon-order-detail-skeleton-totals" aria-hidden="true">
   <div className="salon-order-detail-grand"><span className="salon-order-detail-skeleton-block total-label"/><span className="salon-order-detail-skeleton-block total-amount"/></div>
  </section>
   </div>
  </div>
  <footer className="order-detail-actions salon-order-detail-actions salon-order-detail-skeleton-actions" aria-hidden="true">
   <div className="salon-order-detail-buttons"><span className="salon-order-detail-skeleton-block action primary"/></div>
  </footer>
 </>;
}

function OrdersLoading(){return <div className="orders-loading" aria-label="Cargando pedidos" aria-busy="true">
 <div className="table-skeleton orders-table-skeleton">
  <div className="sk-head"><i/><i/><i/><i/><i/><i/></div>
  {Array.from({length:5},(_,index)=><div className="sk-row" key={index}><i className="sk-name"><span/><b/></i><i/><i/><i/><i/><i/></div>)}
 </div>
 </div>}
function State({icon,title,text,action}:{icon:"alert"|"receipt";title:string;text:string;action?:()=>void}){return <div className="catalog-state"><span><Icon name={icon}/></span><b>{title}</b><p>{text}</p>{action&&<Button kind="ghost" onClick={action}>Reintentar</Button>}</div>}
