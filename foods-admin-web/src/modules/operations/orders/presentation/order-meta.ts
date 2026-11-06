import type {IconName,IconTileTone,TabTone} from "@/design-system";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";

/** Owner: operations/orders — presentación compartida por Pedidos y Salón. */
export const channelIcons:Record<string,IconName>={salon:"utensils",mostrador:"store",recojo:"box",delivery:"truck",whatsapp:"share"};

/** Color de identidad de cada canal en pestañas, con la misma combinación viva aprobada:
 * azul Todos, índigo Salón, naranja Recojo, verde Delivery y violeta WhatsApp. */
export const channelTones:Record<string,TabTone>={salon:"indigo",recojo:"orange",mostrador:"orange",delivery:"green",whatsapp:"violet"};
export const channelTone=(channel:string):TabTone=>channelTones[channel]??"primary";

/** Origen visible del pedido: la venta por mostrador se distingue dentro de Recojo en local. */
export const originLabel=(origin?:string)=>origin==="counter"?"Mostrador":"";

/** Tipo de atención del canal guardado: un pedido QR por WhatsApp se atiende en Salón. */
export const serviceChannel=(channel:string)=>channel==="whatsapp"?"salon":channel==="mostrador"?"recojo":channel;

export type OrderStatusTone="green"|"blue"|"orange"|"purple"|"gray"|"navy"|"red";
/** Preparando = naranja y Por entregar = morado, como los carriles de Cocina; Entregado siempre verde; el violeta queda para WhatsApp. */
export const statusMeta:Record<string,{label:string;tone:OrderStatusTone}>={
  nuevo:{label:"Nuevo",tone:"gray"},
  confirmado:{label:"Confirmado",tone:"blue"},
  preparando:{label:"Preparando",tone:"orange"},
  listo:{label:"Por entregar",tone:"purple"},
  en_camino:{label:"En camino",tone:"navy"},
  entregado:{label:"Entregado",tone:"green"},
  cancelado:{label:"Cancelado",tone:"red"},
};
export const orderStatus=(status:string)=>statusMeta[status]??{label:status,tone:"gray" as const};

/** Los nombres de preparación siguen viniendo del API; solo se centraliza su tono. */
export function serviceStatusTone(status:string):OrderStatusTone{
  return status==="preparando"?"orange":status==="listo"?"purple":status==="entregado"?"green":status==="cancelado"?"red":["pendiente","nuevo","confirmado"].includes(status)?"blue":"gray";
}

/** Estado de preparación de una línea: el común si todas coinciden; si no, lo que más falta (preparando > pendiente > listo). Sin preparaciones, el del pedido. */
export function lineServiceStatus(statuses:string[],fallback:string){
  if(!statuses.length)return fallback;
  if(new Set(statuses).size===1)return statuses[0];
  if(statuses.includes("preparando"))return "preparando";
  const pending=statuses.find(status=>["pendiente","nuevo","confirmado"].includes(status));
  if(pending)return pending;
  return statuses.find(status=>status==="listo")??statuses[0];
}

const stateTiles:Record<OrderStatusTone,IconTileTone>={orange:"warning",purple:"purple",green:"brand",blue:"indigo",red:"danger",gray:"neutral",navy:"navy"};
/** Icono sólido de la línea: misma familia que su `Status`, glifo blanco. */
export function serviceStateLook(status:string):{icon:IconName;tone:IconTileTone}{
  const icon:IconName=status==="preparando"?"flame":status==="listo"?"bell":status==="entregado"?"check":status==="cancelado"?"cancel":status==="en_camino"?"motorbike":"clock";
  // Por entregar morado y Entregado verde (decisión 2026-11-05), igual que su `Status`.
  return {icon,tone:status==="en_camino"?"navy":stateTiles[serviceStatusTone(status)]};
}

/** Estado mostrado en una línea de producto (pedido, venta, cobro): icono, etiqueta y destino de sus preparaciones reales. */
export function lineServiceState(services:{orderItemId:string;status:string;statusLabel:string;destinationLabel:string}[],itemId:string,fallbackStatus:string){
  const own=services.filter(service=>service.orderItemId===itemId);
  const status=lineServiceStatus(own.map(service=>service.status),fallbackStatus);
  return {look:serviceStateLook(status),label:own.find(service=>service.status===status)?.statusLabel??"",destination:[...new Set(own.map(service=>service.destinationLabel).filter(Boolean))].join(" · ")};
}

/** Fecha y hora del registro en la zona del local; nunca un tiempo relativo. */
export function registeredAt(iso:string,country?:string,timeZone?:string){
  return formatRegionalDateTime(iso,{country,timeZone},{dateStyle:"medium",timeStyle:"short"});
}

export function parseIsoDate(iso:string):Date|null{
  if(!iso)return null;
  const normalized=iso.replace(/([+-]\d{2})$/,"$1:00");
  const date=new Date(normalized);
  if(!isNaN(date.getTime()))return date;
  const fallback=new Date(iso);
  return isNaN(fallback.getTime())?null:fallback;
}

/** Tiempo de apertura de una mesa: relativo durante el día, fecha regional después. */
export function openedAgo(iso:string,country?:string,timeZone?:string){
  const date=parseIsoDate(iso);if(!date)return"—";
  const minutes=Math.floor((Date.now()-date.getTime())/60000);
  if(minutes<1)return"Justo ahora";if(minutes<60)return`Hace ${minutes} min`;
  const hours=Math.floor(minutes/60);if(hours<24)return`Hace ${hours} h`;
  return registeredAt(date.toISOString(),country,timeZone);
}

/** La comanda se edita solo antes de prepararse y sin pagos registrados. */
export function isEditableOrder(order:{status:string;paidAmount?:string}){
  return (order.status==="nuevo"||order.status==="confirmado")&&Number(order.paidAmount??0)<=0.00001;
}

export function tableTitle(tableName?:string){
  return tableName?(/^mesa\b/i.test(tableName)?tableName:`Mesa ${tableName}`):"Mesa";
}
