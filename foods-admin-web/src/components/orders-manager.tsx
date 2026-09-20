"use client";
import "../app/salon.css";
import Link from "next/link";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,Icon,IconName,PageHeader,Pagination,RowActionButton,Status} from "@/design-system";
import {apiFetch} from "@/shared/api/client";
import {useFeedback} from "@/providers/feedback-provider";
import {useSettings} from "@/providers/settings-context";
import {useSession} from "@/providers/session-context";

type Option={value:string;label:string};
type OrderItemSelection={groupId:string;groupName:string;productId:string;name:string;surcharge:string};
type OrderItem={id:string;productId:string;name:string;qty:string;unitPrice:string;note:string;itemType?:"product"|"combo";selections?:OrderItemSelection[]};
type Order={id:string;code:string;channel:string;status:string;customerId:string;customerName:string;customerPhone:string;address:string;reference:string;tableId:string;tableName:string;notes:string;subtotal:string;deliveryFee:string;total:string;createdAt:string;updatedAt:string;items?:OrderItem[]};
type Response={items:Order[];total:number;channelCounts:Record<string,number>;channelOptions:Option[];statusOptions:Option[]};

const channelIcons:Record<string,IconName>={salon:"utensils",mostrador:"store",recojo:"box",delivery:"truck",whatsapp:"share"};
const statusMeta:Record<string,{label:string;tone:"green"|"blue"|"orange"|"gray"}>={nuevo:{label:"Nuevo",tone:"blue"},confirmado:{label:"Confirmado",tone:"blue"},preparando:{label:"Preparando",tone:"orange"},listo:{label:"Listo",tone:"green"},en_camino:{label:"En camino",tone:"orange"},entregado:{label:"Entregado",tone:"gray"},cancelado:{label:"Cancelado",tone:"gray"}};

function nextAction(o:Order):{status:string;label:string;icon:IconName}|null{
 if(o.status==="nuevo")return{status:"confirmado",label:"Confirmar",icon:"receipt"};
 if(o.status==="confirmado")return{status:"preparando",label:"Iniciar preparación",icon:"chefHat"};
 if(o.status==="preparando")return{status:"listo",label:"Marcar listo",icon:"check"};
 if(o.status==="listo")return o.channel==="delivery"?{status:"en_camino",label:"En camino",icon:"truck"}:{status:"entregado",label:"Entregar",icon:"check"};
 if(o.status==="en_camino")return{status:"entregado",label:"Entregar",icon:"check"};
 return null;
}
const cancellable=(o:Order)=>!["entregado","cancelado"].includes(o.status);
function parseIsoDate(iso:string):Date|null{
  if(!iso)return null;
  const normalized=iso.replace(/([+-]\d{2})$/,"$1:00");
  const d=new Date(normalized);
  if(!isNaN(d.getTime()))return d;
  const d2=new Date(iso);
  return isNaN(d2.getTime())?null:d2;
}
function timeAgo(iso:string){
  const d=parseIsoDate(iso);if(!d)return"—";
  const m=Math.floor((Date.now()-d.getTime())/60000);
  if(m<1)return"Justo ahora";if(m<60)return`Hace ${m} min`;
  const h=Math.floor(m/60);if(h<24)return`Hace ${h} h`;
  return new Intl.DateTimeFormat("es-PE",{dateStyle:"medium",timeStyle:"short"}).format(d);
}
const money=(v:string|number)=>Number(v).toFixed(2);

