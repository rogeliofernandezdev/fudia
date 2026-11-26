import type {KitchenItem,KitchenTicket} from "./types";

export type KitchenUrgency="fresh"|"warning"|"late"|"ready"|"neutral";

/** El API responde con zona horaria corta (`-05`); se normaliza a `-05:00`. */
export function parseIsoDate(value:string|undefined){
  if(!value)return null;
  const normalized=value.replace(/([+-]\d{2})$/,"$1:00");
  const parsed=new Date(normalized);
  if(!Number.isNaN(parsed.getTime()))return parsed;
  const fallback=new Date(value);
  return Number.isNaN(fallback.getTime())?null:fallback;
}

/** Último cuarto del tiempo de preparación: el plato está por vencer. */
const warningShare=.25;

export type PlateTimer={tone:Extract<KitchenUrgency,"fresh"|"warning"|"late">;label:string};

/**
 * Tiempo del plato en preparación: cuenta desde que su estación lo empezó y
 * descuenta el tiempo de preparación del producto. Sin tiempo definido, sin
 * hora de inicio o fuera de preparación no hay contador.
 */
export function plateTimer(item:KitchenItem,now:number):PlateTimer|null{
  if(item.status!=="preparando"||!item.prepMinutes)return null;
  const start=parseIsoDate(item.preparingSince);
  if(!start)return null;
  const target=item.prepMinutes*60000;
  const remaining=target-Math.max(0,now-start.getTime());
  if(remaining>0){
    const minutes=Math.ceil(remaining/60000);
    return {tone:remaining<=target*warningShare?"warning":"fresh",label:`${minutes===1?"Falta":"Faltan"} ${minutes} de ${item.prepMinutes} min`};
  }
  const over=Math.floor(-remaining/60000);
  return {tone:"late",label:over<1?"Tiempo cumplido":`Demora ${over} min`};
}

/**
 * Tiempo propio del plato antes de empezar («15 min»): cada plato muestra el suyo,
 * la comanda no impone uno común. Null si el producto no lo define o ya hay contador.
 */
export function plateEstimate(item:KitchenItem){
  if(!item.prepMinutes||item.status==="listo")return null;
  if(item.status==="preparando"&&parseIsoDate(item.preparingSince))return null;
  return `${item.prepMinutes} min`;
}

/** Parte consumida del tiempo de cada plato en preparación (1 = vencido). */
function plateRatios(ticket:KitchenTicket,now:number){
  return ticket.items.flatMap(item=>{
    const start=parseIsoDate(item.preparingSince);
    if(item.status!=="preparando"||!item.prepMinutes||!start)return [];
    return [Math.max(0,now-start.getTime())/(item.prepMinutes*60000)];
  });
}

/** Comandas con estado por plato; las históricas solo traen el objetivo de la comanda. */
function hasTrackedPlates(ticket:KitchenTicket){
  return ticket.items.some(item=>item.status!==undefined);
}

/**
 * Inicio del tiempo de la comanda: en preparación, el plato que empezó primero
 * (marcar platos listos no reinicia el reloj); en otro caso, su última actualización.
 */
export function ticketStart(ticket:KitchenTicket){
  if(ticket.status==="preparando"){
    const starts=ticket.items.map(item=>parseIsoDate(item.preparingSince)?.getTime()).filter((value):value is number=>value!==undefined);
    if(starts.length)return Math.min(...starts);
  }
  return parseIsoDate(ticket.updatedAt)?.getTime()??null;
}

export function elapsedMinutes(ticket:KitchenTicket,now:number){
  const start=ticketStart(ticket);
  if(start===null)return 0;
  return Math.max(0,Math.floor((now-start)/60000));
}

const toneRank:Record<PlateTimer["tone"],number>={fresh:0,warning:1,late:2};

/**
 * Urgencia de la comanda, siempre desde el tiempo de cada plato: manda el plato
 * más atrasado, así la cabecera nunca dice «A tiempo» con un plato con demora.
 * Sin platos en marcha no hay urgencia (la espera se ve como tiempo); solo las
 * comandas históricas, sin estado por plato, comparan con el objetivo de la comanda.
 */
export function ticketUrgency(ticket:KitchenTicket,now:number):KitchenUrgency{
  if(ticket.status==="listo")return "ready";
  const timers=ticket.items.map(item=>plateTimer(item,now)).filter((timer):timer is PlateTimer=>timer!==null);
  if(timers.length)return timers.reduce((worst,timer)=>toneRank[timer.tone]>toneRank[worst]?timer.tone:worst,"fresh" as PlateTimer["tone"]);
  if(hasTrackedPlates(ticket)||!ticket.targetMinutes)return "neutral";
  const ratio=elapsedMinutes(ticket,now)/ticket.targetMinutes;
  if(ratio>=1)return "late";
  if(ratio>=1-warningShare)return "warning";
  return "fresh";
}

/**
 * Orden dentro del carril (mayor primero): primero la comanda cuyo plato consumió
 * más de su propio tiempo; sin platos en marcha, la que lleva más esperando.
 */
export function ticketPriority(ticket:KitchenTicket,now:number){
  const elapsed=elapsedMinutes(ticket,now);
  if(ticket.status==="listo")return elapsed;
  const ratios=plateRatios(ticket,now);
  if(ratios.length)return Math.max(...ratios)*10000+elapsed;
  if(hasTrackedPlates(ticket)||!ticket.targetMinutes)return elapsed;
  return elapsed/ticket.targetMinutes*10000+elapsed;
}
