import type {PanelOrder} from "./types";
import {needsPickupDecision} from "./panel";

export type DispatchFilter="all"|"unassigned"|"on_route"|"delivered";
export const normalizedName=(value:string)=>value.trim().replace(/\s+/g," ").normalize("NFD").replace(/[\u0300-\u036f]/g,"").toLocaleLowerCase("es");
export const needsAssignment=(order:PanelOrder)=>order.stage==="accepted"&&!order.courierName.trim();
export const needsAttention=(order:PanelOrder)=>needsPickupDecision(order)||(order.stage==="picked_up"&&Boolean(order.issueReportedAt));

export const DISPATCH_BATCH_SIZE=10;

/** Ventana acumulativa de la respuesta recibida; no pagina la consulta al servidor. */
export function dispatchWindow<T>(items:T[],requestedCount=DISPATCH_BATCH_SIZE){
  const count=Number.isFinite(requestedCount)?Math.max(DISPATCH_BATCH_SIZE,Math.floor(requestedCount)):DISPATCH_BATCH_SIZE;
  const visible=items.slice(0,count);
  return {items:visible,total:items.length,hasMore:visible.length<items.length};
}

/** Prioriza incidencias y ofertas sin alterar la colección recibida del API. */
export function dispatchOrders(orders:PanelOrder[],filter:DispatchFilter,search:string,attentionOnly=false,courierName=""){
  const term=normalizedName(search);
  const priority=(order:PanelOrder)=>needsAttention(order)?0:order.stage==="offer"?1:needsAssignment(order)?2:3;
  return orders.filter(order=>{
    if(filter==="delivered"?order.stage!=="delivered":order.stage==="delivered")return false;
    if(filter==="unassigned"&&!needsAssignment(order))return false;
    if(filter==="on_route"&&order.stage!=="picked_up")return false;
    if(attentionOnly&&!needsAttention(order))return false;
    if(courierName&&normalizedName(order.courierName)!==normalizedName(courierName))return false;
    return !term||[order.orderCode,order.customerName,order.organizationName,order.locationName,order.address,order.courierName].some(value=>normalizedName(value).includes(term));
  }).sort((a,b)=>filter==="delivered"?(b.deliveredAt??"").localeCompare(a.deliveredAt??""):priority(a)-priority(b));
}

/** Carga observada por nombre, sin inventar un catálogo ni disponibilidad de flota. */
export function courierWorkload(orders:PanelOrder[]){
  const couriers=new Map<string,{name:string;collecting:number;onRoute:number}>();
  for(const order of orders){
    if(!["accepted","picked_up"].includes(order.stage)||!order.courierName.trim())continue;
    const key=normalizedName(order.courierName);
    const courier=couriers.get(key)??{name:order.courierName.trim(),collecting:0,onRoute:0};
    if(order.stage==="accepted")courier.collecting++;else courier.onRoute++;
    couriers.set(key,courier);
  }
  return [...couriers.values()].sort((a,b)=>a.name.localeCompare(b.name,"es"));
}
