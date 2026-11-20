import type {Query} from "@tanstack/react-query";

/**
 * Temas que el backend avisa por el canal en vivo (`GET /v1/admin/events`) cuando
 * se confirma una escritura del local. Coinciden con la migración 000138.
 */
export type LiveTopic="orders"|"kitchen"|"payments"|"cash"|"notices"|"pay"|"availability"|"catalog"|"delivery"|"salon"|"reservations";

/** Intervalo de consulta: fijo, apagado o calculado con el estado de la consulta. */
export type LivePoll=number|false|((query:Query)=>number|false|undefined);

/** Temas de la consulta: fijos o según su estado (p. ej. cuántas páginas cargó). */
export type LiveTopics=readonly LiveTopic[]|((query:Query)=>readonly LiveTopic[]);

/**
 * Una consulta en vivo declara en `meta` los temas que la afectan y su intervalo
 * normal: `meta:{live:["kitchen","orders"],poll:10_000}`. Con el canal conectado
 * se refresca al llegar uno de sus temas y su consulta periódica baja a
 * `LIVE_FALLBACK_MS` (respaldo por si se pierde un aviso); sin canal, consulta
 * con `poll` como siempre. No se combina con un `refetchInterval` propio.
 */
export type LiveQueryMeta={live?:LiveTopics;poll?:LivePoll};

declare module "@tanstack/react-query" {
  interface Register {queryMeta:LiveQueryMeta&Record<string,unknown>}
}

/** Respaldo con el canal conectado: el aviso llega en menos de un segundo. */
export const LIVE_FALLBACK_MS=60_000;

let connected=false;

export function isLiveConnected(){return connected}
export function setLiveConnected(value:boolean){connected=value}

function topicsOf(query:Query):readonly LiveTopic[]{
  const live=query.meta?.live;
  return (typeof live==="function"?live(query):live)??[];
}

/** `refetchInterval` por defecto del QueryClient: aplica `meta.poll` según el canal. */
export function liveRefetchInterval(query:Query):number|false{
  const poll=query.meta?.poll;
  const value=typeof poll==="function"?poll(query):poll;
  if(!value)return false;
  return connected&&topicsOf(query).length?Math.max(value,LIVE_FALLBACK_MS):value;
}

/** ¿La consulta es en vivo y depende de alguno de los temas avisados? Sin temas: cualquiera en vivo. */
export function matchesLiveTopics(query:Query,topics?:ReadonlySet<string>){
  const own=topicsOf(query);
  return topics?own.some(topic=>topics.has(topic)):own.length>0;
}
