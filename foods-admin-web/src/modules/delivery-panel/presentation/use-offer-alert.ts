"use client";

import {useEffect,useRef} from "react";
import {playTones,unlockAudio,type Tone} from "@/shared/audio";
import {newOffers,urgentKeys} from "../domain/panel";
import type {PanelOrder} from "../domain/types";

const TITLE="Panel de reparto";

/**
 * Avisa cada pedido nuevo, cada alerta crítica de recogida sin decidir y cada
 * «No pude entregar» del motorizado: sonido, vibración, título de la pestaña y,
 * si el navegador lo permite, una notificación cuando el panel no está a la vista.
 * El sonido necesita que el usuario lo haya activado con un toque (`enabled`).
 */
export function useOfferAlert(orders:PanelOrder[]|undefined,enabled:boolean){
  const seen=useRef<Set<string>>(new Set());
  const urgentSeen=useRef<Set<string>>(new Set());
  const offers=orders?.filter(order=>order.stage==="offer").length??0;
  const urgent=orders?urgentKeys(orders).length:0;

  // El navegador solo deja sonar después de un gesto: el audio compartido se
  // habilita al activar los avisos y se reanuda con el siguiente toque tras recargar.
  useEffect(()=>{if(enabled)unlockAudio()},[enabled]);

  useEffect(()=>{
    document.title=urgent?`(${urgent}) Atención · ${TITLE}`:offers?`(${offers}) Pedido nuevo · ${TITLE}`:TITLE;
    return ()=>{document.title=TITLE};
  },[offers,urgent]);

  useEffect(()=>{
    if(!orders)return;
    const fresh=newOffers(seen.current,orders);
    seen.current=new Set(orders.filter(order=>order.stage==="offer").map(order=>order.token));
    const keys=urgentKeys(orders);
    const freshUrgent=keys.filter(key=>!urgentSeen.current.has(key));
    urgentSeen.current=new Set(keys);
    if((!fresh.length&&!freshUrgent.length)||!enabled)return;
    playTones(OFFER_TONES);
    navigator.vibrate?.([220,120,220]);
    if(document.hidden&&"Notification" in window&&Notification.permission==="granted"){
      // En Android Chrome `new Notification` exige un service worker y lanza: el sonido ya avisó.
      const body=fresh.length?`${fresh[0].organizationName} · ${fresh[0].address}`:"Un reparto necesita tu decisión.";
      try{new Notification(fresh.length?"Pedido nuevo de delivery":"Atención en un reparto",{body,tag:fresh.length?"fudia-delivery-offer":"fudia-delivery-alert"})}catch{/* sin notificación del sistema */}
    }
  },[orders,enabled]);
}

/** Dos tonos cortos: se distinguen del resto de sonidos del teléfono sin ser estridentes. */
const OFFER_TONES:Tone[]=[
  {frequency:880,offset:0,duration:.22,wave:"sine",peak:.35,attack:.02},
  {frequency:1175,offset:.28,duration:.22,wave:"sine",peak:.35,attack:.02},
];

/** Pide el permiso de notificaciones al activar los avisos; negarlo no impide el sonido. */
export function requestAlertPermission(){
  if("Notification" in window&&Notification.permission==="default")void Notification.requestPermission().catch(()=>undefined);
}
