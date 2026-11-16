"use client";

import {createContext,useContext,useEffect,useMemo,useState} from "react";

/** El pedido que el cajero está cobrando ahora mismo en el POS: la sugerencia
 * del Escenario A. El aviso flotante la propone, nunca la aplica solo. */
export type PayOpenCollection={id:string;code:string;pending:string};
type PayCollectionValue={open:PayOpenCollection|null;register:(order:PayOpenCollection)=>void;clear:()=>void};
const PayCollectionContext=createContext<PayCollectionValue>({open:null,register:()=>{},clear:()=>{}});

export function PayCollectionProvider({children}:{children:React.ReactNode}){
  const [open,setOpen]=useState<PayOpenCollection|null>(null);
  const value=useMemo(()=>({open,register:setOpen,clear:()=>setOpen(null)}),[open]);
  return <PayCollectionContext.Provider value={value}>{children}</PayCollectionContext.Provider>;
}

export function usePayCollection(){return useContext(PayCollectionContext);}

/** Registra el modal de cobro abierto del POS mientras esté montado; al
 * cerrarse, el contexto se limpia solo. */
export function PayCollectionRegistration({order}:{order:PayOpenCollection|null}){
  const {register,clear}=usePayCollection();
  useEffect(()=>{
    if(!order)return;
    register(order);
    return ()=>clear();
  },[order?.id,order?.pending,register,clear]);
  return null;
}
