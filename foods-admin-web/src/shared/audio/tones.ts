/**
 * Una nota de un aviso sonoro: frecuencia (Hz), inicio y duración (s) desde
 * que empieza el sonido. `peak` es el volumen máximo y `attack` lo que tarda
 * en alcanzarlo; sin ellos se usa el volumen común de los avisos de Fudia.
 * Solo es un tipo: los dominios lo importan con `import type` y siguen puros.
 */
export type Tone={frequency:number;offset:number;duration:number;wave:"sine"|"triangle";peak?:number;attack?:number};

export const TONE_PEAK=.3;
export const TONE_ATTACK_S=.015;
