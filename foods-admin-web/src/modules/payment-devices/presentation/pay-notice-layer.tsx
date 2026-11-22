"use client";

import {useCallback,useEffect,useRef,useState} from "react";
import {useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,IconButton} from "@/design-system";
import {useMoney,useSession} from "@/providers";
import {PayProviderTag} from "./pay-status-tag";
import {payProviderLabel} from "../domain/pay-status";
import type {PayNotification} from "../domain/types";
import {listPayNotifications} from "../infrastructure/pay-devices-api";
import {PayNotificationDetail} from "./pay-notification-detail";
import {usePayCollection} from "./pay-collection-context";
import {usePayDeviceWake} from "./use-pay-device-wake";
import {usePayAlert} from "./use-pay-alert";
import {usePayLeader} from "./use-pay-leader";

const NOTICE_DURATION_MS=8000;

/** Avisos de pagos para el cajero; cambiar de usuario, empresa o local reinicia
 * el feed y sus avisos. Cerrar la tarjeta nunca modifica el pago detectado. */
export function PayNoticeLayer(){
  const session=useSession();
  const sessionOk=Boolean(session.user&&session.organization&&session.location)
    &&session.can("cash.read")&&session.can("cash.manage")&&!session.isUnauthorized&&!session.isLoading;
  if(!sessionOk)return null;
  const scopeKey=[session.organization!.id,session.location!.id,session.user!.id];
  return <PayNoticeFeed key={scopeKey.join(":")} scopeKey={scopeKey}/>;
}

function PayNoticeFeed({scopeKey}:{scopeKey:string[]}){
  const qc=useQueryClient();
  const [visible,setVisible]=useState<PayNotification[]>([]);
  const [detail,setDetail]=useState<PayNotification|null>(null);
  const seen=useRef<Set<string>|null>(null);
  // Con varias pestañas abiertas, solo una suena, parpadea y pide el push.
  const leader=usePayLeader(`fudia-pay-alert:${scopeKey.join(":")}`);
  // Los teléfonos congelados por el fabricante procesan sus pagos al recibir el push.
  const pushDevices=usePayDeviceWake(Boolean(usePayCollection().open),leader);
  // Sonido propio de Yape o Plin y título de la pestaña que parpadea.
  const {alertPayments,soundBlocked,enableSound}=usePayAlert();
  // Solo si el local usa Fudia Pay: el navegador bloquea el sonido hasta un gesto.
  const askSound=leader&&soundBlocked&&(pushDevices??0)>0;
  const money=useMoney();
  // El efecto de pagos nuevos solo depende del feed: el formato y el aviso se
  // leen de una referencia para no repetirlo (ni sonar) en cada render.
  const announce=useRef({alertPayments,money,leader});
  useEffect(()=>{announce.current={alertPayments,money,leader}},[alertPayments,money,leader]);
  const feed=useQuery({
    queryKey:["pay-notice-feed",...scopeKey],
    queryFn:()=>listPayNotifications({status:"detected",provider:""}),
    meta:{live:["pay"],poll:10000},
    refetchOnWindowFocus:false,
    retry:1,
    gcTime:0,
  });

  useEffect(()=>{
    if(!feed.data)return;
    const items=feed.data.items;
    // El historial inicial queda en la campana; solo los nuevos pagos emergen.
    if(seen.current===null){seen.current=new Set(items.map(item=>item.id));return;}
    const fresh=items.filter(item=>!seen.current!.has(item.id));
    fresh.forEach(item=>seen.current!.add(item.id));
    if(fresh.length){
      void qc.invalidateQueries({queryKey:["staff-notices"]});
      const {alertPayments:alert,money:format,leader:announces}=announce.current;
      if(announces)alert(fresh.map(item=>({provider:item.provider,providerLabel:payProviderLabel[item.provider],amount:format(item.amount)})));
    }
    setVisible(current=>[...fresh,...current.filter(stale=>items.some(item=>item.id===stale.id))].slice(0,4));
  },[feed.data,qc]);

  const closeNotice=useCallback((id:string)=>setVisible(current=>current.filter(item=>item.id!==id)),[]);
  const openNotice=useCallback((item:PayNotification)=>{closeNotice(item.id);setDetail(item)},[closeNotice]);
  return <>
    {(visible.length>0||askSound)&&<div className="pay-notice-layer" role="region" aria-label="Pagos detectados por Yape y Plin">
      {askSound&&<Button kind="secondary" icon="volume" className="pay-sound-gate" onClick={enableSound}>Activar sonido de pagos</Button>}
      {visible.map(item=><PayNotice key={item.id} item={item} onOpen={openNotice} onClose={closeNotice}/>)}
    </div>}
    <PayNotificationDetail notification={detail} mode="associate" onClose={()=>setDetail(null)} onDone={()=>{setDetail(null);void feed.refetch();void qc.invalidateQueries({queryKey:["staff-notices"]})}}/>
  </>;
}

function PayNotice({item,onOpen,onClose}:{item:PayNotification;onOpen:(item:PayNotification)=>void;onClose:(id:string)=>void}){
  const money=useMoney();
  const [hovered,setHovered]=useState(false);
  const [focused,setFocused]=useState(false);
  const [hidden,setHidden]=useState(false);
  const remaining=useRef(NOTICE_DURATION_MS);
  useEffect(()=>{
    const sync=()=>setHidden(document.hidden);
    sync();document.addEventListener("visibilitychange",sync);
    return()=>document.removeEventListener("visibilitychange",sync);
  },[]);
  useEffect(()=>{
    if(hovered||focused||hidden)return;
    const started=Date.now();
    const timer=window.setTimeout(()=>onClose(item.id),remaining.current);
    return()=>{window.clearTimeout(timer);remaining.current=Math.max(0,remaining.current-(Date.now()-started))};
  },[item.id,onClose,hovered,focused,hidden]);
  return <section className="pay-notice" aria-label={`Pago detectado por ${payProviderLabel[item.provider]}`}
    onMouseEnter={()=>setHovered(true)} onMouseLeave={()=>setHovered(false)}
    onFocusCapture={()=>setFocused(true)} onBlurCapture={event=>{if(!event.currentTarget.contains(event.relatedTarget as Node|null))setFocused(false)}}>
    <button type="button" className="pay-notice-open" onClick={()=>onOpen(item)}>
      <PayProviderTag provider={item.provider}/>
      <span className="pay-notice-main"><strong>{money(item.amount)}</strong><small>{item.senderName||"Sin remitente"}</small><span className="pay-notice-hint">Pago detectado</span></span>
    </button>
    <IconButton icon="close" label="Cerrar aviso de pago" className="pay-notice-close" onClick={()=>onClose(item.id)}/>
  </section>;
}
