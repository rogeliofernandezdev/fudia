"use client";
import "./notification-bell.css";
import {useEffect,useId,useRef,useState} from "react";
import {createPortal} from "react-dom";
import {useRouter} from "next/navigation";
import {Button,EmptyState,ErrorState,Icon,IconButton,IconTile,Skeleton,type IconName} from "@/design-system";
import {PayProviderTag,usePayLeader} from "@/modules/payment-devices";
import {useSession} from "@/providers/session-context";
import {useStoredFlag} from "@/shared/hooks/use-stored-flag";
import {useNotices} from "../application/use-notices";
import {freshNotices,noticeAge,noticeHref,noticePaymentSummary,noticeTargets,noticeTone,unreadLabel} from "../domain/notice-rules";
import {noticeSoundFor} from "../domain/notice-sound";
import type {NoticeKind,StaffNotice} from "../domain/types";
import {useNoticeSound} from "./use-notice-sound";

/** Icono de cada aviso: el mismo dominio que la pantalla donde se atiende. */
const noticeIcons:Record<NoticeKind,IconName>={table_order:"tables",counter_order:"store",delivery_order:"motorbike",pickup_order:"shoppingBag",order_ready:"checks",bill_requested:"receipt",assistance_requested:"bell",stale_account:"clock",delivery_unassigned:"motorbike",delivery_pickup_delayed:"clock",pay_detected:"wallet"};
/** El aviso emergente se retira solo; se pausa mientras el usuario lo señala o enfoca, o con la pestaña oculta. */
const toastDuration=8000;
/** Salida más rápida que la entrada (`--duration-fast`): el sistema responde, no hace esperar. */
const toastExit=160;

/**
 * Campana del personal: componente independiente que consulta sus propios
 * avisos. El contenedor solo indica qué pantallas puede abrir el usuario.
 */
