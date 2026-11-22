"use client";

import {useEffect,useState} from "react";
import {nextPayWakeDelay} from "../domain/pay-wake";
import {wakePayDevices} from "../infrastructure/pay-devices-api";
import {createBackgroundTimer} from "./background-timer";

/** Mientras Caja está abierta despierta los teléfonos de Fudia Pay del local
 * (cada 30 s; cada 10 s con el modal de cobro abierto, y al abrirlo). La espera
 * corre en un Web Worker para que siga a tiempo con la pestaña minimizada. Es
 * de mejor esfuerzo: un error no se muestra y el siguiente intento lo repite.
 * Solo la pestaña líder lo hace (ver usePayLeader); las demás no generan tráfico.
 * Devuelve cuántos teléfonos del local reciben push (null hasta saberlo). */
export function usePayDeviceWake(charging:boolean,active=true){
  const [devices,setDevices]=useState<number|null>(null);
  useEffect(()=>{
    if(!active)return;
    let cancelled=false;
    const timer=createBackgroundTimer();
    const tick=async()=>{
      let found:number|null=null;
      try{found=(await wakePayDevices()).devices}catch{/* sin red o sin permiso: se reintenta */}
      if(cancelled)return;
      if(found!==null)setDevices(found);
      timer.after(nextPayWakeDelay(found,charging),()=>void tick());
    };
    void tick();
    return()=>{cancelled=true;timer.dispose()};
  },[charging,active]);
  return active?devices:null;
}