export function OrdersManager(){
 const qc=useQueryClient();const{notify}=useFeedback();const settings=useSettings();const{can}=useSession();const canManage=can("orders.manage");
 const[q,setQ]=useState("");const[channel,setChannel]=useState("");const[status,setStatus]=useState("abiertos");const[page,setPage]=useState(1);const[size,setSize]=useState(12);
 const[detailId,setDetailId]=useState<string|null>(null);const[cancelTarget,setCancelTarget]=useState<Order|null>(null);
 const list=useQuery({queryKey:["orders",q,channel,status,page,size],queryFn:()=>apiFetch<Response>(`orders?q=${encodeURIComponent(q)}&channel=${channel}&status=${status}&page=${page}&pageSize=${size}`)});
 const detail=useQuery({queryKey:["order",detailId],queryFn:()=>apiFetch<Order>(`orders/${detailId}`),enabled:Boolean(detailId)});
 const invalidate=()=>{void qc.invalidateQueries({queryKey:["orders"]});void qc.invalidateQueries({queryKey:["order",detailId]})};
 const advance=useMutation({mutationFn:(v:{id:string;status:string})=>apiFetch<Order>(`orders/${v.id}/status`,{method:"PATCH",body:JSON.stringify({status:v.status})}),onSuccess:()=>{invalidate();notify({tone:"success",title:"Pedido actualizado",message:"El estado del pedido fue actualizado."})},onError:e=>notify({tone:"danger",title:"No se pudo actualizar",message:e.message})});
 const cancel=useMutation({mutationFn:(o:Order)=>apiFetch<Order>(`orders/${o.id}/status`,{method:"PATCH",body:JSON.stringify({status:"cancelado"})}),onSuccess:()=>{setCancelTarget(null);invalidate();notify({tone:"success",title:"Pedido cancelado",message:"El pedido quedó marcado como cancelado."})},onError:e=>notify({tone:"danger",title:"No se pudo cancelar",message:e.message})});
 const items=list.data?.items??[];const counts=list.data?.channelCounts??{};const channelOptions=list.data?.channelOptions??[];
 const channelLabel=(v:string)=>channelOptions.find(o=>o.value===v)?.label??v;
 const openTotal=Object.values(counts).reduce((a,b)=>a+b,0);
 return <><PageHeader eyebrow="OPERACIÓN" title="Pedidos" description="Supervisa pedidos de todos los canales, su avance y las entregas desde una sola bandeja." action={canManage?<Link href="/salon" className="orders-salon-link"><Icon name="utensils" size={16}/><span>Ir a Salón</span></Link>:undefined}/>
 <div className="catalog-tabs-row orders-tabs-row">
  <div className="catalog-tabs orders-tabs" role="tablist" aria-label="Filtrar pedidos por canal">
   <button className={channel===""?"active":""} onClick={()=>{setChannel("");setPage(1)}}><Icon name="receipt" size={14}/><span>Todos</span><b>{openTotal}</b></button>
   {channelOptions.map(o=><button key={o.value} className={channel===o.value?"active":""} onClick={()=>{setChannel(o.value);setPage(1)}}><Icon name={channelIcons[o.value]??"receipt"} size={14}/><span>{o.label}</span><b>{counts[o.value]??0}</b></button>)}
  </div>
 </div>
 <section className="panel management standardized-management orders-panel">
  <div className="toolbar">
   <label><Icon name="search" size={18}/><input value={q} onChange={e=>{setQ(e.target.value);setPage(1)}} placeholder="Buscar por código, cliente o teléfono..."/></label>
   <select aria-label="Filtrar por estado" value={status} onChange={e=>{setStatus(e.target.value);setPage(1)}}><option value="">Todos los estados</option><option value="abiertos">Abiertos</option>{(list.data?.statusOptions??[]).map(o=><option key={o.value} value={o.value}>{o.label}</option>)}</select>
  </div>
  {list.isLoading?<Loading/>:list.isError?<State icon="alert" title="No pudimos cargar los pedidos" text={list.error.message} action={()=>list.refetch()}/>:!items.length?<State icon="receipt" title="Sin pedidos" text="No hay pedidos que coincidan con los filtros actuales."/>:<>
   <div className="table-wrap hover-scroll">
    <table className="orders-table">
     <thead><tr><th>PEDIDO</th><th>CANAL</th><th>ESTADO</th><th>TOTAL</th><th>ACCIONES</th></tr></thead>
     <tbody>{items.map((o,index)=>{const meta=statusMeta[o.status]??{label:o.status,tone:"gray" as const};const subject=o.tableName||o.customerName||"Pedido sin nombre";return <tr className={index%2?"alternate":""} key={o.id}>
      <td>
       <span className={"row-icon order-row-icon oc-"+o.channel}><Icon name={channelIcons[o.channel]??"receipt"} size={17}/></span>
       <b>{subject}</b>
       <small>{o.code} · {timeAgo(o.createdAt)}{o.tableName&&o.customerName?` · ${o.customerName}`:""}{o.notes?` · ${o.notes}`:""}</small>
      </td>
      <td><span className="order-channel-cell"><Icon name={channelIcons[o.channel]??"receipt"} size={13}/>{channelLabel(o.channel)}</span></td>
      <td><Status tone={meta.tone}>{meta.label}</Status></td>
      <td><b className="order-table-total">{settings.currencySymbol} {money(o.total)}</b></td>
      <td><div className="orders-table-actions"><RowActionButton action="view" onClick={()=>setDetailId(o.id)}/></div></td>
     </tr>})}</tbody>
    </table>
   </div>
   <div className="management-cards orders-mobile-cards">{items.map(o=>{const meta=statusMeta[o.status]??{label:o.status,tone:"gray" as const};const subject=o.tableName||o.customerName||"Pedido sin nombre";return <article key={o.id}>
    <header>
     <span className={"row-icon order-row-icon oc-"+o.channel}><Icon name={channelIcons[o.channel]??"receipt"} size={17}/></span>
     <div><b>{subject}</b><small>{o.code} · {channelLabel(o.channel)}</small></div>
     <Status tone={meta.tone}>{meta.label}</Status>
    </header>
    <dl>
     <div><dt>REGISTRADO</dt><dd>{timeAgo(o.createdAt)}</dd></div>
     <div><dt>TOTAL</dt><dd>{settings.currencySymbol} {money(o.total)}</dd></div>
    </dl>
    {(o.address||o.customerName||o.notes)&&<p className="orders-mobile-detail">{o.tableName&&o.customerName?o.customerName:o.address||o.notes||o.customerName}</p>}
    <footer><RowActionButton action="view" onClick={()=>setDetailId(o.id)}/></footer>
   </article>})}</div>
  </>}
  <Pagination page={page} size={size} total={list.data?.total??0} onPage={setPage} onSize={v=>{setSize(v);setPage(1)}}/>
 </section>
 {detailId&&<OrderDetail loading={detail.isLoading} order={detail.data} error={detail.error?.message} channels={channelOptions} currencySymbol={settings.currencySymbol} canManage={canManage} busy={advance.isPending} close={()=>setDetailId(null)} advance={st=>advance.mutate({id:detailId,status:st})} cancel={o=>setCancelTarget(o)}/>}
 <ConfirmDialog open={Boolean(cancelTarget)} title="Cancelar pedido" description={`El pedido ${cancelTarget?.code??""} quedará cancelado y no podrá reactivarse.`} tone="danger" confirmLabel="Cancelar pedido" pending={cancel.isPending} onCancel={()=>setCancelTarget(null)} onConfirm={()=>cancelTarget&&cancel.mutate(cancelTarget)}/></>;
}

