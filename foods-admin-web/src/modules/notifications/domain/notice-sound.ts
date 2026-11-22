import type {NoticeKind,StaffNotice} from "./types";

/** Una nota del aviso sonoro: frecuencia (Hz), inicio y duración (s). */
export type NoticeTone={frequency:number;offset:number;duration:number;wave:"sine"|"triangle"};

/**
 * Sonido de la campana: uno solo para todo lo que pide atención («general») y
 * el mismo dos veces para lo que se agrava con el tiempo («alert»). Los pagos
 * tienen su propio sonido por billetera (pay-alert); el pedido de mostrador lo
 * creó el propio cajero y la cuenta abierta hace mucho es informativa: callan.
 */
export type NoticeSound="general"|"alert";

const soundByKind:Record<NoticeKind,NoticeSound|null>={
  table_order:"general",
  delivery_order:"general",
  pickup_order:"general",
  order_ready:"general",
  bill_requested:"general",
  assistance_requested:"general",
  delivery_unassigned:"alert",
  delivery_pickup_delayed:"alert",
  counter_order:null,
  stale_account:null,
  pay_detected:null,
};

export function noticeSound(kind:NoticeKind):NoticeSound|null{
  return soundByKind[kind]??null;
}

/** Avisos que llegan juntos suenan una sola vez; si alguno es alerta, suena la alerta. */
export function noticeSoundFor(notices:readonly Pick<StaffNotice,"kind">[]):NoticeSound|null{
  const sounds=notices.map(notice=>noticeSound(notice.kind));
  return sounds.includes("alert")?"alert":sounds.includes("general")?"general":null;
}

/**
 * «Ding-dong» propio de Fudia: dos notas suaves que bajan (sol–mi). No repite
 * la escalera que sube de Yape ni el brillo agudo de Plin.
 */
export const noticeTones:NoticeTone[]=[
  {frequency:783.99,offset:0,duration:.5,wave:"sine"},
  {frequency:659.25,offset:.28,duration:.7,wave:"sine"},
];

/** Pausa entre las dos repeticiones de la alerta. */
export const NOTICE_ALERT_REPEAT_S=1.1;

/** Notas a tocar: el ding-dong una vez, o dos para la alerta. */
export function noticeSoundSchedule(sound:NoticeSound):NoticeTone[]{
  if(sound==="general")return noticeTones;
  return [...noticeTones,...noticeTones.map(tone=>({...tone,offset:tone.offset+NOTICE_ALERT_REPEAT_S}))];
}
