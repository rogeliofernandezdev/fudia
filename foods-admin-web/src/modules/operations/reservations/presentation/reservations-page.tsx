"use client";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,Icon,Input,PageHeader,Pagination,RowActionButton,Select,Status,Textarea} from "@/design-system";
import {useFeedback,useSession} from "@/providers";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {listTables} from "../../tables/infrastructure/tables-api";
import type {Reservation,ReservationDraft,ReservationStatus} from "../domain/types";
import {listReservations,saveReservation,setReservationStatus} from "../infrastructure/reservations-api";

const blank:ReservationDraft={customerName:"",customerPhone:"",startsAt:"",guests:"2",tableId:"",notes:""};
const statusMeta:Record<ReservationStatus,{label:string;tone:"green"|"blue"|"orange"|"gray"}>={
  pending:{label:"Pendiente",tone:"orange"},confirmed:{label:"Confirmada",tone:"blue"},seated:{label:"Sentada",tone:"green"},
  cancelled:{label:"Cancelada",tone:"gray"},no_show:{label:"No asistió",tone:"gray"},
};
function toLocalInput(value:string){
  const d=new Date(value);if(Number.isNaN(d.getTime()))return"";
  const pad=(n:number)=>String(n).padStart(2,"0");
  return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export function ReservationsPage(){
  const qc=useQueryClient();const{notify}=useFeedback();const{can,location}=useSession();
  const canManage=can("reservations.manage");
  const[q,setQ]=useState("");const[status,setStatus]=useState("");const[page,setPage]=useState(1);const[size,setSize]=useState(10);
  const[draft,setDraft]=useState<ReservationDraft|null>(null);
  const[statusTarget,setStatusTarget]=useState<{item:Reservation;status:ReservationStatus}|null>(null);
  const list=useQuery({queryKey:["reservations",q,status,page,size],queryFn:()=>listReservations({q,status,page,pageSize:size}),refetchInterval:30000});
  const tables=useQuery({queryKey:["reservation-tables"],queryFn:()=>listTables({q:"",status:"active",page:1,pageSize:100})});
  const save=useMutation({mutationFn:saveReservation,onSuccess:()=>{setDraft(null);void qc.invalidateQueries({queryKey:["reservations"]});notify({tone:"success",title:"Reserva guardada",message:"La reserva quedó disponible para el equipo del local."})},onError:e=>notify({tone:"danger",title:"No se pudo guardar",message:e.message})});
  const change=useMutation({mutationFn:({id,status}:{id:string;status:ReservationStatus})=>setReservationStatus(id,status),onSuccess:()=>{setStatusTarget(null);void qc.invalidateQueries({queryKey:["reservations"]});notify({tone:"success",title:"Reserva actualizada",message:"El estado se guardó correctamente."})},onError:e=>notify({tone:"danger",title:"No se pudo actualizar",message:e.message})});
  const date=(value:string)=>formatRegionalDateTime(value,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium",timeStyle:"short"});
  const edit=(item:Reservation)=>setDraft({id:item.id,customerName:item.customerName,customerPhone:item.customerPhone,startsAt:toLocalInput(item.startsAt),guests:String(item.guests),tableId:item.tableId??"",notes:item.notes});
  const items=list.data?.items??[];
  return <><PageHeader eyebrow="OPERACIÓN" title="Reservas" description="Agenda clientes, asigna mesas y controla asistencia por local." action={canManage?<Button icon="plus" onClick={()=>setDraft({...blank})}>Nueva reserva</Button>:undefined}/>
    <section className="panel management standardized-management">
      <div className="toolbar"><label><Icon name="search" size={18}/><Input value={q} onChange={e=>{setQ(e.target.value);setPage(1)}} placeholder="Buscar cliente, teléfono o mesa..."/></label><Select value={status} onChange={e=>{setStatus(e.target.value);setPage(1)}}><option value="">Todos los estados</option>{Object.entries(statusMeta).map(([value,meta])=><option value={value} key={value}>{meta.label}</option>)}</Select></div>
      {list.isLoading?<div className="catalog-state"><b>Cargando reservas…</b></div>:list.isError?<div className="catalog-state error"><span><Icon name="alert"/></span><b>No pudimos cargar las reservas</b><p>{list.error.message}</p></div>:!items.length?<div className="catalog-state"><span><Icon name="clock"/></span><b>Sin reservas registradas</b><p>Registra la primera reserva del local para comenzar.</p></div>:<div className="table-wrap hover-scroll"><table><thead><tr><th>CLIENTE</th><th>FECHA Y HORA</th><th>PERSONAS</th><th>MESA</th><th>ESTADO</th><th>ACCIONES</th></tr></thead><tbody>{items.map((item,i)=>{const meta=statusMeta[item.status];return <tr className={i%2?"alternate":""} key={item.id}><td><span className="row-icon"><Icon name="users"/></span><b>{item.customerName}</b><small>{item.customerPhone||item.notes||"Sin teléfono"}</small></td><td>{date(item.startsAt)}</td><td>{item.guests}</td><td>{item.tableName||"Por asignar"}</td><td><Status tone={meta.tone}>{meta.label}</Status></td><td><div className="table-actions">{canManage&&item.status!=="cancelled"&&item.status!=="no_show"&&<RowActionButton action="edit" onClick={()=>edit(item)}/>} {canManage&&item.status==="pending"&&<Button kind="ghost" onClick={()=>setStatusTarget({item,status:"confirmed"})}>Confirmar</Button>} {canManage&&item.status==="confirmed"&&<Button kind="ghost" onClick={()=>setStatusTarget({item,status:"seated"})}>Sentar</Button>} {canManage&&["pending","confirmed"].includes(item.status)&&<RowActionButton action="deactivate" label="Cancelar reserva" onClick={()=>setStatusTarget({item,status:"cancelled"})}/>}</div></td></tr>})}</tbody></table></div>}
      {!list.isLoading&&!list.isError&&<Pagination page={page} size={size} total={list.data?.total??0} onPage={setPage} onSize={v=>{setSize(v);setPage(1)}}/>}
    </section>
    {draft&&<ReservationDialog value={draft} tables={tables.data?.items??[]} busy={save.isPending} close={()=>setDraft(null)} save={v=>save.mutate(v)}/>}
    <ConfirmDialog open={Boolean(statusTarget)} title="Actualizar reserva" description={statusTarget?`Cambiar ${statusTarget.item.customerName} a “${statusMeta[statusTarget.status].label}”.`:""} confirmLabel="Confirmar" pending={change.isPending} onCancel={()=>setStatusTarget(null)} onConfirm={()=>statusTarget&&change.mutate({id:statusTarget.item.id,status:statusTarget.status})}/>
  </>;
}

function ReservationDialog({value:initial,tables,busy,close,save}:{value:ReservationDraft;tables:Array<{id:string;name:string;seats:number;active:boolean}>;busy:boolean;close:()=>void;save:(v:ReservationDraft)=>void}){
  const[value,setValue]=useState(initial);
  return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal modal-panel-in" role="dialog" aria-modal="true"><div className="modal-accent"/><header><span className="modal-title-icon"><Icon name="clock"/></span><div><small>{value.id?"EDITAR RESERVA":"NUEVA RESERVA"}</small><h2>Datos de la reserva</h2></div><button aria-label="Cerrar" onClick={close}><Icon name="close"/></button></header><form onSubmit={e=>{e.preventDefault();save(value)}}><div className="form-grid"><label>Cliente<Input required maxLength={180} value={value.customerName} onChange={e=>setValue({...value,customerName:e.target.value})}/></label><label>Teléfono<Input maxLength={40} value={value.customerPhone} onChange={e=>setValue({...value,customerPhone:e.target.value})}/></label><label>Fecha y hora<Input required type="datetime-local" value={value.startsAt} onChange={e=>setValue({...value,startsAt:e.target.value})}/></label><label>Personas<Input required type="number" min="1" max="100" value={value.guests} onChange={e=>setValue({...value,guests:e.target.value})}/></label><label className="span-2">Mesa<Select value={value.tableId} onChange={e=>setValue({...value,tableId:e.target.value})}><option value="">Asignar después</option>{tables.filter(t=>t.active&&t.seats>=Number(value.guests||0)).map(t=><option key={t.id} value={t.id}>{t.name} · {t.seats} personas</option>)}</Select></label><label className="span-2">Notas<Textarea maxLength={500} rows={3} value={value.notes} onChange={e=>setValue({...value,notes:e.target.value})}/></label></div><footer><Button kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy||!value.startsAt}>{busy?"Guardando…":"Guardar reserva"}</Button></footer></form></section></div>;
}
