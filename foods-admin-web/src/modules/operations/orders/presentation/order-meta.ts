import type {IconName} from "@/design-system";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";

/** Owner: operations/orders — presentación compartida por Pedidos y Salón. */
export const channelIcons:Record<string,IconName>={salon:"utensils",mostrador:"store",recojo:"box",delivery:"truck",whatsapp:"share"};

export type OrderStatusTone="green"|"blue"|"orange"|"gray"|"violet"|"teal"|"red";
export const statusMeta:Record<string,{label:string;tone:OrderStatusTone}>={
  nuevo:{label:"Nuevo",tone:"gray"},
  confirmado:{label:"Confirmado",tone:"blue"},
  preparando:{label:"Preparando",tone:"violet"},
  listo:{label:"Listo para entregar",tone:"green"},
  en_camino:{label:"En camino",tone:"orange"},
  entregado:{label:"Entregado",tone:"teal"},
  cancelado:{label:"Cancelado",tone:"red"},
};
export const orderStatus=(status:string)=>statusMeta[status]??{label:status,tone:"gray" as const};

/** Los nombres de preparación siguen viniendo del API; solo se centraliza su tono. */
export function serviceStatusTone(status:string):OrderStatusTone{
  return status==="preparando"?"violet":status==="listo"?"green":status==="entregado"?"teal":status==="cancelado"?"red":["pendiente","nuevo","confirmado"].includes(status)?"orange":"gray";
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
