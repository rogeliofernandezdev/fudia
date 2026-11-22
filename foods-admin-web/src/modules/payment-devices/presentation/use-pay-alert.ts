"use client";

import {useCallback,useEffect,useRef} from "react";
import {playTones,unlockAudio,useAudioBlocked} from "@/shared/audio";
import {PAY_TITLE_BLINK_MS,payAlertSchedule,payAlertTitle} from "../domain/pay-alert";
import type {PayProvider} from "../domain/types";

/**
 * Aviso de pago nuevo para el cajero: un sonido propio por billetera (Yape y
 * Plin suenan distinto) y, mientras la pestaña no tiene el foco, su título
 * parpadea como en LinkedIn hasta que el cajero vuelve.
 *
 * Suena por el audio compartido de la página (shared/audio): si el navegador
 * aún lo bloquea, `soundBlocked` avisa para mostrar «Activar sonido de pagos»,
 * y cualquier toque o tecla también lo habilita.
 */
export function usePayAlert(){
  const blink=useRef<{count:number;base:string;stop:(restore?:boolean)=>void}|null>(null);
  const soundBlocked=useAudioBlocked();
  useEffect(()=>()=>blink.current?.stop(),[]);

  const alertPayments=useCallback((items:{provider:PayProvider;providerLabel:string;amount:string}[])=>{
    if(!items.length)return;
    playTones(payAlertSchedule(items.map(item=>item.provider)));
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

  return {alertPayments,soundBlocked,enableSound:unlockAudio};
}
