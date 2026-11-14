"use client";

import {useQuery} from "@tanstack/react-query";
import {IconCheck,IconX} from "@tabler/icons-react";
import {RouteMap,Skeleton} from "@/design-system";
import {courierNote,showsMap,trackingHeadline,trackingSteps} from "../domain/tracking";
import type {TrackingPoint} from "../domain/types";
import {loadChatTracking} from "../infrastructure/chat-api";

const point=(value:TrackingPoint|null)=>value?{lat:value.latitude,lng:value.longitude}:null;

/**
 * Seguimiento del pedido registrado en este chat: en qué paso va y, en un
 * delivery, el mapa con el local, la entrega y el repartidor. Solo consulta
 * mientras está abierto.
 */
export function OrderTracking({qr,onClose}:{qr:string;onClose:()=>void}) {
  const tracking=useQuery({queryKey:["web-chat-tracking",qr],queryFn:()=>loadChatTracking(qr),retry:false,staleTime:0,refetchInterval:5000});
  const data=tracking.data;
  const steps=data?.available?trackingSteps(data):[];
  const note=data?.available?courierNote(data,tracking.dataUpdatedAt):"";
  const delivery=data?.delivery;
  return <section className="restaurant-chat-tracking" aria-label="Seguimiento de tu pedido">
    <div className="restaurant-chat-section-heading"><h2>Tu pedido{data?.code&&<span className="restaurant-chat-cart-count">{data.code}</span>}</h2><button className="restaurant-chat-icon" type="button" aria-label="Ocultar seguimiento" onClick={onClose}><IconX size={20}/></button></div>
    {tracking.isPending?<div className="restaurant-chat-tracking-loading" aria-busy="true"><Skeleton className="restaurant-chat-skeleton restaurant-chat-loading-line"/><Skeleton className="restaurant-chat-skeleton restaurant-chat-tracking-map-skeleton"/><span className="sr-only">Consultando tu pedido</span></div>
    :!data?<div className="restaurant-chat-system" role="alert"><p>{tracking.error?.message??"No pudimos consultar tu pedido."}</p><button type="button" onClick={()=>void tracking.refetch()}>Volver a consultar</button></div>
    :!data.available?<p className="restaurant-chat-system">Aún no tienes un pedido registrado para seguir.</p>
    :<>
      <p className="restaurant-chat-tracking-headline" role="status">{trackingHeadline(data)}</p>
      {steps.length>0&&<ol className="restaurant-chat-tracking-steps" aria-label="Avance del pedido">{steps.map(step=><li key={step.label} className={`is-${step.state}`} aria-current={step.state==="current"?"step":undefined}><i aria-hidden="true">{step.state==="done"&&<IconCheck size={14}/>}</i><span>{step.label}</span></li>)}</ol>}
      {delivery&&showsMap(data)&&<RouteMap origin={point(delivery.origin)} destination={point(delivery.destination)} courier={point(delivery.courier)} label="Mapa del recorrido de tu pedido"/>}
      {delivery&&<dl className="restaurant-chat-tracking-facts">
        {delivery.originName&&<div><dt>Sale de</dt><dd>{delivery.originName}</dd></div>}
        {delivery.destinationAddress&&<div><dt>Entrega en</dt><dd>{delivery.destinationAddress}</dd></div>}
        {delivery.partnerName&&<div><dt>Reparto</dt><dd>{delivery.partnerName}</dd></div>}
      </dl>}
      {note&&<p className="restaurant-chat-cart-hint" role="status">{note}</p>}
      {tracking.isError&&<p className="restaurant-chat-cart-hint" role="alert">Sin conexión: mostramos el último estado conocido.</p>}
    </>}
  </section>;
}
