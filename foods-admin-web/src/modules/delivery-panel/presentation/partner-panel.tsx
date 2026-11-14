"use client";

import {useEffect,useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,EmptyState,ErrorState,Icon,RouteMap,Skeleton,Status} from "@/design-system";
import {useFeedback} from "@/providers";
import {dialablePhone,directionsUrl,formatCountdown,groupOrders,nextStep,outcomeFeedback,remainingSeconds,stageMeta} from "../domain/panel";
import type {CourierPosition,PanelAction,PanelOrder} from "../domain/types";
import {loadPartnerPanel,sendPanelAction} from "../infrastructure/panel-api";
import {useCourierLocation,type LocationStatus} from "./use-courier-location";
import {requestAlertPermission,useOfferAlert} from "./use-offer-alert";
import "./partner-panel.css";

const POLL_MS=5_000;
const coordinates=(latitude:number|null,longitude:number|null)=>latitude===null||longitude===null?null:{lat:latitude,lng:longitude};
const hour=(value?:string)=>value?new Date(value).toLocaleTimeString(undefined,{hour:"2-digit",minute:"2-digit"}):"";
const quantity=(value:number)=>Number.isInteger(value)?String(value):value.toFixed(2);

const locationCopy:Record<LocationStatus,string>={
  off:"Se comparte solo mientras llevas un pedido.",
  locating:"Buscando tu ubicación…",
  active:"El cliente ve por dónde vas.",
  denied:"Permite la ubicación en el navegador para compartirla.",
  unsupported:"Este navegador no permite compartir la ubicación.",
  error:"No pudimos obtener tu ubicación. Revisa el GPS.",
};

/**
 * Preferencia del dispositivo (avisos, ubicación). Se lee al crear el estado: el
 * servidor solo pinta la carga, así que el primer render del cliente no depende de ella.
 */
function useStoredFlag(name:string):[boolean,(value:boolean)=>void]{
  const [value,setValue]=useState(()=>{
    if(typeof window==="undefined")return false;
    try{return window.localStorage.getItem(name)==="1"}catch{return false}
  });
  return [value,next=>{setValue(next);try{window.localStorage.setItem(name,next?"1":"0")}catch{/* almacenamiento bloqueado */}}];
}

function OrderCard({order,remaining,courier,busy,onAct}:{order:PanelOrder;remaining:number;courier:CourierPosition|null;busy:boolean;onAct:(action:PanelAction)=>void}){
  const [showMap,setShowMap]=useState(false);
  const meta=stageMeta[order.stage];
  const step=nextStep(order);
  const origin=coordinates(order.locationLatitude,order.locationLongitude);
  const destination=coordinates(order.latitude,order.longitude);
  const pickupUrl=directionsUrl(order.locationLatitude,order.locationLongitude);
  const dropUrl=directionsUrl(order.latitude,order.longitude);
  const phone=dialablePhone(order.customerPhone);
  const money=(amount:string)=>`${order.currencySymbol} ${amount}`.trim();
  const delivered=order.stage==="delivered";
  return <article className={`partner-panel-card is-${order.stage}`} aria-label={`Pedido ${order.orderCode}`}>
    <header className="partner-panel-card-head">
      <div><b>{order.orderCode}</b><span>{order.organizationName}</span></div>
      <Status tone={meta.tone} icon={meta.icon}>{meta.label}</Status>
    </header>
    {order.stage==="offer"&&<p className="partner-panel-timer" role="timer" aria-live="off"><Icon name="clock" size={16}/>Responde en <b>{formatCountdown(remaining)}</b></p>}
    <dl className="partner-panel-route">
      {!delivered&&<div>
        <dt><Icon name="store" size={16}/>Recoge en</dt>
        <dd><b>{order.locationName}</b>{order.locationAddress&&<span>{order.locationAddress}</span>}{pickupUrl&&<a href={pickupUrl} target="_blank" rel="noreferrer noopener">Cómo llegar</a>}</dd>
      </div>}
      <div>
        <dt><Icon name="mapPin" size={16}/>Entrega en</dt>
        <dd><b>{order.address||"Dirección por confirmar"}</b>{order.reference&&<span>Ref.: {order.reference}</span>}{!delivered&&order.distanceKm!==null&&<span>{order.distanceKm.toFixed(1)} km desde el local</span>}{!delivered&&dropUrl&&<a href={dropUrl} target="_blank" rel="noreferrer noopener">Cómo llegar</a>}</dd>
      </div>
      <div>
        <dt><Icon name="user" size={16}/>Cliente</dt>
        <dd><b>{order.customerName||"Sin nombre"}</b>{!delivered&&(phone?<a href={`tel:${phone}`}>Llamar al {order.customerPhone}</a>:order.customerPhone&&<span>{order.customerPhone}</span>)}{delivered&&order.deliveredAt&&<span>Entregado a las {hour(order.deliveredAt)}</span>}</dd>
      </div>
    </dl>
    {!delivered&&<>
      <ul className="partner-panel-items" aria-label="Productos del pedido">{order.items.map((item,index)=><li key={index}><b>{quantity(item.quantity)} ×</b><span>{item.name}{item.note&&<small>{item.note}</small>}</span></li>)}</ul>
      {order.notes&&<p className="partner-panel-note"><Icon name="note" size={16}/>{order.notes}</p>}
    </>}
    <dl className="partner-panel-money">
      <div><dt>Pagas al local</dt><dd>{money(order.subtotal)}</dd></div>
      <div><dt>Cobras al cliente</dt><dd>{money(order.total)}</dd></div>
    </dl>
    {!delivered&&(origin||destination)&&<div className="partner-panel-map">
      <button type="button" className="partner-panel-link" aria-expanded={showMap} onClick={()=>setShowMap(!showMap)}><Icon name="route" size={16}/>{showMap?"Ocultar mapa":"Ver mapa"}</button>
      {showMap&&<RouteMap origin={origin} destination={destination} courier={courier?{lat:courier.latitude,lng:courier.longitude}:null} label={`Ruta del pedido ${order.orderCode}`}/>}
    </div>}
    {order.stage==="offer"&&<footer className="partner-panel-actions">
      <Button kind="secondary" disabled={busy} onClick={()=>onAct("reject")}>Rechazar</Button>
      <Button icon="check" disabled={busy} onClick={()=>onAct("accept")}>Aceptar</Button>
    </footer>}
    {step&&<footer className="partner-panel-actions is-single">
      <p>{step.hint}</p>
      <Button icon={step.action==="pickup"?"shoppingBag":"circleCheck"} disabled={busy||!step.enabled} onClick={()=>onAct(step.action)}>{step.label}</Button>
    </footer>}
  </article>;
}

