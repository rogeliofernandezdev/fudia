"use client";
import {broadcastSessionChange} from "./session-events";

let expirationInProgress=false;

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
