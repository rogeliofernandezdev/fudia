"use client";
import {broadcastSessionChange} from "./session-events";

let expirationInProgress=false;

/**
 * Cierre manual en curso: el 401 que llega después es esperado y no debe abrir
 * la salida por «sesión vencida» (segunda llamada y recarga completa).
 */
export function beginSignOut():void{
  expirationInProgress=true;
}

export function expireBrowserSession():void{
  if(typeof window==="undefined"||expirationInProgress)return;
  expirationInProgress=true;
  // Solo borra la cookie en el BFF: la sesión ya venció en el backend.
  void fetch("/api/session?scope=local",{method:"DELETE"})
    .catch(()=>{})
    .finally(()=>{
      broadcastSessionChange("signed-out");
      window.location.replace("/login");
    });
}
