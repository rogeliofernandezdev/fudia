"use client";
import {useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {Button,Icon,Pagination,Modal,ModalBody} from "@/design-system";
import {useSession} from "@/providers/session-context";
import {formatRegionalCalendarDate,formatRegionalDateTime,formatRegionalNumber} from "@/shared/i18n/regional-format";
import {listAvailabilityHistory} from "../infrastructure/availability-api";
import type {AvailabilityHistoryItem,AvailabilityItem} from "../domain/types";

export function AvailabilityHistoryDialog({item,close}:{item:AvailabilityItem;close:()=>void}){
 const{organization,location,can}=useSession();
 const[page,setPage]=useState(1),[size,setSize]=useState(10);
 const query=useQuery({queryKey:["availability-history",organization?.id,location?.id,item.productId,page,size],queryFn:()=>listAvailabilityHistory(item.productId,page,size),enabled:Boolean(location&&can("menu.read")),staleTime:0});
 return <Modal title="Historial de disponibilidad" subtitle={item.name} icon="clock" size="xl" onClose={close} className="availability-history-modal">
  <ModalBody className="availability-history-body">
   {!can("menu.read")?<div className="catalog-state"><b>Sin permiso de lectura</b></div>:query.isLoading?<div className="table-skeleton" aria-label="Cargando historial"><div className="sk-head">{Array.from({length:5},(_,i)=><i key={i}/>)}</div>{Array.from({length:5},(_,i)=><div className="sk-row" key={i}><i className="sk-name"><span/><b/><small/></i><i/><i/><i/><i/></div>)}</div>:query.isError?<div className="catalog-state error"><b>No pudimos cargar el historial</b><p>{query.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>query.refetch()}>Reintentar</Button></div>:!query.data?.items.length?<div className="catalog-state"><Icon name="clock"/><b>Aún no hay cambios registrados</b><p>Los cambios de disponibilidad aparecerán aquí.</p></div>:<>
    <div className="table-wrap hover-scroll availability-history-scroll" role="region" aria-label={`Historial de ${item.name}; tabla desplazable`} tabIndex={0}><table className="availability-history-table"><thead><tr><th scope="col">Fecha y hora</th><th scope="col">Cambio</th><th scope="col">Cantidad</th><th scope="col">Motivo</th><th scope="col">Usuario</th></tr></thead><tbody>{query.data.items.map(entry=>{const createdDate=formatRegionalDateTime(entry.createdAt,{country:location?.country,timeZone:location?.timezone},{dateStyle:"short"});const businessDate=entry.businessDate?formatRegionalCalendarDate(entry.businessDate,location?.country):null;return <tr key={entry.id}><td><time dateTime={entry.createdAt}><span>{createdDate}</span><small>{formatRegionalDateTime(entry.createdAt,{country:location?.country,timeZone:location?.timezone},{timeStyle:"short"})}</small></time>{businessDate&&businessDate!==createdDate&&<small className="availability-history-day">Día {businessDate}</small>}</td><td><HistoryChange entry={entry}/></td><td><HistoryQuantity entry={entry} country={location?.country} control={item.quantityControl}/></td><td>{entry.reason||<span className="availability-history-unrecorded">Sin motivo registrado</span>}</td><td>{entry.userName}</td></tr>})}</tbody></table></div>
    <Pagination page={page} size={size} total={query.data.total} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>
   </>}
  </ModalBody>
 </Modal>;
}

export function HistoryChange({entry}:{entry:AvailabilityHistoryItem}){
 if(!entry.businessDate)return <span className="availability-history-unrecorded">Cambio anterior sin detalle</span>;
 const quantityChanged=entry.previousPortionQuantity!==entry.portionQuantity;
 const statusChanged=entry.previousManualStatus!==entry.manualStatus;
 const status=(value:string|null)=>value==="sold_out"?"Agotado":value==="available"?"Disponible":"Sin registrar";
 return <>{quantityChanged&&<span>Cupo actualizado</span>}{statusChanged&&<span>{status(entry.previousManualStatus)} → {status(entry.manualStatus)}</span>}{!quantityChanged&&!statusChanged&&<span>Disponibilidad registrada</span>}</>;
}

/** Solo cantidades del evento: nunca completar históricos con el saldo actual. */
export function HistoryQuantity({entry,country,control}:{entry:AvailabilityHistoryItem;country?:string;control:AvailabilityItem["quantityControl"]}){
 const before=entry.previousPortionQuantity,after=entry.portionQuantity;
 if(before===null&&after===null)return <span className="availability-history-unrecorded">{entry.businessDate&&control==="none"?"No aplica":"Sin registrar"}</span>;
 const number=(value:number|null)=>value===null?"—":formatRegionalNumber(value,country);
 return <div className="availability-history-quantity">
  {before!==after?<span aria-label={`Cupo anterior: ${before===null?"sin registrar":number(before)}. Cupo nuevo: ${after===null?"sin registrar":number(after)}.`}>{number(before)} <span aria-hidden="true">→</span> <strong>{number(after)}</strong></span>:<strong>{number(after)}</strong>}
  {entry.soldQuantity!==null&&<small>{formatRegionalNumber(entry.soldQuantity,country)} vendidas al cambiar</small>}
 </div>;
}
