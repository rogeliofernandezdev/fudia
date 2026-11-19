"use client";

import {useCallback,useEffect,useRef,useState} from "react";
import {useQuery,useQueryClient} from "@tanstack/react-query";
import {IconButton} from "@/design-system";
import {useMoney,useSession} from "@/providers";
import {PayProviderTag} from "./pay-status-tag";
import {payProviderLabel} from "../domain/pay-status";
import type {PayNotification} from "../domain/types";
import {listPayNotifications} from "../infrastructure/pay-devices-api";
import {PayNotificationDetail} from "./pay-notification-detail";

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
  const feed=useQuery({
    queryKey:["pay-notice-feed",...scopeKey],
    queryFn:()=>listPayNotifications({status:"detected",provider:""}),
    refetchInterval:10000,
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
    if(fresh.length)void qc.invalidateQueries({queryKey:["staff-notices"]});
    setVisible(current=>[...fresh,...current.filter(stale=>items.some(item=>item.id===stale.id))].slice(0,4));
  },[feed.data,qc]);

  const closeNotice=useCallback((id:string)=>setVisible(current=>current.filter(item=>item.id!==id)),[]);
  const openNotice=useCallback((item:PayNotification)=>{closeNotice(item.id);setDetail(item)},[closeNotice]);
  return <>
    {visible.length>0&&<div className="pay-notice-layer" role="region" aria-label="Pagos detectados por Yape y Plin">
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
