"use client";
import {useEffect,useState} from "react";

/**
 * Reloj de un segundo para un solo contador (oferta, espera de recogida o entrega
 * fallida). Vive en la tarjeta o fila que lo muestra: así solo ese elemento se
 * vuelve a renderizar cada segundo, no todo el panel. Nunca devuelve un instante
 * anterior a `since` (la hora en que llegaron los datos).
 */
export function useTickingClock(active:boolean,since:number){
  const [now,setNow]=useState(0);
  useEffect(()=>{
    if(!active)return;
    const timer=window.setInterval(()=>setNow(Date.now()),1000);
    return ()=>window.clearInterval(timer);
  },[active]);
  return Math.max(now,since);
}
