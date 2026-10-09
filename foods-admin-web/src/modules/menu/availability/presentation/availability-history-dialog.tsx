"use client";
import {useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {Button,Dialog,Icon,Pagination} from "@/design-system";
import {useSession} from "@/providers/session-context";
import {formatRegionalCalendarDate,formatRegionalDateTime,formatRegionalNumber} from "@/shared/i18n/regional-format";
import {listAvailabilityHistory} from "../infrastructure/availability-api";
import type {AvailabilityHistoryItem,AvailabilityItem} from "../domain/types";

export function AvailabilityHistoryDialog({item,close}:{item:AvailabilityItem;close:()=>void}){
 const{organization,location,can}=useSession();
 const[page,setPage]=useState(1),[size,setSize]=useState(10);
 const query=useQuery({queryKey:["availability-history",organization?.id,location?.id,item.productId,page,size],queryFn:()=>listAvailabilityHistory(item.productId,page,size),enabled:Boolean(location&&can("menu.read")),staleTime:0});
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal availability-history-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="availability-history-title">
  <div className="modal-accent"/>
  <header><span className="modal-title-icon"><Icon name="clock"/></span><div><small>HISTORIAL DE DISPONIBILIDAD</small><h2 id="availability-history-title">{item.name}</h2></div><button type="button" aria-label="Cerrar" onClick={close}><Icon name="close"/></button></header>
  <div className="availability-history-body">
   {!can("menu.read")?<div className="catalog-state"><b>Sin permiso de lectura</b></div>:query.isLoading?<div className="table-skeleton" aria-label="Cargando historial"><div className="sk-head">{Array.from({length:4},(_,i)=><i key={i}/>)}</div>{Array.from({length:5},(_,i)=><div className="sk-row" key={i}><i className="sk-name"><span/><b/><small/></i><i/><i/><i/></div>)}</div>:query.isError?<div className="catalog-state error"><b>No pudimos cargar el historial</b><p>{query.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>query.refetch()}>Reintentar</Button></div>:!query.data?.items.length?<div className="catalog-state"><Icon name="clock"/><b>Aún no hay cambios registrados</b><p>Los cambios de disponibilidad aparecerán aquí.</p></div>:<>
    <div className="table-wrap hover-scroll" tabIndex={0}><table className="availability-history-table"><thead><tr><th>FECHA Y HORA</th><th>CAMBIO</th><th>MOTIVO</th><th>USUARIO</th></tr></thead><tbody>{query.data.items.map(entry=><tr key={entry.id}><td><time dateTime={entry.createdAt}>{formatRegionalDateTime(entry.createdAt,{country:location?.country,timeZone:location?.timezone})}</time>{entry.businessDate&&<small>Día {formatRegionalCalendarDate(entry.businessDate,location?.country)}</small>}</td><td><HistoryChange entry={entry} country={location?.country}/></td><td>{entry.reason||"Sin motivo registrado (cambio anterior)"}</td><td>{entry.userName}</td></tr>)}</tbody></table></div>
    <Pagination page={page} size={size} total={query.data.total} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>
   </>}
  </div>
 </Dialog></div>;
}

export function HistoryChange({entry,country}:{entry:AvailabilityHistoryItem;country?:string}){
 if(!entry.businessDate)return <>Cambio anterior sin detalle registrado</>;
 const number=(value:number|null)=>value===null?"Sin registrar":formatRegionalNumber(value,country);
 const quantityChanged=entry.previousPortionQuantity!==entry.portionQuantity;
 const statusChanged=entry.previousManualStatus!==entry.manualStatus;
 const status=(value:string|null)=>value==="sold_out"?"Agotado":value==="available"?"Disponible":"Sin registrar";
 return <>{quantityChanged&&<b>Cupo: {number(entry.previousPortionQuantity)} → {number(entry.portionQuantity)}</b>}{statusChanged&&<span>{status(entry.previousManualStatus)} → {status(entry.manualStatus)}</span>}{!quantityChanged&&!statusChanged&&<span>Disponibilidad registrada</span>}{entry.soldQuantity!==null&&entry.portionQuantity!==null&&<small>{formatRegionalNumber(entry.soldQuantity,country)} vendidas al cambiar</small>}</>;
}
