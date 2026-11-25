import type {IconName} from "@/design-system";
import type {CourierAction,PanelAction,PanelOrder,PanelOutcome,PanelStage,PickupAlert} from "./types";

type Tone="orange"|"blue"|"green";

/**
 * Etapa del reparto para la central (`Status` suave, registro): naranja = espera
 * su respuesta; azul = asignado, por recoger y en camino (información);
 * verde = entregado. Misma semántica que la matriz de Delivery Fudia.
 */
export const stageMeta:Record<PanelStage,{label:string;tone:Tone;icon:IconName}>={
  offer:{label:"Nuevo",tone:"orange",icon:"bell"},
  accepted:{label:"Por recoger",tone:"blue",icon:"store"},
  picked_up:{label:"En camino",tone:"blue",icon:"motorbike"},
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

/** El motorizado recoge solo cuando cocina terminó y caja confirmó el pago completo. */
export function readyForPickup(order:PanelOrder){
  return order.paymentConfirmed===true&&["listo","en_camino","entregado"].includes(order.orderStatus);
}

type Step={action:CourierAction;label:string;enabled:boolean;hint:string};

/**
 * Acciones del motorizado en su enlace: «Recogí el pedido» cuando está listo y
 * pagado; en camino, «Entregado» y, como alternativa, «No pude entregar». Tras
 * reportar el problema, «Entregado» sigue disponible por si el cliente aparece.
 */
export function courierSteps(order:PanelOrder):{primary:Step;secondary:Step|null}|null{
  if(order.stage==="accepted"){
    const ready=readyForPickup(order);
    const preparing=!["listo","en_camino","entregado"].includes(order.orderStatus);
    return {primary:{action:"pickup",label:"Recogí el pedido",enabled:ready,hint:ready?"Pedido listo y pago confirmado por caja.":preparing?"El local aún lo está preparando. Esta pantalla se actualiza sola.":"Paga el pedido en caja; cuando lo confirme podrás registrar la recogida."},secondary:null};
  }
  if(order.stage==="picked_up"){
    const reported=Boolean(order.issueReportedAt);
    return {
      primary:{action:"deliver",label:"Entregado",enabled:true,hint:reported?"Si el cliente aparece, entrégalo y confírmalo aquí.":"Confirma cuando el cliente reciba su pedido."},
      secondary:reported?null:{action:"not_delivered",label:"No pude entregar",enabled:true,hint:"Úsalo solo después de llamar al cliente y volver a intentarlo."},
    };
  }
  return null;
}

/** Nombre del motorizado sin espacios repetidos; vacío si no cumple 2 a 60 caracteres. */
export function courierNameValue(value:string){
  const name=value.trim().replace(/\s+/g," ");
  return name.length>=2&&name.length<=60?name:"";
}

/** Enlace privado que la central comparte con su motorizado. */
export function courierLink(origin:string,courierToken:string){
  return /^[a-f0-9]{32}$/.test(courierToken)?`${origin.replace(/\/$/,"")}/delivery-courier/${courierToken}`:"";
}

/** Liberar solo antes de la recogida: después, el motorizado ya tiene la comida. */
export const canRelease=(order:PanelOrder)=>order.stage==="accepted";

/** Minutos completos de espera, en una línea («4 min»). */
export function waitLabel(seconds:number){
  return `${Math.max(0,Math.floor(seconds/60))} min`;
}

/**
 * Alerta de recogida (cuenta desde que cocina marca listo o desde el último
 * cambio de motorizado). La central decide; el panel nunca libera por tiempo.
 */
export const pickupAlertMeta:Record<Exclude<PickupAlert,"">,{label:string;tone:"warning"|"danger";title:string;message:string}>={
  warning:{label:"Aviso",tone:"warning",title:"El motorizado aún no llega",message:"El pedido está listo y espera la recogida."},
  urgent:{label:"Urgente",tone:"warning",title:"Recogida urgente",message:"El pedido se está enfriando. Confirma con el motorizado o cambia de responsable."},
  critical:{label:"Crítica",tone:"danger",title:"Recogida crítica",message:"Avisamos también al restaurante. Decide si mantienes, cambias o liberas el pedido."},
};

/** La central debe decidir: hay alerta, sigue por recoger y no eligió «Mantener» en este nivel. */
export function needsPickupDecision(order:PanelOrder){
  return order.stage==="accepted"&&Boolean(order.pickupAlert)&&!order.pickupAlertAcknowledged;
}

/** Segundos para registrar la entrega fallida, descontando el tiempo desde la última consulta. */
export function failCountdown(order:PanelOrder,receivedAt:number,now:number){
  if(!order.issueReportedAt)return null;
  return Math.max(0,(order.failAvailableInSeconds??0)-Math.floor((now-receivedAt)/1000));
}

/** Claves de lo que debe sonar en el panel de la central: alertas críticas sin decidir y reportes del motorizado. */
export function urgentKeys(orders:PanelOrder[]){
  return orders.flatMap(order=>[
    ...(needsPickupDecision(order)&&order.pickupAlert==="critical"?[`${order.token}:critical`]:[]),
    ...(order.stage==="picked_up"&&order.issueReportedAt?[`${order.token}:issue:${order.issueReportedAt}`]:[]),
  ]);
}

type Feedback={tone:"success"|"warning"|"danger";title:string;message:string};

/** Respuesta visible de cada resultado del backend; el panel no inventa estados. */
export function outcomeFeedback(outcome:PanelOutcome,orderCode="",action?:PanelAction|CourierAction):Feedback{
  const order=orderCode?`El pedido ${orderCode}`:"El pedido";
  switch(outcome){
    case "accepted":case "already_accepted":return {tone:"success",title:"Pedido aceptado",message:`${order} es tuyo. Registra al motorizado para generar su enlace.`};
    case "rejected":return {tone:"success",title:"Pedido rechazado",message:`${order} ya no aparecerá en tu panel.`};
    case "courier_assigned":return {tone:"success",title:"Enlace listo",message:"Compártelo con el motorizado: ahí registra la recogida y la entrega."};
    case "released":return {tone:"success",title:"Pedido liberado",message:`${order} se ofrecerá a otras centrales. El enlace del motorizado dejó de funcionar.`};
    case "kept":return {tone:"success",title:"Mantienes al motorizado",message:"Te avisaremos si la espera continúa."};
    case "picked_up":return {tone:"success",title:"Recogida registrada",message:`${order} va en camino al cliente.`};
    case "delivered":return {tone:"success",title:"Entrega registrada",message:`${order} quedó entregado.`};
    case "issue_reported":return {tone:"success",title:"Aviso enviado",message:"Fudia avisó a tu central y pidió al cliente que se comunique."};
    case "delivery_failed":return {tone:"success",title:"Entrega fallida registrada",message:"Fudia conserva los datos y montos para gestionar el reclamo."};
    case "not_ready":return {tone:"warning",title:"Aún no está listo",message:"El local todavía prepara este pedido."};
    case "payment_pending":return {tone:"warning",title:"Pago pendiente de confirmar",message:"Espera a que caja confirme el pago completo para recoger el pedido."};
    case "not_picked_up":return {tone:"warning",title:"Primero registra la recogida",message:"Marca «Recogí el pedido» antes de confirmar la entrega."};
    case "already_picked_up":return {tone:"warning",title:"Ya se recogió",message:"El motorizado ya tiene el pedido; no se puede liberar."};
    case "not_reported":return {tone:"warning",title:"Falta el reporte del motorizado",message:"Debe marcar «No pude entregar» en su enlace antes de registrar la entrega fallida."};
    case "too_soon":return {tone:"warning",title:"Aún no puedes registrarla",message:"Dale tiempo al cliente para responder antes de registrar la entrega fallida."};
    case "expired":return {tone:"warning",title:"La oferta venció",message:"Pasaron los 3 minutos para responder."};
    case "taken":return {tone:"warning",title:"Ya no está disponible",message:"Otra central tomó este pedido o dejó de ofrecerse."};
    case "cancelled":return {tone:"warning",title:"Pedido cancelado",message:"Este pedido ya no requiere delivery. No lo recojas."};
    case "payment_conflict":return action==="release"
      ?{tone:"danger",title:"Coordina con el local",message:"Caja ya registró el pago de este pedido. Coordina con el local antes de liberarlo."}
      :{tone:"danger",title:"Coordina con el local",message:"El local ya registró pagos de este pedido. No lo recojas ni cobres al cliente."};
    default:return {tone:"danger",title:"No encontramos el pedido",message:"Actualiza la pantalla e inténtalo de nuevo."};
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

/** Problema de la contraseña nueva del operador, con el mismo criterio del backend (8 caracteres). */
export function passwordChangeProblem(current:string,next:string,repeat:string){
  if(!current)return "Ingresa la contraseña temporal.";
  if(Array.from(next).length<8)return "La contraseña debe tener al menos 8 caracteres.";
  if(new TextEncoder().encode(next).length>72)return "La contraseña es demasiado larga.";
  if(next===current)return "La contraseña nueva debe ser distinta de la temporal.";
  if(next!==repeat)return "Las contraseñas no coinciden.";
  return "";
}
