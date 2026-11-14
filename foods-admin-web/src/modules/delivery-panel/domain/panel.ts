import type {IconName} from "@/design-system";
import type {PanelAction,PanelOrder,PanelOutcome,PanelStage} from "./types";

type Tone="orange"|"blue"|"navy"|"green";

/**
 * Etapa del reparto para la central (`Status` suave, registro): naranja = espera
 * su respuesta; azul = asignado, por recoger; navy = en camino, como el pedido;
 * verde = entregado. Misma semántica que la matriz de Delivery Fudia.
 */
export const stageMeta:Record<PanelStage,{label:string;tone:Tone;icon:IconName}>={
  offer:{label:"Nuevo",tone:"orange",icon:"bell"},
  accepted:{label:"Por recoger",tone:"blue",icon:"store"},
  picked_up:{label:"En camino",tone:"navy",icon:"motorbike"},
  delivered:{label:"Entregado",tone:"green",icon:"circleCheck"},
};

export function groupOrders(orders:PanelOrder[]){
  return {
    offers:orders.filter(order=>order.stage==="offer"),
    active:orders.filter(order=>order.stage==="accepted"||order.stage==="picked_up"),
    delivered:orders.filter(order=>order.stage==="delivered"),
  };
}

/** Ofertas que el panel aún no había mostrado: disparan el aviso sonoro. */
export function newOffers(seen:ReadonlySet<string>,orders:PanelOrder[]){
  return orders.filter(order=>order.stage==="offer"&&!seen.has(order.token));
}

/** Segundos que quedan para responder, descontando lo transcurrido desde la última consulta. */
export function remainingSeconds(order:PanelOrder,receivedAt:number,now:number){
  return Math.max(0,order.expiresInSeconds-Math.floor((now-receivedAt)/1000));
}

export function formatCountdown(seconds:number){
  const safe=Math.max(0,Math.floor(seconds));
  return `${Math.floor(safe/60)}:${String(safe%60).padStart(2,"0")}`;
}

/** El local ya terminó el pedido (o lo despachó): la central puede registrar la recogida. */
export function readyForPickup(order:PanelOrder){
  return order.orderStatus==="listo"||order.orderStatus==="en_camino"||order.orderStatus==="entregado";
}

/** Acción principal de un reparto en curso; la oferta tiene Aceptar y Rechazar. */
export function nextStep(order:PanelOrder):{action:PanelAction;label:string;enabled:boolean;hint:string}|null{
  if(order.stage==="accepted"){
    const ready=readyForPickup(order);
    return {action:"pickup",label:"Recogí el pedido",enabled:ready,hint:ready?"El pedido está listo en el local.":"El local aún lo está preparando. Te avisaremos aquí cuando esté listo."};
  }
  if(order.stage==="picked_up")return {action:"deliver",label:"Entregado",enabled:true,hint:"Confirma cuando el cliente reciba su pedido."};
  return null;
}

type Feedback={tone:"success"|"warning"|"danger";title:string;message:string};

/** Respuesta visible de cada resultado del backend; el panel no inventa estados. */
export function outcomeFeedback(outcome:PanelOutcome,orderCode=""):Feedback{
  const order=orderCode?`El pedido ${orderCode}`:"El pedido";
  switch(outcome){
    case "accepted":case "already_accepted":return {tone:"success",title:"Pedido aceptado",message:`${order} es tuyo. Te avisaremos cuando esté listo para recoger.`};
    case "rejected":return {tone:"success",title:"Pedido rechazado",message:`${order} pasó a otro proveedor.`};
    case "picked_up":return {tone:"success",title:"Recogida registrada",message:`${order} va en camino al cliente.`};
    case "delivered":return {tone:"success",title:"Entrega registrada",message:`${order} quedó entregado.`};
    case "not_ready":return {tone:"warning",title:"Aún no está listo",message:"El local todavía prepara este pedido."};
    case "not_picked_up":return {tone:"warning",title:"Primero registra la recogida",message:"Marca «Recogí el pedido» antes de confirmar la entrega."};
    case "expired":return {tone:"warning",title:"La oferta venció",message:"Pasaron los 3 minutos y el pedido pasó a otro proveedor."};
    case "taken":return {tone:"warning",title:"Ya no está disponible",message:"Otro proveedor tomó este pedido o dejó de ofrecerse."};
    case "cancelled":return {tone:"warning",title:"Pedido cancelado",message:"Este pedido ya no requiere delivery. No lo recojas."};
    case "payment_conflict":return {tone:"danger",title:"Coordina con el local",message:"El local ya registró pagos de este pedido. No lo recojas ni cobres al cliente."};
    default:return {tone:"danger",title:"No encontramos el pedido",message:"Actualiza el panel e inténtalo de nuevo."};
  }
}

/** Indicaciones hacia un punto en la app de mapas del teléfono. */
export function directionsUrl(latitude:number|null,longitude:number|null){
  if(latitude===null||longitude===null)return "";
  return `https://www.google.com/maps/dir/?api=1&destination=${latitude},${longitude}`;
}

/** Teléfono para `tel:`: solo dígitos y el signo inicial. */
export function dialablePhone(phone:string){
  const digits=phone.replace(/[^\d+]/g,"");
  return /^\+?\d{6,15}$/.test(digits)?digits:"";
}
