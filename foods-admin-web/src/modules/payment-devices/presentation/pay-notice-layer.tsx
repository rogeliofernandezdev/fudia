"use client";

import {useEffect,useRef,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {Tag} from "@/design-system";
import {payProviderLabel} from "../domain/pay-status";
import type {PayNotification} from "../domain/types";
import {listPayNotifications} from "../infrastructure/pay-devices-api";
import {PayNotificationDetail} from "./pay-notification-detail";

/** Capa global de pagos detectados (§1): en cualquier pantalla del cajero,
 * cada notificación nueva de Yape o Plin aparece como tarjeta flotante
 * apilable, sin cerrar ni alterar los modales abiertos. El clic abre el
 * detalle para asociar; si se cierra, el movimiento queda pendiente en el
 * historial de Caja. Sin permiso de cobro la capa no se muestra. */
export function PayNoticeLayer(){
  const [visible,setVisible]=useState<PayNotification[]>([]);
  const [detail,setDetail]=useState<PayNotification|null>(null);
  const seen=useRef<Set<string>>(new Set());

  const feed=useQuery({
    queryKey:["pay-notice-feed"],
    queryFn:()=>listPayNotifications({status:"detected",provider:""}),
    refetchInterval:10000,
    retry:false,
  });

  useEffect(()=>{
    const items=feed.data?.items??[];
    const fresh=items.filter(item=>!seen.current.has(item.id));
    if(fresh.length){
      fresh.forEach(item=>seen.current.add(item.id));
      setVisible(current=>[...fresh,...current.filter(stale=>items.some(item=>item.id===stale.id))].slice(0,4));
      return;
    }
    setVisible(current=>current.filter(stale=>items.some(item=>item.id===stale.id)));
  },[feed.data]);

  if(feed.isError||feed.isLoading&&!feed.data)return null;
  return <>
    {visible.length>0&&<div className="pay-notice-layer" role="region" aria-label="Pagos detectados por Yape y Plin">
      {visible.map(item=><button key={item.id} type="button" className="pay-notice" onClick={()=>setDetail(item)}>
        <Tag>{payProviderLabel[item.provider]}</Tag>
        <span className="pay-notice-main"><strong>{item.currency} {item.amount}</strong><small>{item.senderName||"Sin remitente"} · {item.deviceName}</small></span>
        <span className="pay-notice-hint">Ver pago detectado</span>
      </button>)}
    </div>}
    <PayNotificationDetail notification={detail} onClose={()=>setDetail(null)} onDone={()=>{setDetail(null);void feed.refetch()}}/>
  </>;
}
