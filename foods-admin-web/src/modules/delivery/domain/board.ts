import type {IconName} from "@/design-system";
import type {DeliveryBoardOrder,PickupAlert} from "./types";

/** Pestañas de Delivery; el valor viaja en `?tab=`. */
export type DeliveryTab="providers"|"fees"|"failed";
type TabDefinition={value:DeliveryTab;label:string;icon:IconName};

/**
 * Delivery lo administra Fudia (2026-11-23): la opción del menú es solo de
 * Plataforma. Centrales, tarifas de envío por local e incidencias; los pedidos
 * se gestionan en Pedidos.
 */
export const deliveryTabs:readonly TabDefinition[]=[
  {value:"providers",label:"Proveedores",icon:"motorbike"},
  {value:"fees",label:"Tarifas de envío",icon:"route"},
  {value:"failed",label:"Entregas fallidas",icon:"alert"},
];

/** Pestaña inicial: la pedida en la URL si existe; si no, Proveedores. */
export function deliveryTabFrom(value:string|null|undefined):DeliveryTab{
  return deliveryTabs.find(tab=>tab.value===value)?.value??"providers";
}

/** Ruta a una pestaña de Delivery, para enlaces y redirecciones. */
export function deliveryTabPath(tab:DeliveryTab){
  return `/delivery?tab=${tab}`;
}

/** Un delivery sigue en curso mientras no se entrega, cancela ni registra como fallido. */
export function deliveryInCourse(order:Pick<DeliveryBoardOrder,"status"|"failedDelivery">){
  return !order.failedDelivery&&order.status!=="entregado"&&order.status!=="cancelado";
}

export type BoardFilter="active"|"today";
export function filterBoard(orders:DeliveryBoardOrder[],filter:BoardFilter){
  return filter==="active"?orders.filter(deliveryInCourse):orders;
}

/** Modo de reparto: las centrales de Delivery Fudia o el repartidor del local. */
export function deliveryMode(order:Pick<DeliveryBoardOrder,"dispatchStatus">){
  return order.dispatchStatus===""||order.dispatchStatus==="own"?"Delivery propio":"Delivery Fudia";
}

/** Nivel de la alerta de recogida para el restaurante (tono de atención, nunca libera el pedido). */
export const pickupAlertLabels:Record<Exclude<PickupAlert,"">,{label:string;tone:"orange"|"red"}>={
  warning:{label:"Recogida demorada",tone:"orange"},
  urgent:{label:"Recogida urgente",tone:"orange"},
  critical:{label:"Recogida crítica",tone:"red"},
};

export function waitMinutes(seconds:number){
  return `${Math.max(0,Math.floor(seconds/60))} min`;
}

/** Incidencia visible en el tablero: entrega fallida registrada o el motorizado no pudo entregar. */
export function deliveryIssue(order:Pick<DeliveryBoardOrder,"failedDelivery"|"issueReportedAt">){
  if(order.failedDelivery)return "Entrega fallida";
  if(order.issueReportedAt)return "No pudo entregar";
  return "";
}

/** Teléfono con código de país, agrupado para leerlo: +51 987 654 321. */
export function formatPartnerPhone(digits:string){
  const match=/^(51)(\d{3})(\d{3})(\d{3})$/.exec(digits);
  return match?`+${match[1]} ${match[2]} ${match[3]} ${match[4]}`:digits?`+${digits}`:"";
}
