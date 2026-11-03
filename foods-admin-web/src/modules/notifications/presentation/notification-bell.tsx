"use client";
import "./notification-bell.css";
import {useEffect,useId,useRef,useState} from "react";
import {useRouter} from "next/navigation";
import {Button,EmptyState,ErrorState,Icon,IconButton,Skeleton,type IconName} from "@/design-system";
import {useNotices} from "../application/use-notices";
import {noticeAge,noticeTargets,unreadLabel} from "../domain/notice-rules";
import type {NoticeKind,StaffNotice} from "../domain/types";

/** Icono de cada aviso: el mismo dominio que la pantalla donde se atiende. */
const noticeIcons:Record<NoticeKind,IconName>={table_order:"tables",counter_order:"store",delivery_order:"motorbike",pickup_order:"shoppingBag",order_ready:"checks",bill_requested:"receipt",assistance_requested:"bell"};

/**
 * Campana del personal: componente independiente que consulta sus propios
 * avisos. El contenedor solo indica qué pantallas puede abrir el usuario.
 */
export function NotificationBell({canOpen=()=>true}:{canOpen?:(href:string)=>boolean}){
 const{enabled,feed,read,readAll}=useNotices();
 const router=useRouter();
 const[open,setOpen]=useState(false);
 const[announcement,setAnnouncement]=useState("");
 const ref=useRef<HTMLDivElement>(null);
 const triggerRef=useRef<HTMLButtonElement>(null);
 const seen=useRef<Set<string>|null>(null);
 const panelId=useId();
 const unread=feed.data?.unread??0;

 // Anuncia por lector de pantalla solo los avisos que llegan después de la primera carga.
 useEffect(()=>{
  const items=feed.data?.items;
  if(!items)return;
  if(seen.current){const fresh=items.find(item=>!item.read&&!seen.current!.has(item.id));if(fresh)setAnnouncement(`Nuevo aviso: ${fresh.title}`)}
  seen.current=new Set(items.map(item=>item.id));
 },[feed.data]);
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
  const target=noticeTargets(notice).find(canOpen);
  if(target)router.push(target);
 }
 const items=feed.data?.items??[];
 const label=unread?`Avisos: ${unread} sin leer`:"Avisos";
 return <div className="notification-bell" ref={ref}>
  <button ref={triggerRef} type="button" className="notification-trigger" data-open={open} aria-label={label} title="Avisos" aria-haspopup="dialog" aria-expanded={open} aria-controls={open?panelId:undefined} onClick={()=>{if(!open)void feed.refetch();setOpen(value=>!value)}}>
   <Icon name="bell" size={20}/>
   {unread>0&&<span className="notification-count" aria-hidden="true">{unreadLabel(unread)}</span>}
  </button>
  <span className="notification-live" role="status" aria-live="polite">{announcement}</span>
  {open&&<section id={panelId} className="notification-popover" role="dialog" aria-modal="false" aria-labelledby={`${panelId}-title`} aria-busy={feed.isLoading}>
   <header className="notification-head">
    <span className="notification-head-icon"><Icon name="bell" size={18}/></span>
    <h2 id={`${panelId}-title`}>Avisos</h2>
    {unread>0&&<Button kind="ghost" className="notification-read-all" disabled={readAll.isPending} onClick={()=>readAll.mutate()}>Marcar como leídos</Button>}
    <IconButton icon="close" label="Cerrar avisos" className="notification-close" onClick={()=>{setOpen(false);triggerRef.current?.focus()}}/>
   </header>
   {feed.isLoading?<div className="notification-loading" role="status" aria-label="Cargando avisos">{Array.from({length:3},(_,index)=><div key={index}><Skeleton width="var(--size-36)" height="var(--size-36)" radius="var(--radius-tile)"/><span><Skeleton width="70%"/><Skeleton width="45%"/></span></div>)}</div>
   :feed.isError&&!feed.data?<ErrorState className="notification-state" title="No pudimos cargar los avisos" text={feed.error.message} onRetry={()=>void feed.refetch()} retrying={feed.isFetching}/>
   :!items.length?<EmptyState className="notification-state" icon="bell" title="Sin avisos" text="Aquí verás los pedidos, cuentas y llamados de las últimas 24 horas."/>
   :<ul className="notification-list">{items.map(item=><li key={item.id}><button type="button" className="notification-item" data-read={item.read} onClick={()=>choose(item)}>
     <span className="notification-item-icon"><Icon name={noticeIcons[item.kind]??"bell"} size={18}/></span>
     <span className="notification-item-copy"><b>{item.title}</b>{item.detail&&<small>{item.detail}</small>}<time dateTime={item.createdAt}>{noticeAge(item.createdAt)}</time></span>
     {!item.read&&<i className="notification-unread" aria-label="Sin leer"/>}
    </button></li>)}</ul>}
  </section>}
 </div>;
}
