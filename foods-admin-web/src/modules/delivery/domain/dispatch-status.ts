import type {IconName} from "@/design-system";
import type {DeliveryLink,DispatchStatus} from "./types";

type Tone="green"|"blue"|"orange"|"violet"|"gray";

/**
 * Estado del despacho a las centrales (registro, no estado operativo del
 * pedido): `Status` suave según la matriz de 02_DESIGN_SYSTEM.md › Delivery Fudia.
 * Violeta = oferta en curso en el panel de la central; azul = asignado;
 * verde = recogido y entregado (la etiqueta los distingue);
 * naranja = requiere acción de caja; gris = fuera de las centrales.
 */
export const dispatchStatusMeta:Record<DispatchStatus,{label:string;tone:Tone;icon:IconName}>={
  searching:{label:"Buscando central",tone:"violet",icon:"motorbike"},
  accepted:{label:"Central asignada",tone:"blue",icon:"motorbike"},
  picked_up:{label:"Recogido por la central",tone:"green",icon:"motorbike"},
  delivered:{label:"Entregado por la central",tone:"green",icon:"circleCheck"},
  unassigned:{label:"Sin central",tone:"orange",icon:"alert"},
  own:{label:"Delivery propio",tone:"gray",icon:"store"},
  cancelled:{label:"Despacho cancelado",tone:"gray",icon:"circleSlash"},
};

export function isDispatchStatus(value:unknown):value is DispatchStatus{
  return typeof value==="string"&&value in dispatchStatusMeta;
}

/**
 * Caja puede dejar el pedido para su propio repartidor solo mientras ninguna
 * central lo aceptó; después lo coordina Fudia. Los reintentos a las centrales
 * son automáticos (2026-11-23).
 */
export function canUseOwnDelivery(status:DispatchStatus){return status==="searching"||status==="unassigned";}

export function hasDeliveryLink(link:DeliveryLink|undefined):link is DeliveryLink{return Boolean(link?.token&&link.active);}
