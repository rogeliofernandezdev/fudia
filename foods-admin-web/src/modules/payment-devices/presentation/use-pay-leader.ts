"use client";

import {useEffect,useState} from "react";
import {holdPayLeadership} from "./pay-leader";

/** ¿Esta pestaña es la que avisa los pagos del cajero en este local? Al
 * enfocarla o tocarla, la reclama (ver holdPayLeadership). */
export function usePayLeader(name:string){
  const [leader,setLeader]=useState(false);
  useEffect(()=>{
    const leadership=holdPayLeadership(navigator.locks,name,setLeader);
    const claim=()=>{if(!document.hidden)leadership.claim()};
    window.addEventListener("focus",claim);
    window.addEventListener("pointerdown",claim);
    if(document.hasFocus())claim();
    return()=>{
      window.removeEventListener("focus",claim);
      window.removeEventListener("pointerdown",claim);
      leadership.dispose();
    };
  },[name]);
  return leader;
}
