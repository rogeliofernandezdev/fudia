"use client";

import {useEffect,useRef} from "react";
import {newOffers} from "../domain/panel";
import type {PanelOrder} from "../domain/types";

const TITLE="Panel de reparto";

/**
 * Avisa cada pedido nuevo: sonido, vibración, título de la pestaña y, si el
 * navegador lo permite, una notificación cuando el panel no está a la vista.
 * El sonido necesita que el usuario lo haya activado con un toque (`enabled`).
 */
export function useOfferAlert(orders:PanelOrder[]|undefined,enabled:boolean){
  const seen=useRef<Set<string>>(new Set());
  const audio=useRef<AudioContext|null>(null);
  const offers=orders?.filter(order=>order.stage==="offer").length??0;

  // El navegador solo deja sonar después de un gesto: el contexto se crea al
  // activar los avisos y se reanuda con el siguiente toque tras recargar.
  useEffect(()=>{
    if(!enabled)return;
    const resume=()=>{
      const Context=window.AudioContext;
      if(!Context)return;
      audio.current??=new Context();
      void audio.current.resume().catch(()=>undefined);
    };
    resume();
    window.addEventListener("pointerdown",resume);
    return ()=>window.removeEventListener("pointerdown",resume);
  },[enabled]);

  useEffect(()=>{
    document.title=offers?`(${offers}) Pedido nuevo · ${TITLE}`:TITLE;
    return ()=>{document.title=TITLE};
  },[offers]);

  useEffect(()=>{
    if(!orders)return;
    const fresh=newOffers(seen.current,orders);
    seen.current=new Set(orders.filter(order=>order.stage==="offer").map(order=>order.token));
    if(!fresh.length||!enabled)return;
    beep(audio.current);
    navigator.vibrate?.([220,120,220]);
    if(document.hidden&&"Notification" in window&&Notification.permission==="granted"){
      // En Android Chrome `new Notification` exige un service worker y lanza: el sonido ya avisó.
      try{new Notification("Pedido nuevo de delivery",{body:`${fresh[0].organizationName} · ${fresh[0].address}`,tag:"fudia-delivery-offer"})}catch{/* sin notificación del sistema */}
    }
  },[orders,enabled]);
}

/** Dos tonos cortos: se distinguen del resto de sonidos del teléfono sin ser estridentes. */
function beep(context:AudioContext|null){
  if(!context||context.state!=="running")return;
  [0,.28].forEach((offset,index)=>{
    const oscillator=context.createOscillator();
    const gain=context.createGain();
    const start=context.currentTime+offset;
    oscillator.type="sine";
    oscillator.frequency.value=index?1175:880;
    gain.gain.setValueAtTime(.0001,start);
    gain.gain.exponentialRampToValueAtTime(.35,start+.02);
    gain.gain.exponentialRampToValueAtTime(.0001,start+.22);
    oscillator.connect(gain).connect(context.destination);
    oscillator.start(start);
    oscillator.stop(start+.24);
  });
}

/** Pide el permiso de notificaciones al activar los avisos; negarlo no impide el sonido. */
export function requestAlertPermission(){
  if("Notification" in window&&Notification.permission==="default")void Notification.requestPermission().catch(()=>undefined);
}
