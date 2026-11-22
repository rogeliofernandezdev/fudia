"use client";

import {useState} from "react";

/**
 * Preferencia del dispositivo (avisos, ubicación). Se lee al crear el estado: el
 * servidor solo pinta la carga, así que el primer render del cliente no depende de ella.
 */
export function useStoredFlag(name:string):[boolean,(value:boolean)=>void]{
  const [value,setValue]=useState(()=>{
    if(typeof window==="undefined")return false;
    try{return window.localStorage.getItem(name)==="1"}catch{return false}
  });
  return [value,next=>{setValue(next);try{window.localStorage.setItem(name,next?"1":"0")}catch{/* almacenamiento bloqueado */}}];
}