export function NotificationBell({canOpen=()=>true}:{canOpen?:(href:string)=>boolean}){
 const{enabled,feed,read,readAll}=useNotices();
 const{user,organization,location}=useSession();
 const scope=[organization?.id,location?.id,user?.id].join(":");
 // Sonido de la campana: suena en una sola pestaña por usuario y local, y cada
 // usuario puede silenciarlo en este navegador (los pagos conservan el suyo).
 const soundLeader=usePayLeader(`fudia-notice-sound:${scope}`);
 const[muted,setMuted]=useStoredFlag(`fudia-notice-sound-muted:${user?.id??""}`);
 const playSound=useNoticeSound();
 const sound=useRef({soundLeader,muted,playSound});
 useEffect(()=>{sound.current={soundLeader,muted,playSound}},[soundLeader,muted,playSound]);
 const router=useRouter();
 const[open,setOpen]=useState(false);
 const[announcement,setAnnouncement]=useState("");
 const[toast,setToast]=useState<{notice:StaffNotice;more:number}|null>(null);
 const[toastHovered,setToastHovered]=useState(false);
 const[toastLeaving,setToastLeaving]=useState(false);
 const[pageHidden,setPageHidden]=useState(false);
 // Tiempo restante del aviso visible: al pausar se conserva y al reanudar continúa, igual que su barra.
 const toastClock=useRef({id:"",left:toastDuration});
 const toastPaused=toastHovered||pageHidden;
 const ref=useRef<HTMLDivElement>(null);
 const triggerRef=useRef<HTMLButtonElement>(null);
 const seen=useRef<Set<string>|null>(null);
 const openRef=useRef(open);
 const panelId=useId();
 const unread=feed.data?.unread??0;
 useEffect(()=>{openRef.current=open},[open]);

 // Solo los avisos que llegan después de la primera carga se anuncian y, con la campana cerrada, emergen.
 useEffect(()=>{
  const items=feed.data?.items;
  if(!items)return;
  if(seen.current){
   const fresh=freshNotices(items,seen.current);
   if(fresh.length){
    setAnnouncement(`Nuevo aviso: ${fresh[0].title}`);
    // Los pagos tienen su aviso con logo y acceso al detalle en PayNoticeLayer.
    const popup=fresh.filter(item=>item.kind!=="pay_detected");
    const tone=noticeSoundFor(fresh);
    if(tone&&sound.current.soundLeader&&!sound.current.muted)sound.current.playSound(tone);
    if(!openRef.current&&popup.length){
     setToast({notice:popup[0],more:popup.length-1});setToastHovered(false);setToastLeaving(false);
     if(typeof navigator!=="undefined"&&navigator.userActivation?.hasBeenActive)navigator.vibrate?.([120,60,120]);
    }
   }
  }
  seen.current=new Set(items.map(item=>item.id));
 },[feed.data]);
 useEffect(()=>{
  const sync=()=>setPageHidden(document.hidden);
  document.addEventListener("visibilitychange",sync);
  return()=>document.removeEventListener("visibilitychange",sync);
 },[]);
 useEffect(()=>{
  if(!toast||toastPaused||toastLeaving)return;
  const clock=toastClock.current;
  if(clock.id!==toast.notice.id){clock.id=toast.notice.id;clock.left=toastDuration}
  const startedAt=Date.now();
  const timer=window.setTimeout(()=>setToastLeaving(true),clock.left);
  return()=>{window.clearTimeout(timer);clock.left=Math.max(0,clock.left-(Date.now()-startedAt))};
 },[toast,toastPaused,toastLeaving]);
 // Sale hacia arriba, por donde entró; con movimiento reducido se retira sin animación.
 useEffect(()=>{
  if(!toastLeaving)return;
  const reduced=window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  const timer=window.setTimeout(()=>{setToast(null);setToastLeaving(false)},reduced?0:toastExit);
  return()=>window.clearTimeout(timer);
 },[toastLeaving]);
 useEffect(()=>{
  if(!open)return;
  const close=(event:MouseEvent)=>{if(!ref.current?.contains(event.target as Node))setOpen(false)};
  const escape=(event:KeyboardEvent)=>{if(event.key==="Escape"){setOpen(false);triggerRef.current?.focus()}};
  document.addEventListener("mousedown",close);document.addEventListener("keydown",escape);
  return()=>{document.removeEventListener("mousedown",close);document.removeEventListener("keydown",escape)};
 },[open]);
 if(!enabled)return null;

 function choose(notice:StaffNotice){
  read(notice.id);
  setOpen(false);
  setToast(null);
  const target=noticeTargets(notice).find(canOpen);
  if(target)router.push(noticeHref(target,notice));
 }
 const items=feed.data?.items??[];
 const label=unread?`Avisos: ${unread} sin leer`:"Avisos";
 return <div className="notification-bell" ref={ref}>
  <button ref={triggerRef} type="button" className="notification-trigger" data-open={open} aria-label={label} title="Avisos" aria-haspopup="dialog" aria-expanded={open} aria-controls={open?panelId:undefined} onClick={()=>{if(!open)void feed.refetch();setToast(null);setOpen(value=>!value)}}>
   <Icon name="bell" size={20}/>
   {unread>0&&<span className="notification-count" aria-hidden="true">{unreadLabel(unread)}</span>}
  </button>
  <span className="notification-live" role="status" aria-live="polite">{announcement}</span>
  {open&&<section id={panelId} className="notification-popover" role="dialog" aria-modal="false" aria-labelledby={`${panelId}-title`} aria-busy={feed.isLoading}>
   <header className="notification-head">
    <h2 id={`${panelId}-title`}>Avisos</h2>
    {unread>0&&<span className="notification-new">{unread===1?"1 nuevo":`${unreadLabel(unread)} nuevos`}</span>}
    {unread>0&&<Button kind="ghost" className="notification-read-all" disabled={readAll.isPending} onClick={()=>readAll.mutate()}>Marcar como leídos</Button>}
    <IconButton icon="close" label="Cerrar avisos" className="notification-close" onClick={()=>{setOpen(false);triggerRef.current?.focus()}}/>
   </header>
   {feed.isLoading?<div className="notification-loading" role="status" aria-label="Cargando avisos">{Array.from({length:4},(_,index)=><div key={index}><Skeleton width="var(--size-40)" height="var(--size-40)" radius="var(--radius-tile)"/><span><Skeleton width="70%"/><Skeleton width="45%"/></span></div>)}</div>
   :feed.isError&&!feed.data?<ErrorState className="notification-state" title="No pudimos cargar los avisos" text={feed.error.message} onRetry={()=>void feed.refetch()} retrying={feed.isFetching}/>
   :!items.length?<EmptyState className="notification-state" icon="bell" title="Sin avisos" text="Aquí verás los pedidos, pagos, cuentas y llamados de las últimas 24 horas."/>
   :<ul className="notification-list">{items.map(item=>{
    // Pagos como movimiento: quien pagó, «Yape · hace 5 min» y el monto a la derecha; los demás, título y «detalle · hace N».
    const payment=noticePaymentSummary(item);
    const origin=payment?payment.wallet:item.detail;
    return <li key={item.id}><button type="button" className="notification-item" data-read={item.read} data-payment={Boolean(payment)} onClick={()=>choose(item)}>
     {payment?<PayProviderTag provider={payment.provider} appearance="circle" decorative/>:<IconTile name={noticeIcons[item.kind]??"bell"} tone={item.read?"neutral":noticeTone(item.kind)} size="lg"/>}
     <span className="notification-item-copy"><b>{payment?.title??item.title}</b><small>{origin&&`${origin} · `}<time dateTime={item.createdAt}>{noticeAge(item.createdAt).toLowerCase()}</time></small></span>
     <span className="notification-item-end">{payment?.amount&&<strong className="notification-amount">{payment.amount}</strong>}{!item.read&&<i className="notification-unread" aria-label="Sin leer"/>}</span>
    </button></li>})}</ul>}
   {/* Al pie y no en la cabecera: en móvil la cabecera ya ocupa su ancho. Mismo interruptor que el panel de reparto. */}
   <button type="button" className="notification-sound" aria-pressed={!muted} onClick={()=>setMuted(!muted)}><b>Avisos con sonido</b><i aria-hidden="true"/></button>
  </section>}
  {/* Portal: la barra superior crea su propio contexto y atraparía el `position:fixed` del aviso. */}
  {/* La `key` reinicia entrada y barra de tiempo cuando un aviso nuevo reemplaza al visible. */}
  {toast&&createPortal(<section key={toast.notice.id} className="notification-toast" aria-label="Aviso nuevo" data-paused={toastPaused} data-leaving={toastLeaving} onMouseEnter={()=>setToastHovered(true)} onMouseLeave={()=>setToastHovered(false)} onFocus={()=>setToastHovered(true)} onBlur={()=>setToastHovered(false)}>
   {/* Todo el aviso abre el detalle, igual que en la lista: sin botón «Ver» redundante. */}
   <button type="button" className="notification-toast-main" onClick={()=>choose(toast.notice)}>
    <IconTile name={noticeIcons[toast.notice.kind]??"bell"} tone={noticeTone(toast.notice.kind)} size="lg"/>
    <span className="notification-toast-copy">
     <span className="notification-toast-title"><b>{toast.notice.title}</b><time dateTime={toast.notice.createdAt}>{noticeAge(toast.notice.createdAt)}</time></span>
     {toast.notice.detail&&<small>{toast.notice.detail}</small>}
     {toast.more>0&&<em>{`y ${toast.more} ${toast.more===1?"aviso más":"avisos más"}`}</em>}
    </span>
   </button>
   <IconButton icon="close" label="Descartar aviso" className="notification-toast-close" onClick={()=>setToastLeaving(true)}/>
   <span className="notification-toast-timer" aria-hidden="true" style={{animationDuration:`${toastDuration}ms`}}/>
  </section>,document.body)}
 </div>;
}
