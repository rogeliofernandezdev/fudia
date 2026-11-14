"use client";

import {useState} from "react";
import {Icon,RouteMap} from "@/design-system";
import {dialablePhone,directionsUrl} from "../domain/panel";
import type {CourierPosition,PanelOrder} from "../domain/types";

const coordinates=(latitude:number|null,longitude:number|null)=>latitude===null||longitude===null?null:{lat:latitude,lng:longitude};
export const hour=(value?:string)=>value?new Date(value).toLocaleTimeString(undefined,{hour:"2-digit",minute:"2-digit"}):"";
const quantity=(value:number)=>Number.isInteger(value)?String(value):value.toFixed(2);

/**
 * Datos del reparto que comparten la central y el motorizado: recojo, entrega,
 * cliente, productos, montos y mapa. Un pedido entregado muestra solo el resumen.
 */
export function OrderDetails({order,courier}:{order:PanelOrder;courier:CourierPosition|null}){
  const [showMap,setShowMap]=useState(false);
  const origin=coordinates(order.locationLatitude,order.locationLongitude);
  const destination=coordinates(order.latitude,order.longitude);
  const pickupUrl=directionsUrl(order.locationLatitude,order.locationLongitude);
  const dropUrl=directionsUrl(order.latitude,order.longitude);
  const phone=dialablePhone(order.customerPhone);
  const money=(amount:string)=>`${order.currencySymbol} ${amount}`.trim();
  const delivered=order.stage==="delivered";
  return <>
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
  </>;
}