/**
 * Panel del proveedor de Delivery Fudia. La central lo abre con su enlace
 * privado: recibe los pedidos, confirma la atención, la recogida y la entrega,
 * y comparte su ubicación para el seguimiento del cliente.
 */
export function PartnerPanel({token}:{token:string}){
  const {notify}=useFeedback();
  const client=useQueryClient();
  const queryKey=["delivery-partner-panel",token];
  const panel=useQuery({queryKey,queryFn:()=>loadPartnerPanel(token),retry:false,staleTime:0,refetchIntervalInBackground:true,refetchInterval:query=>query.state.error?POLL_MS*6:POLL_MS});
  const [alerts,setAlerts]=useStoredFlag("fudia-panel-alerts");
  const [sharing,setSharing]=useStoredFlag("fudia-panel-location");
  const [confirming,setConfirming]=useState<{order:PanelOrder;action:"reject"|"deliver"}|null>(null);
  const [showDelivered,setShowDelivered]=useState(false);
  const [now,setNow]=useState(0);
  const groups=groupOrders(panel.data?.orders??[]);
  const location=useCourierLocation(token,sharing&&groups.active.length>0);
  useOfferAlert(panel.data?.orders,alerts);
  // El reloj de las ofertas avanza entre consultas; sin ofertas no hay temporizador.
  const ticking=groups.offers.length>0;
  useEffect(()=>{
    if(!ticking)return;
    const timer=window.setInterval(()=>setNow(Date.now()),1000);
    return ()=>window.clearInterval(timer);
  },[ticking]);
  const act=useMutation({
    mutationFn:({order,action}:{order:PanelOrder;action:PanelAction})=>sendPanelAction(token,action,order.token),
    onSuccess:(result,{order})=>{
      setConfirming(null);
      const feedback=outcomeFeedback(result.outcome,order.orderCode);
      notify({tone:feedback.tone==="warning"?"info":feedback.tone,title:feedback.title,message:feedback.message});
      void client.invalidateQueries({queryKey});
    },
    onError:error=>{setConfirming(null);notify({tone:"danger",title:"No se pudo registrar",message:error.message})},
  });
  // Rechazar y entregar no se deshacen: piden confirmación. Aceptar y recoger se aplican al toque.
  const request=(order:PanelOrder,action:PanelAction)=>action==="reject"||action==="deliver"?setConfirming({order,action}):act.mutate({order,action});
  const card=(order:PanelOrder)=><OrderCard key={order.token} order={order} busy={act.isPending} courier={location.position}
    remaining={remainingSeconds(order,panel.dataUpdatedAt,now||panel.dataUpdatedAt)} onAct={action=>request(order,action)}/>;

  if(panel.isPending)return <main className="partner-panel-stage"><section className="partner-panel" aria-busy="true">
    <header className="partner-panel-header"><div><h1>Panel de reparto</h1><p>Cargando tus pedidos…</p></div></header>
    <div className="partner-panel-body"><Skeleton height="var(--size-56)" radius="var(--radius-tile)"/><Skeleton height="var(--size-280)" radius="var(--radius-card)"/><Skeleton height="var(--size-280)" radius="var(--radius-card)"/><span className="sr-only">Cargando pedidos</span></div>
  </section></main>;
  if(!panel.data)return <main className="partner-panel-stage"><section className="partner-panel">
    <header className="partner-panel-header"><div><h1>Panel de reparto</h1><p>Delivery Fudia</p></div></header>
    <div className="partner-panel-body"><ErrorState title="No pudimos abrir el panel" text={panel.error?.message} retrying={panel.isFetching} onRetry={()=>void panel.refetch()}/></div>
  </section></main>;

  const data=panel.data;
  const empty=!groups.offers.length&&!groups.active.length;
  return <main className="partner-panel-stage"><section className="partner-panel" aria-label={`Panel de reparto de ${data.partnerName}`}>
    <header className="partner-panel-header">
      <span className="partner-panel-badge"><Icon name="motorbike" size={24}/></span>
      <div><h1>{data.partnerName}</h1><p>Panel de reparto · Delivery Fudia</p></div>
    </header>
    <div className="partner-panel-body">
      <div className="partner-panel-controls">
        <button type="button" className="partner-panel-toggle" aria-pressed={alerts} onClick={()=>{if(!alerts)requestAlertPermission();setAlerts(!alerts)}}>
          <Icon name="bell" size={18}/><span><b>Avisos con sonido</b><small>{alerts?"Sonará cada pedido nuevo.":"Actívalos para no perder pedidos."}</small></span><i aria-hidden="true"/>
        </button>
        <button type="button" className="partner-panel-toggle" aria-pressed={sharing} onClick={()=>setSharing(!sharing)}>
          <Icon name="mapPin" size={18}/><span><b>Compartir ubicación</b><small>{sharing?locationCopy[location.status]:"Actívala para que el cliente siga su pedido."}</small></span><i aria-hidden="true"/>
        </button>
      </div>
      {!data.active&&<p className="partner-panel-banner" role="status"><Icon name="info" size={18}/>Tu cuenta está pausada: no recibirás pedidos nuevos. Puedes terminar los que ya tienes.</p>}
      {panel.isError&&<p className="partner-panel-banner is-danger" role="alert"><Icon name="alert" size={18}/>Sin conexión con Fudia. Los pedidos pueden estar desactualizados; reintentando…</p>}
      {groups.offers.length>0&&<section aria-labelledby="partner-panel-offers"><h2 id="partner-panel-offers">Pedidos nuevos<span>{groups.offers.length}</span></h2>{groups.offers.map(card)}</section>}
      {groups.active.length>0&&<section aria-labelledby="partner-panel-active"><h2 id="partner-panel-active">En curso<span>{groups.active.length}</span></h2>{groups.active.map(card)}</section>}
      {empty&&<EmptyState icon="motorbike" title="Sin pedidos por ahora" text="Deja esta pantalla abierta: los pedidos nuevos aparecen aquí al instante."/>}
      {groups.delivered.length>0&&<section aria-labelledby="partner-panel-delivered">
        <h2 id="partner-panel-delivered">Entregados hoy<span>{groups.delivered.length}</span></h2>
        <button type="button" className="partner-panel-link" aria-expanded={showDelivered} onClick={()=>setShowDelivered(!showDelivered)}>{showDelivered?"Ocultar entregados":"Ver entregados"}</button>
        {showDelivered&&groups.delivered.map(card)}
      </section>}
    </div>
  </section>
  <ConfirmDialog open={confirming?.action==="reject"} tone="danger" title="Rechazar este pedido" description="El pedido pasará a otro proveedor y ya no podrás tomarlo." subject={confirming?{label:"Pedido",name:confirming.order.orderCode}:undefined}
    confirmLabel="Rechazar pedido" pending={act.isPending} onCancel={()=>setConfirming(null)} onConfirm={()=>confirming&&act.mutate(confirming)}/>
  <ConfirmDialog open={confirming?.action==="deliver"} tone="success" title="Confirmar la entrega" description="Confirma solo cuando el cliente ya recibió su pedido." subject={confirming?{label:"Pedido",name:confirming.order.orderCode}:undefined}
    confirmLabel="Confirmar entrega" pending={act.isPending} onCancel={()=>setConfirming(null)} onConfirm={()=>confirming&&act.mutate(confirming)}/>
  </main>;
}
