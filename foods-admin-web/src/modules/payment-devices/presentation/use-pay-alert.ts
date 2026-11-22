"use client";

import {useCallback,useEffect,useRef,useState} from "react";
import {PAY_ALERT_GAP_S,PAY_TITLE_BLINK_MS,payAlertSequence,payAlertTitle,payAlertTones} from "../domain/pay-alert";
import type {PayProvider} from "../domain/types";

/**
 * Aviso de pago nuevo para el cajero: un sonido propio por billetera (Yape y
 * Plin suenan distinto) y, mientras la pestaña no tiene el foco, su título
 * parpadea como en LinkedIn hasta que el cajero vuelve.
 *
 * El navegador solo deja sonar después de un gesto en la página (ningún worker
 * lo evita). El audio se prepara al cargar: si el navegador ya lo permite
 * (política del equipo o uso frecuente del sitio) queda activo; si no,
 * `soundBlocked` avisa para mostrar «Activar sonido de pagos», y cualquier
 * toque o tecla también lo habilita.
 */
export function usePayAlert(){
  const audio=useRef<AudioContext|null>(null);
  const blink=useRef<{count:number;base:string;stop:(restore?:boolean)=>void}|null>(null);
  const [soundBlocked,setSoundBlocked]=useState(false);

  const enableSound=useCallback(()=>{
    const context=audio.current;
    if(context&&context.state!=="running")void context.resume().catch(()=>undefined);
  },[]);

  useEffect(()=>{
    const Context=window.AudioContext;
    if(!Context)return;
    const context=new Context();
    audio.current=context;
    const sync=()=>setSoundBlocked(context.state!=="running");
    context.addEventListener("statechange",sync);
    sync();
    window.addEventListener("pointerdown",enableSound);
    window.addEventListener("keydown",enableSound);
    return()=>{
      context.removeEventListener("statechange",sync);
      window.removeEventListener("pointerdown",enableSound);
      window.removeEventListener("keydown",enableSound);
      blink.current?.stop();
      audio.current=null;
      void context.close().catch(()=>undefined);
    };
  },[enableSound]);

  const alertPayments=useCallback((items:{provider:PayProvider;providerLabel:string;amount:string}[])=>{
    if(!items.length)return;
    play(audio.current,payAlertSequence(items.map(item=>item.provider)));
    if(document.hasFocus())return;
    // Pagos que siguen llegando sin que el cajero vuelva suman al mismo parpadeo.
    const previous=blink.current;
    const count=(previous?.count??0)+items.length;
    const base=previous?.base??document.title;
    previous?.stop(false);
    const alert=payAlertTitle(count,{provider:items[0].providerLabel,amount:items[0].amount});
    let showing=true;
    document.title=alert;
    const timer=window.setInterval(()=>{showing=!showing;document.title=showing?alert:base},PAY_TITLE_BLINK_MS);
    const stop=(restore=true)=>{
      window.clearInterval(timer);
      window.removeEventListener("focus",onFocus);
      if(restore)document.title=base;
      blink.current=null;
    };
    const onFocus=()=>stop();
    window.addEventListener("focus",onFocus);
    blink.current={count,base,stop};
  },[]);

  return {alertPayments,soundBlocked,enableSound};
}

function play(context:AudioContext|null,providers:PayProvider[]){
  if(!context||context.state!=="running")return;
  providers.forEach((provider,index)=>{
    for(const tone of payAlertTones[provider]){
      const start=context.currentTime+index*PAY_ALERT_GAP_S+tone.offset;
      const oscillator=context.createOscillator();
      const gain=context.createGain();
      oscillator.type=tone.wave;
      oscillator.frequency.value=tone.frequency;
      gain.gain.setValueAtTime(.0001,start);
      gain.gain.exponentialRampToValueAtTime(.3,start+.015);
      gain.gain.exponentialRampToValueAtTime(.0001,start+tone.duration);
      oscillator.connect(gain).connect(context.destination);
      oscillator.start(start);
      oscillator.stop(start+tone.duration+.02);
    }
  });
}
