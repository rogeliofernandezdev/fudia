"use client";
import Link from "next/link";
import {Icon,Status} from "@/design-system";
import {deliveryTabPath} from "../domain/board";
import {dispatchStatusMeta,isDispatchStatus} from "../domain/dispatch-status";
import "./delivery.css";

type DispatchOrder={id:string;deliveryDispatchStatus?:string;deliveryPartnerName?:string;deliveryPartnerFee?:string};

/**
 * Resumen del despacho a las centrales de Delivery Fudia en el bloque de
 * entrega del detalle del pedido: estado y central. Reintentar y Usar delivery
 * propio viven en Delivery › Pedidos. Sin despacho no se muestra nada.
 */
export function OrderDispatchBlock({order,money}:{order:DispatchOrder;money:(value:string)=>string}){
  const status=order.deliveryDispatchStatus;
  if(!isDispatchStatus(status))return null;
  const meta=dispatchStatusMeta[status];
  const partner=status==="own"||status==="cancelled"?"":order.deliveryPartnerName;
  return <div className="order-dispatch" aria-label="Despacho a centrales">
    <div className="order-dispatch-state">
      <Status tone={meta.tone} icon={meta.icon}>{meta.label}</Status>
      {partner&&<b>{partner}</b>}
    </div>
    {order.deliveryPartnerFee&&<p className="order-dispatch-fee">Envío {money(order.deliveryPartnerFee)}: lo cobra la central al cliente.</p>}
    <Link className="order-dispatch-link" href={deliveryTabPath("orders")}>Ver en Delivery<Icon name="arrowUpRight" size={14}/></Link>
  </div>;
}
