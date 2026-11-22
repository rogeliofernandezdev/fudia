import type {PayProvider} from "./types";

/** Una nota del aviso sonoro: frecuencia (Hz), inicio y duración (s). */
export type PayTone={frequency:number;offset:number;duration:number;wave:"sine"|"triangle"};

/**
 * Cada billetera tiene su propio sonido para que el cajero sepa cuál llegó sin
 * mirar la pantalla. Son tonos propios de Fudia, no los de las apps de Yape o
 * Plin (AGENTS regla 11):
 * - Yape: tres notas cálidas que suben (do–mi–sol).
 * - Plin: dos notas agudas y brillantes, cortas, como un «plin».
 */
export const payAlertTones:Record<PayProvider,PayTone[]>={
  yape:[
    {frequency:523.25,offset:0,duration:.16,wave:"triangle"},
    {frequency:659.25,offset:.15,duration:.16,wave:"triangle"},
    {frequency:783.99,offset:.3,duration:.36,wave:"triangle"},
  ],
  plin:[
    {frequency:1318.51,offset:0,duration:.12,wave:"sine"},
    {frequency:1975.53,offset:.11,duration:.42,wave:"sine"},
  ],
};

/** Pausa entre los sonidos cuando llegan pagos de las dos billeteras juntos. */
export const PAY_ALERT_GAP_S=.9;

/** Billeteras a sonar, en orden de llegada y sin repetir: tres Yape juntos suenan una vez. */
export function payAlertSequence(providers:PayProvider[]):PayProvider[]{
  return Array.from(new Set(providers));
}

/** Texto que alterna en la pestaña mientras el cajero no la está mirando. */
export function payAlertTitle(count:number,first:{provider:string;amount:string}|null){
  if(count>1)return `(${count}) Pagos recibidos`;
  if(!first)return "";
  return `${first.provider} · ${first.amount} recibido`;
}

/** Intervalo del parpadeo del título. */
export const PAY_TITLE_BLINK_MS=1000;
