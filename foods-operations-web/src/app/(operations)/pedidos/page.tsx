"use client";

import {useCallback,useEffect,useState} from "react";
import {Button} from "@/components/ui/controls";
import {Icon} from "@/components/icon";
import {operationsFetch} from "@/lib/operations-api";

type Channel="salon"|"mostrador"|"recojo"|"delivery"|"whatsapp";
type OrderStatus="nuevo"|"confirmado"|"preparando"|"listo"|"en_camino"|"entregado"|"cancelado";
type OrderItem={id:string;productId:string;name:string;qty:string;unitPrice:string;note:string;itemType:string};
type Order={
 id:string;code:string;channel:Channel;status:OrderStatus;customerName:string;customerPhone:string;
 address:string;reference:string;tableId:string;tableName:string;notes:string;subtotal:string;deliveryFee:string;
 total:string;createdAt:string;updatedAt:string;items?:OrderItem[];
};
type Option={value:string;label:string};
type OrderList={
 items:Order[];total:number;page:number;pageSize:number;channelCounts:Record<string,number>;
 channelOptions:Option[];statusOptions:Option[];currencySymbol:string;
};
type ChannelFilter=Channel|"all";

const channelPresentation:Record<Channel,{fallback:string;icon:"whatsapp"|"bike"|"store"|"tables"}>={
 whatsapp:{fallback:"WhatsApp",icon:"whatsapp"},
 delivery:{fallback:"Delivery",icon:"bike"},
 recojo:{fallback:"Recojo",icon:"store"},
 mostrador:{fallback:"Mostrador",icon:"store"},
 salon:{fallback:"Salón",icon:"tables"},
};

const stateLabel:Record<OrderStatus,string>={
 nuevo:"Nuevo",confirmado:"Confirmado",preparando:"Preparando",listo:"Listo",
 en_camino:"En camino",entregado:"Entregado",cancelado:"Cancelado",
};
const stateTone:Record<OrderStatus,string>={
 nuevo:"violet",confirmado:"blue",preparando:"blue",listo:"green",
 en_camino:"violet",entregado:"muted",cancelado:"muted",
};

const channels:ChannelFilter[]=["all","whatsapp","delivery","recojo","mostrador","salon"];
const PAGE_SIZE=20;

function formatMoney(symbol:string,value:string){
 const amount=Number(value);
 const formatted=Number.isFinite(amount)?amount.toFixed(2):value;
 return symbol?symbol+" "+formatted:formatted;
}
function formatCreatedAt(value:string){
 const date=new Date(value);
 if(Number.isNaN(date.getTime()))return value;
 return new Intl.DateTimeFormat("es-PE",{
  day:"2-digit",month:"2-digit",hour:"2-digit",minute:"2-digit",
 }).format(date);
}
function lineTotal(item:OrderItem){
 const qty=Number(item.qty);
 const price=Number(item.unitPrice);
 if(!Number.isFinite(qty)||!Number.isFinite(price))return item.unitPrice;
 return (qty*price).toFixed(2);
}
function nextAction(order:Order):{label:string;status:OrderStatus}|null{
 if(order.status==="nuevo")return{label:"Confirmar",status:"confirmado"};
 if(order.status==="listo"&&order.channel==="delivery")return{label:"En camino",status:"en_camino"};
 if(order.status==="listo")return{label:"Entregar",status:"entregado"};
 if(order.status==="en_camino")return{label:"Entregar",status:"entregado"};
 return null;
}
function contactWhatsApp(phone:string){
 const digits=phone.replace(/\D/g,"");
 if(!digits)return;
 window.open("https://wa.me/"+digits,"_blank","noopener,noreferrer");
}

