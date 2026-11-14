import type {ChatTracking} from "./types";

export type TrackingStep={label:string;state:"done"|"current"|"pending"};

const DELIVERY_STEPS=["Recibido","Preparando","En camino","Entregado"];
const PICKUP_STEPS=["Recibido","Preparando","Listo","Entregado"];

const isDelivery=(tracking:ChatTracking)=>tracking.channel==="delivery";

/** Posición del pedido en sus cuatro pasos; «listo» de un delivery sigue esperando al repartidor. */
function stepIndex(tracking:ChatTracking){
  switch(tracking.status){
    case "nuevo":return 0;
    case "confirmado":case "preparando":return 1;
    case "listo":return isDelivery(tracking)?1:2;
    case "en_camino":return 2;
    case "entregado":return 4;
    default:return 0;
  }
}

/** Pasos del pedido para el comensal; un pedido cancelado no muestra avance. */
export function trackingSteps(tracking:ChatTracking):TrackingStep[]{
  if(tracking.status==="cancelado")return [];
  const index=stepIndex(tracking);
  return (isDelivery(tracking)?DELIVERY_STEPS:PICKUP_STEPS).map((label,position)=>({label,state:position<index?"done":position===index?"current":"pending"}));
}

/** Una frase que dice qué pasa ahora con el pedido. */
export function trackingHeadline(tracking:ChatTracking){
  const assigned=tracking.delivery?.dispatchStatus==="accepted";
  switch(tracking.status){
    case "nuevo":return "El local está revisando tu pedido.";
    case "confirmado":case "preparando":return assigned?"Tu pedido se está preparando y ya tiene repartidor.":"Tu pedido se está preparando.";
    case "listo":return !isDelivery(tracking)?"Tu pedido está listo para recoger en el local.":assigned?"Tu pedido está listo. El repartidor va a recogerlo.":"Tu pedido está listo y saldrá pronto.";
    case "en_camino":return "Tu pedido va en camino.";
    case "entregado":return "Tu pedido fue entregado. ¡Buen provecho!";
    case "cancelado":return "El local canceló este pedido.";
    default:return "Estamos consultando tu pedido.";
  }
}

/** El mapa acompaña al reparto en curso; entregado o cancelado ya no lo necesita. */
export function showsMap(tracking:ChatTracking){
  const delivery=tracking.delivery;
  return Boolean(delivery&&(delivery.origin||delivery.destination)&&tracking.status!=="entregado"&&tracking.status!=="cancelado");
}

function elapsed(since:string,now:number){
  const seconds=Math.max(0,Math.round((now-new Date(since).getTime())/1000));
  if(seconds<45)return "hace unos segundos";
  const minutes=Math.round(seconds/60);
  return minutes<=1?"hace 1 minuto":`hace ${minutes} minutos`;
}

/** Qué se sabe del repartidor; vacío si el pedido no tiene reparto que mostrar. */
export function courierNote(tracking:ChatTracking,now:number){
  const delivery=tracking.delivery;
  if(!delivery||!showsMap(tracking))return "";
  if(delivery.courier)return `Ubicación del repartidor actualizada ${delivery.courierLocatedAt?elapsed(delivery.courierLocatedAt,now):"hace poco"}.`;
  if(delivery.dispatchStatus==="searching")return "Estamos buscando un repartidor para tu pedido.";
  if(delivery.dispatchStatus==="accepted"||delivery.dispatchStatus==="picked_up")return "El repartidor aún no comparte su ubicación.";
  return "";
}