function OrderDetail({loading,order,error,channels,currencySymbol,canManage,busy,close,advance,cancel}:{loading:boolean;order?:Order;error?:string;channels:Option[];currencySymbol:string;canManage:boolean;busy:boolean;close:()=>void;advance:(st:string)=>void;cancel:(o:Order)=>void}){
 const channelLabel=(value:string)=>channels.find(option=>option.value===value)?.label??value;
 const meta=order?statusMeta[order.status]??{label:order.status,tone:"gray" as const}:null;
 const action=order?nextAction(order):null;
 const itemCount=order?(order.items??[]).reduce((sum,it)=>sum+Number(it.qty||0),0):0;
 const subject=order?(order.tableName||order.customerName||order.code):"Pedido";
 const subtitle=order?.tableName&&order.customerName?order.customerName:order?.code;
 return <div className="modal-backdrop modal-overlay-in">
  <section className="crud-modal order-detail salon-order-detail modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="orders-preview-title">
   <div className="salon-order-detail-accent"/>
   <header className="salon-order-detail-head">
    <div className="salon-order-detail-identity">
     <span className="salon-order-detail-icon"><Icon name={order?channelIcons[order.channel]??"receipt":"receipt"} size={20}/></span>
     <div className="salon-order-detail-heading">
      <small>{order?.channel==="salon"?"MESA ACTIVA":order?`PEDIDO · ${channelLabel(order.channel).toUpperCase()}`:"PEDIDO"}</small>
      <h2 id="orders-preview-title">{subject}</h2>
      {subtitle&&<p>{subtitle}</p>}
     </div>
    </div>
    <div className="salon-order-detail-status">{order&&meta&&<Status tone={meta.tone}>{meta.label}</Status>}</div>
    <button type="button" className="salon-order-detail-close" aria-label="Cerrar detalle" onClick={close}><Icon name="close" size={17}/></button>
   </header>

   {loading?<Loading/>:error?(
    <div className="order-detail-body"><div className="catalog-state error"><span><Icon name="alert" size={22}/></span><b>Error al cargar el pedido</b><p>{error}</p></div></div>
   ):order&&meta&&<>
    <div className="order-detail-body salon-order-detail-body">
     <section className="salon-order-detail-meta" aria-label="Datos del pedido">
      <div><span className="salon-order-detail-meta-icon"><Icon name="receipt" size={15}/></span><span><small>PEDIDO</small><b>{order.code}</b></span></div>
      <div><span className="salon-order-detail-meta-icon"><Icon name="clock" size={15}/></span><span><small>REGISTRADO</small><b>{timeAgo(order.createdAt)}</b></span></div>
      <div><span className="salon-order-detail-meta-icon"><Icon name="utensils" size={15}/></span><span><small>CONSUMO</small><b>{itemCount} ítem{itemCount===1?"":"s"}</b></span></div>
     </section>

     <section className="salon-order-detail-consumption">
      <header className="salon-order-detail-section-head"><div><small>DETALLE</small><h3>Productos del pedido</h3></div><span>{(order.items??[]).length} línea{(order.items??[]).length===1?"":"s"}</span></header>
      <div className="order-detail-items salon-order-detail-items">
       {(order.items??[]).map(it=><div className="order-detail-line salon-order-detail-line" key={it.id}>
        <b className="salon-order-detail-qty">{Number(it.qty)}×</b>
        <div className="order-detail-line-info">
         <span>{it.name}</span>
         {Number(it.qty)>1&&<small>{currencySymbol} {money(it.unitPrice)} c/u</small>}
         {it.itemType==="combo"&&(it.selections??[]).length>0&&<div className="salon-order-detail-selections">{(it.selections??[]).map(sel=><small key={sel.groupId+sel.productId}><b>{sel.groupName}:</b> {sel.name}{Number(sel.surcharge)>0?` (+${currencySymbol} ${money(sel.surcharge)})`:""}</small>)}</div>}
         {it.note&&<em>{it.note}</em>}
        </div>
        <strong className="salon-order-detail-line-total">{currencySymbol} {money(Number(it.qty)*Number(it.unitPrice))}</strong>
       </div>)}
       {!(order.items??[]).length&&<div className="salon-order-detail-empty"><Icon name="receipt" size={20}/><span>Sin ítems cargados aún.</span></div>}
      </div>

      {(order.address||order.reference)&&<div className="salon-order-detail-notes"><span><Icon name="truck" size={15}/></span><div><b>Entrega</b><p>{order.address}{order.reference?` · ${order.reference}`:""}</p></div></div>}
      {order.notes&&<div className="salon-order-detail-notes"><span><Icon name="edit" size={15}/></span><div><b>Notas generales</b><p>{order.notes}</p></div></div>}
     </section>
    </div>

    <section className="salon-order-detail-totals" aria-label="Totales del pedido">
     {Number(order.deliveryFee)>0&&<div className="salon-order-detail-subtotal"><span>Productos</span><b>{currencySymbol} {money(order.subtotal)}</b></div>}
     {Number(order.deliveryFee)>0&&<div className="salon-order-detail-subtotal"><span>Delivery</span><b>{currencySymbol} {money(order.deliveryFee)}</b></div>}
     <div className="salon-order-detail-grand"><span>Total del pedido</span><strong>{currencySymbol} {money(order.total)}</strong></div>
    </section>

    {canManage&&<footer className="order-detail-actions salon-order-detail-actions">
     {action&&<Button icon={action.icon} className="order-detail-primary" disabled={busy} onClick={()=>advance(action.status)}>{action.label}</Button>}
     {order.status==="entregado"&&<span className="order-detail-done"><Icon name="check" size={15}/>Pedido completado</span>}
     {cancellable(order)&&<Button icon="alert" kind="ghost" className="order-detail-cancel" disabled={busy} onClick={()=>cancel(order)}>Cancelar pedido</Button>}
    </footer>}
   </>}
  </section>
 </div>;
}

function Loading(){return <div className="customers-loading" aria-label="Cargando"><i/><i/><i/><i/></div>}
function State({icon,title,text,action}:{icon:"alert"|"receipt";title:string;text:string;action?:()=>void}){return <div className="catalog-state"><span><Icon name={icon}/></span><b>{title}</b><p>{text}</p>{action&&<Button kind="ghost" onClick={action}>Reintentar</Button>}</div>}