export default function OrdersPage(){
 const[channel,setChannel]=useState<ChannelFilter>("all");
 const[page,setPage]=useState(1);
 const[data,setData]=useState<OrderList|null>(null);
 const[loading,setLoading]=useState(true);
 const[error,setError]=useState("");
 const[selected,setSelected]=useState<Order|null>(null);
 const[detailLoading,setDetailLoading]=useState(false);
 const[actionId,setActionId]=useState<string|null>(null);

 const load=useCallback(async()=>{
  setLoading(true);
  try{
   const params=new URLSearchParams({
    status:"abiertos",page:String(page),pageSize:String(PAGE_SIZE),
   });
   if(channel!=="all")params.set("channel",channel);
   const result=await operationsFetch<OrderList>("orders?"+params.toString());
   setData(result);
   setError("");
  }catch(e){
   setError(e instanceof Error?e.message:"No pudimos cargar los pedidos.");
  }finally{
   setLoading(false);
  }
 },[channel,page]);

 useEffect(()=>{
  const task=window.setTimeout(()=>void load(),0);
  return()=>window.clearTimeout(task);
 },[load]);

 const selectOrder=async(order:Order)=>{
  setSelected(order);
  setDetailLoading(true);
  try{
   const detail=await operationsFetch<Order>("orders/"+order.id);
   setSelected(detail);
  }catch(e){
   setError(e instanceof Error?e.message:"No pudimos cargar el detalle.");
  }finally{
   setDetailLoading(false);
  }
 };

 const changeChannel=(value:ChannelFilter)=>{
  setChannel(value);
  setPage(1);
  setSelected(null);
 };

 const advance=async(order:Order)=>{
  const action=nextAction(order);
  if(!action||actionId)return;
  setActionId(order.id);
  try{
   await operationsFetch<Order>("orders/"+order.id+"/status",{
    method:"PATCH",
    body:JSON.stringify({status:action.status}),
   });
   if(selected?.id===order.id){
    const detail=await operationsFetch<Order>("orders/"+order.id);
    setSelected(detail);
   }
   await load();
  }catch(e){
   setError(e instanceof Error?e.message:"No pudimos actualizar el pedido.");
  }finally{
   setActionId(null);
  }
 };

 const labelFor=(value:Channel)=>{
  const dynamic=data?.channelOptions.find(option=>option.value===value)?.label;
  return dynamic??channelPresentation[value].fallback;
 };
 const countFor=(value:Channel)=>data?.channelCounts[value]??0;
 const allCount=data?Object.values(data.channelCounts).reduce((sum,count)=>sum+count,0):0;
 const totalPages=data?Math.max(1,Math.ceil(data.total/data.pageSize)):1;
 const currency=data?.currencySymbol??"";

 return <div className="orders-page">
  <header className="orders-header">
   <div>
    <span className="orders-eyebrow">VENTA OMNICANAL</span>
    <h1>Pedidos</h1>
    <p>Pedidos reales del local activo, incluidos los confirmados por Fudia Concierge.</p>
   </div>
  </header>

  <div className="orders-channels" aria-label="Filtrar pedidos por canal">
   {channels.map(value=>{
    const info=value==="all"?{fallback:"Todos",icon:"orders" as const}:channelPresentation[value];
    const count=value==="all"?allCount:countFor(value);
    return <Button key={value} className={"orders-ch"+(channel===value?" active":"")} onClick={()=>changeChannel(value)}>
     <Icon name={info.icon} size={18}/>
     <span>{value==="all"?"Todos":labelFor(value)}</span>
     <b>{count}</b>
    </Button>;
   })}
  </div>

  {error&&<div className="orders-live-error" role="alert"><Icon name="wifi" size={18}/><span>{error}</span><Button onClick={()=>void load()}><Icon name="refresh" size={15}/>Reintentar</Button></div>}

  <div className="orders-list" aria-busy={loading}>
   {loading&&Array.from({length:5},(_,index)=><div className="order-card orders-skeleton" key={index} aria-hidden="true">
    <i/><span><b/><small/></span><em/>
   </div>)}

   {!loading&&data?.items.map(order=>{
    const info=channelPresentation[order.channel];
    const tone=stateTone[order.status];
    const action=nextAction(order);
    const person=order.customerName||order.tableName||"Cliente";
    return <article className={"order-card"+(order.id===selected?.id?" selected":"")} key={order.id} onClick={()=>void selectOrder(order)}>
     <div className="order-card-left">
      <div className={"order-channel-icon "+order.channel}><Icon name={info.icon} size={18}/></div>
      <div className="order-card-info">
       <div className="order-card-top">
        <strong>{person}</strong>
        <span className="order-id">{order.code}</span>
       </div>
       <div className="order-card-meta">
        <span className="order-channel-tag"><Icon name={info.icon} size={12}/>{labelFor(order.channel)}</span>
        <span className={"order-state-tag os-"+tone}>{stateLabel[order.status]}</span>
        <small>{formatCreatedAt(order.createdAt)}</small>
        {order.tableName&&<small>· {order.tableName}</small>}
       </div>
      </div>
     </div>
     <div className="order-card-right">
      <b className="order-total">{formatMoney(currency,order.total)}</b>
      {action?<Button tone="operational" className="order-advance" disabled={actionId===order.id} onClick={event=>{event.stopPropagation();void advance(order);}}>
       {actionId===order.id?"Guardando…":action.label}<Icon name="chevron" size={14}/>
      </Button>:<span className="order-managed">{order.status==="confirmado"||order.status==="preparando"?"Gestionado en cocina":stateLabel[order.status]}</span>}
     </div>
    </article>;
   })}

   {!loading&&data&&data.items.length===0&&<div className="orders-empty">
    <Icon name={channel==="whatsapp"?"whatsapp":"orders"} size={28}/>
    <b>Sin pedidos activos</b>
    <span>No hay pedidos pendientes en este canal.</span>
   </div>}
  </div>

  {!loading&&data&&data.total>0&&<footer className="orders-live-pagination">
   <span>Mostrando {(data.page-1)*data.pageSize+1}–{Math.min(data.page*data.pageSize,data.total)} de {data.total}</span>
   <div>
    <Button disabled={page<=1} onClick={()=>setPage(current=>Math.max(1,current-1))}>Anterior</Button>
    <b>{page} / {totalPages}</b>
    <Button disabled={page>=totalPages} onClick={()=>setPage(current=>Math.min(totalPages,current+1))}>Siguiente</Button>
   </div>
  </footer>}

  {selected&&<aside className="order-detail-overlay" onClick={()=>setSelected(null)}>
   <div className="order-detail" onClick={event=>event.stopPropagation()}>
    <header className="order-detail-head">
     <div className="order-detail-head-left">
      <div className={"order-channel-icon "+selected.channel}><Icon name={channelPresentation[selected.channel].icon} size={20}/></div>
      <div>
       <h2>{selected.customerName||selected.tableName||"Cliente"}</h2>
       <small>{selected.code} · {labelFor(selected.channel)}{selected.tableName?" · "+selected.tableName:""}</small>
      </div>
     </div>
     <Button className="order-detail-close" onClick={()=>setSelected(null)} aria-label="Cerrar"><Icon name="close" size={18}/></Button>
    </header>

    <div className="order-detail-status">
     <span className={"order-state-tag os-"+stateTone[selected.status]}>{stateLabel[selected.status]}</span>
     <small>{formatCreatedAt(selected.createdAt)}</small>
    </div>

    {(selected.customerPhone||selected.address)&&<div className="order-detail-contact">
     {selected.customerPhone&&<div><Icon name="whatsapp" size={15}/><span>{selected.customerPhone}</span></div>}
     {selected.address&&<div><Icon name="bike" size={15}/><span>{selected.address}</span></div>}
    </div>}

    <div className="order-detail-items">
     <h3>Detalle del pedido</h3>
     {detailLoading?<div className="order-detail-loading">Cargando detalle…</div>:
      selected.items?.map(item=><div className="order-detail-line" key={item.id}>
       <b>{item.qty}×</b>
       <div className="order-detail-line-info">
        <span>{item.name}</span>
        {item.note&&<em>{item.note}</em>}
       </div>
       <strong>{formatMoney(currency,lineTotal(item))}</strong>
      </div>)}
     {!detailLoading&&(!selected.items||selected.items.length===0)&&<div className="order-detail-loading">Sin líneas disponibles.</div>}
    </div>

    <div className="order-detail-totals">
     <div><span>Subtotal</span><b>{formatMoney(currency,selected.subtotal)}</b></div>
     {Number(selected.deliveryFee)>0&&<div><span>Delivery</span><b>{formatMoney(currency,selected.deliveryFee)}</b></div>}
     <div className="order-detail-grand"><span>Total</span><strong>{formatMoney(currency,selected.total)}</strong></div>
    </div>

    <footer className="order-detail-actions">
     {nextAction(selected)&&<Button tone="operational" className="order-detail-primary" disabled={actionId===selected.id} onClick={()=>void advance(selected)}>
      {actionId===selected.id?"Guardando…":nextAction(selected)?.label}<Icon name="chevron" size={16}/>
     </Button>}
     {selected.customerPhone&&<Button className="order-detail-secondary" onClick={()=>contactWhatsApp(selected.customerPhone)}><Icon name="whatsapp" size={16}/>Contactar</Button>}
    </footer>
   </div>
  </aside>}
 </div>;
}
