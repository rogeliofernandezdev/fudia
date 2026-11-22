"use client";

import {useSyncExternalStore} from "react";
import {TONE_ATTACK_S,TONE_PEAK,type Tone} from "./tones";

/**
 * Un solo AudioContext para toda la página: la campana, los pagos de Fudia Pay
 * y el panel de reparto suenan por aquí. El navegador solo deja sonar después
 * de un gesto en la página (ningún worker lo evita): el contexto se prepara al
 * montar el primer aviso y cualquier toque o tecla lo habilita para todos.
 * Nunca se cierra; vive lo que vive la pestaña.
 */
let context:AudioContext|null=null;
const listeners=new Set<()=>void>();

function sharedContext():AudioContext|null{
  if(context)return context;
  if(typeof window==="undefined"||!window.AudioContext)return null;
  context=new window.AudioContext();
  context.addEventListener("statechange",()=>listeners.forEach(listener=>listener()));
  window.addEventListener("pointerdown",unlockAudio,{passive:true});
  window.addEventListener("keydown",unlockAudio);
  return context;
}

/** Deja el audio listo para que el próximo gesto lo habilite. */
export function prepareAudio(){
  sharedContext();
}

/** Habilita el sonido; solo funciona dentro de un gesto del usuario o si el navegador ya lo permite. */
export function unlockAudio(){
  const audio=sharedContext();
  if(audio&&audio.state!=="running")void audio.resume().catch(()=>undefined);
}

/** Toca las notas ahora; si el navegador aún bloquea el sonido, calla. */
export function playTones(tones:readonly Tone[]){
  const audio=context;
  if(!audio||audio.state!=="running")return;
  for(const tone of tones){
    const start=audio.currentTime+tone.offset;
    const oscillator=audio.createOscillator();
    const gain=audio.createGain();
    oscillator.type=tone.wave;
    oscillator.frequency.value=tone.frequency;
    gain.gain.setValueAtTime(.0001,start);
    gain.gain.exponentialRampToValueAtTime(tone.peak??TONE_PEAK,start+(tone.attack??TONE_ATTACK_S));
    gain.gain.exponentialRampToValueAtTime(.0001,start+tone.duration);
    oscillator.connect(gain).connect(audio.destination);
    oscillator.start(start);
    oscillator.stop(start+tone.duration+.02);
  }
}

function subscribe(listener:()=>void){
  sharedContext();
  listeners.add(listener);
  return()=>{listeners.delete(listener)};
}

/** Sin Web Audio no hay nada que habilitar: no se considera bloqueado. */
function blocked(){
  return context!==null&&context.state!=="running";
}

/** ¿El navegador está bloqueando el sonido hasta un gesto? Prepara el audio al montar. */
export function useAudioBlocked(){
  return useSyncExternalStore(subscribe,blocked,()=>false);
}
