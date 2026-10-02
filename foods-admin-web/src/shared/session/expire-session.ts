"use client";
import {broadcastSessionChange} from "./session-events";

let expirationInProgress=false;

export function expireBrowserSession():void{
  if(typeof window==="undefined"||expirationInProgress)return;
  expirationInProgress=true;
  void fetch("/api/session",{method:"DELETE"})
    .catch(()=>{})
    .finally(()=>{
      broadcastSessionChange("signed-out");
      window.location.replace("/login");
    });
}
