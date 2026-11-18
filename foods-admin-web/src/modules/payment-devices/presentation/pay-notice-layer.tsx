"use client";

import {useEffect,useRef,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {useSession} from "@/providers";
import {PayProviderTag} from "./pay-status-tag";
import type {PayNotification} from "../domain/types";
import {listPayNotifications} from "../infrastructure/pay-devices-api";
import {PayNotificationDetail} from "./pay-notification-detail";

/** Capa global de pagos detectados (§1): en cualquier pantalla del cajero,
 * cada notificación nueva de Yape o Plin aparece como tarjeta flotante
 * apilable, sin cerrar ni alterar los modales abiertos. El clic abre el
 * detalle para asociar; si se cierra, el movimiento queda pendiente en el
 * historial de Caja.
 *
 * Sondea SOLO con sesión de cajero vigente: sin sesión el BFF responde 401 y
 * cada sondeo forzaría el cierre de sesión — por eso la capa se silencia sin
 * usuario, sin permiso o ante el primer 401. */
export function PayNoticeLayer(){
  const session=useSession();
  const [visible,setVisible]=useState<PayNotification[]>([]);
  const [detail,setDetail]=useState<PayNotification|null>(null);
  const seen=useRef<Set<string>>(new Set());

  // Sondea SOLO con sesión de cajero vigente: sin sesión el BFF responde 401 y
  // cada sondeo forzaría el cierre de sesión. Un 401 detiene el intervalo; el
  // próximo ingreso reinicia el cliente de consultas y la capa vuelve sola.
  // Un fallo pasajero del API (503, red) no mata la capa: el sondeo sigue
  // cada 10 s y conserva el último dato bueno, para que un tropiezo no
  // silencie los avisos hasta recargar la página.
  const sessionOk=Boolean(session.user)&&session.can("cash.read")&&!session.isUnauthorized&&!session.isLoading;
  const feed=useQuery({
    queryKey:["pay-notice-feed"],
    queryFn:()=>listPayNotifications({status:"detected",provider:""}),
    enabled:sessionOk,
    refetchInterval:query=>sessionOk?10000:false,
    refetchOnWindowFocus:false,
    retry:1,
    gcTime:0,
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

  if(!sessionOk)return null;
  return <>
    {visible.length>0&&<div className="pay-notice-layer" role="region" aria-label="Pagos detectados por Yape y Plin">
      {visible.map(item=><button key={item.id} type="button" className="pay-notice" onClick={()=>setDetail(item)}>
        <PayProviderTag provider={item.provider}/>
        <span className="pay-notice-main"><strong>{item.currency} {item.amount}</strong><small>{item.senderName||"Sin remitente"} · {item.deviceName}</small></span>
        <span className="pay-notice-hint">Ver pago detectado</span>
      </button>)}
    </div>}
    <PayNotificationDetail notification={detail} onClose={()=>setDetail(null)} onDone={()=>{setDetail(null);void feed.refetch()}}/>
  </>;
}
