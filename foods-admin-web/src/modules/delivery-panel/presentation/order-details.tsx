"use client";

import {useState} from "react";
import {Icon,IconTile,RouteMap} from "@/design-system";
import {dialablePhone,directionsUrl} from "../domain/panel";
import type {CourierPosition,PanelOrder} from "../domain/types";

const coordinates=(latitude:number|null,longitude:number|null)=>latitude===null||longitude===null?null:{lat:latitude,lng:longitude};
export const hour=(value?:string)=>value?new Date(value).toLocaleTimeString(undefined,{hour:"2-digit",minute:"2-digit"}):"";
const quantity=(value:number)=>Number.isInteger(value)?String(value):value.toFixed(2);

/**
 * Datos del reparto que comparten la central y el motorizado: la ruta como línea
 * de tiempo (recolección y entrega unidas por su conector), el contacto del
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
    <ol className="partner-panel-route">
      {!delivered&&<li className="partner-panel-stop is-linked is-pickup">
        <span className="partner-panel-stop-mark"><IconTile name="store" tone="info" appearance="vivid"/></span>
        <div className="partner-panel-stop-copy">
          <p className="partner-panel-stop-label">Recoge en</p>
          <b>{order.locationName}</b>
          {order.locationAddress&&<span>{order.locationAddress}</span>}
          {pickupUrl&&<a href={pickupUrl} target="_blank" rel="noreferrer noopener"><Icon name="arrowUpRight" size={16}/>Cómo llegar</a>}
        </div>
      </li>}
      <li className="partner-panel-stop is-drop">
        <span className="partner-panel-stop-mark"><IconTile name="mapPin" tone="brand" appearance="vivid"/></span>
        <div className="partner-panel-stop-copy">
          <p className="partner-panel-stop-label">Entrega en</p>
          <b>{order.address||"Dirección por confirmar"}</b>
          {order.reference&&<span>Ref.: {order.reference}</span>}
          {!delivered&&order.distanceKm!==null&&<span className="partner-panel-distance"><Icon name="route" size={14}/>{order.distanceKm.toFixed(1)} km desde el local</span>}
          {!delivered&&dropUrl&&<a href={dropUrl} target="_blank" rel="noreferrer noopener"><Icon name="arrowUpRight" size={16}/>Cómo llegar</a>}
        </div>
      </li>
    </ol>
    <div className="partner-panel-contact">
      <IconTile name="user" tone="primary" appearance="vivid"/>
      <div>
        <p>Cliente</p>
        <b>{order.customerName||"Sin nombre"}</b>
        {!delivered&&(phone?<a className="partner-panel-call" href={`tel:${phone}`}><Icon name="phone" size={16}/>Llamar al {order.customerPhone}</a>:order.customerPhone&&<span>{order.customerPhone}</span>)}
        {delivered&&order.deliveredAt&&<span>Entregado a las {hour(order.deliveredAt)}</span>}
      </div>
    </div>
    {!delivered&&<>
      <ul className="partner-panel-items" aria-label="Productos del pedido">{order.items.map((item,index)=><li key={index}><b>{quantity(item.quantity)} ×</b><span>{item.name}{item.note&&<small>{item.note}</small>}</span></li>)}</ul>
      {order.notes&&<p className="partner-panel-note"><Icon name="note" size={16}/>{order.notes}</p>}
    </>}
    <dl className="partner-panel-money">
      <div><dt>Pagas al local</dt><dd>{money(order.subtotal)}</dd></div>
      <div className="partner-panel-money-main"><dt>Cobras al cliente</dt><dd>{money(order.total)}</dd></div>
    </dl>
    {!delivered&&(origin||destination)&&<div className="partner-panel-map">
      <button type="button" className="partner-panel-link is-map" aria-expanded={showMap} onClick={()=>setShowMap(!showMap)}><Icon name="route" size={16}/>{showMap?"Ocultar mapa":"Ver mapa"}</button>
      {showMap&&<RouteMap origin={origin} destination={destination} courier={courier?{lat:courier.latitude,lng:courier.longitude}:null} label={`Ruta del pedido ${order.orderCode}`}/>}
    </div>}
  </>;
}
