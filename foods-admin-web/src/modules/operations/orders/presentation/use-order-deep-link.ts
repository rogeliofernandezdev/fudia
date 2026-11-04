"use client";
import {useEffect,useRef} from "react";
import {usePathname,useRouter} from "next/navigation";

/**
 * Abre el detalle indicado por `?orderId=` (por ejemplo, al elegir un aviso de la
 * campana) y consume el parámetro de inmediato: la URL vuelve a la pantalla y el
 * mismo aviso puede reabrir el detalle aunque ya se haya cerrado.
 */
export function useOrderDeepLink(orderId:string|undefined,open:(id:string)=>void){
  const router=useRouter();
  const pathname=usePathname();
  const openRef=useRef(open);
  useEffect(()=>{openRef.current=open});
  useEffect(()=>{
    if(!orderId)return;
    openRef.current(orderId);
    router.replace(pathname,{scroll:false});
  },[orderId,pathname,router]);
}
