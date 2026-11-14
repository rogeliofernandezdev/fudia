"use client";

import {useEffect,useRef,useState} from "react";
import type {CourierPosition} from "../domain/types";
import {sendCourierLocation} from "../infrastructure/panel-api";

export type LocationStatus="off"|"locating"|"active"|"denied"|"unsupported"|"error";

/** Cada cuánto se envía como máximo una ubicación nueva. */
const SEND_EVERY_MS=8_000;
/** Detenido, el repartidor reenvía su punto para que el seguimiento no lo dé por perdido. */
const KEEP_ALIVE_MS=30_000;

/**
 * Comparte la ubicación del dispositivo mientras `enabled` es true. El panel la
 * activa solo con un reparto en curso: sin pedidos no se pide ni se envía nada.
 * Mantiene la pantalla encendida mientras comparte, porque el navegador deja de
 * informar la ubicación con la pantalla apagada.
 */
export function useCourierLocation(token:string,enabled:boolean){
  // Solo los avisos del GPS cambian el estado; apagado, sin GPS o esperando se derivan.
  const [reading,setReading]=useState<{status:"active"|"denied"|"error";position:CourierPosition|null}|null>(null);
  const latest=useRef<CourierPosition|null>(null);
  const sentAt=useRef(0);
  const supported=typeof navigator==="undefined"||"geolocation" in navigator;

  useEffect(()=>{
    if(!enabled||!("geolocation" in navigator))return;
    let active=true;
    const send=(force=false)=>{
      const point=latest.current;
      if(!point||(!force&&Date.now()-sentAt.current<SEND_EVERY_MS))return;
      sentAt.current=Date.now();
      // Un envío fallido no interrumpe el reparto: el siguiente punto lo corrige.
      void sendCourierLocation(token,point).catch(()=>undefined);
    };
    const watch=navigator.geolocation.watchPosition(result=>{
      if(!active)return;
      const point={latitude:result.coords.latitude,longitude:result.coords.longitude};
      latest.current=point;
      setReading({status:"active",position:point});
      send();
    },error=>{
      if(!active)return;
      setReading(current=>({status:error.code===error.PERMISSION_DENIED?"denied":"error",position:current?.position??null}));
    },{enableHighAccuracy:true,maximumAge:5_000,timeout:20_000});
    const keepAlive=window.setInterval(()=>send(true),KEEP_ALIVE_MS);

    let lock:WakeLockSentinel|null=null;
    const hold=()=>{
      if(document.visibilityState!=="visible"||!("wakeLock" in navigator))return;
      void navigator.wakeLock.request("screen").then(sentinel=>{if(active)lock=sentinel;else void sentinel.release()}).catch(()=>undefined);
    };
    hold();
    document.addEventListener("visibilitychange",hold);
    return ()=>{
      active=false;
      navigator.geolocation.clearWatch(watch);
      window.clearInterval(keepAlive);
      document.removeEventListener("visibilitychange",hold);
      void lock?.release().catch(()=>undefined);
      latest.current=null;
      setReading(null);
    };
  },[token,enabled]);

  const status:LocationStatus=!enabled?"off":!supported?"unsupported":reading?.status??"locating";
  return {status,position:enabled?reading?.position??null:null};
}
