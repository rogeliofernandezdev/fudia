"use client";
import {DataTable} from "@/design-system/data-table";
import {Dialog} from "@/design-system/dialog";
import "./reservations.css";
import {useState} from "react";
import {useForm,useWatch} from "react-hook-form";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,EmptyState,ErrorState,FieldLabel,Icon,Input,PageHeader,Pagination,RowActionButton,Select,Status,TableSkeleton,Textarea,ModalHeader,ModalFooter,ModalForm,ModalBody} from "@/design-system";
import {useFeedback,useSession} from "@/providers";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {listTables} from "../../tables/infrastructure/tables-api";
import {reservationResolver} from "../domain/reservation-schema";
import type {Reservation,ReservationDraft,ReservationStatus} from "../domain/types";
import {listReservations,saveReservation,setReservationStatus} from "../infrastructure/reservations-api";

const columns=["CLIENTE","FECHA Y HORA","PERSONAS","DURACIÓN","MESA","ESTADO","ACCIONES"];
const blank:ReservationDraft={customerName:"",customerPhone:"",startsAt:"",guests:"2",durationMinutes:"90",tableId:"",notes:""};
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
  const edit=(item:Reservation)=>setDraft({id:item.id,customerName:item.customerName,customerPhone:item.customerPhone,startsAt:toLocalInput(item.startsAt),guests:String(item.guests),durationMinutes:String(item.durationMinutes),tableId:item.tableId??"",notes:item.notes});
  const items=list.data?.items??[];
  return <><PageHeader title="Reservas" description="Agenda clientes, asigna mesas y controla asistencia por local." action={canManage?<Button icon="plus" onClick={()=>setDraft({...blank})}>Nueva reserva</Button>:undefined}/>
    <section className="panel management standardized-management">
      <div className="toolbar"><label><Icon name="search" size={18}/><Input value={q} onChange={e=>{setQ(e.target.value);setPage(1)}} placeholder="Buscar cliente, teléfono o mesa..."/></label><Select value={status} onChange={e=>{setStatus(e.target.value);setPage(1)}}><option value="">Todos los estados</option>{Object.entries(statusMeta).map(([value,meta])=><option value={value} key={value}>{meta.label}</option>)}</Select></div>
      {list.isLoading?<TableSkeleton columns={columns} label="Cargando reservas"/>:list.isError?<ErrorState title="No pudimos cargar las reservas" text={list.error.message} onRetry={()=>void list.refetch()} retrying={list.isFetching}/>:!items.length?(q||status?<EmptyState icon="search" title="Sin coincidencias" text="No hay reservas que coincidan con la búsqueda o el estado."/>:<EmptyState icon="clock" title="Aún no hay reservas" text="Registra la primera reserva del local para comenzar." action={canManage?<Button icon="plus" onClick={()=>setDraft({...blank})}>Nueva reserva</Button>:undefined}/>):<DataTable label="Listado de reservas" containerClassName="hover-scroll"><thead><tr>{columns.map(column=><th key={column}>{column}</th>)}</tr></thead><tbody>{items.map((item,i)=>{const meta=statusMeta[item.status];return <tr className={i%2?"alternate":""} key={item.id}><td><span className="row-icon"><Icon name="users"/></span><b>{item.customerName}</b><small>{item.customerPhone||item.notes||"Sin teléfono"}</small></td><td>{date(item.startsAt)}</td><td>{item.guests}</td><td>{item.durationMinutes} min</td><td>{item.tableName||"Por asignar"}</td><td><Status tone={meta.tone}>{meta.label}</Status></td><td><div className="table-actions">{canManage&&item.status!=="cancelled"&&item.status!=="no_show"&&<RowActionButton action="edit" onClick={()=>edit(item)}/>} {canManage&&item.status==="pending"&&<Button kind="ghost" onClick={()=>setStatusTarget({item,status:"confirmed"})}>Confirmar</Button>} {canManage&&item.status==="confirmed"&&<Button kind="ghost" onClick={()=>setStatusTarget({item,status:"seated"})}>Sentar</Button>} {canManage&&["pending","confirmed"].includes(item.status)&&<RowActionButton action="cancel" label="Cancelar reserva" onClick={()=>setStatusTarget({item,status:"cancelled"})}/>}</div></td></tr>})}</tbody></DataTable>}
      {!list.isLoading&&!list.isError&&<Pagination page={page} size={size} total={list.data?.total??0} onPage={setPage} onSize={v=>{setSize(v);setPage(1)}}/>}
    </section>
    {draft&&<ReservationDialog value={draft} tables={tables.data?.items??[]} tablesLoading={tables.isLoading} tablesError={tables.isError?tables.error.message:""} busy={save.isPending} close={()=>setDraft(null)} save={v=>save.mutate(v)}/>}
    <ConfirmDialog open={Boolean(statusTarget)} title="Actualizar reserva" description={statusTarget?`Cambiar ${statusTarget.item.customerName} a “${statusMeta[statusTarget.status].label}”.`:""} confirmLabel="Confirmar" pending={change.isPending} onCancel={()=>setStatusTarget(null)} onConfirm={()=>statusTarget&&change.mutate({id:statusTarget.item.id,status:statusTarget.status})}/>
  </>;
}

function ReservationDialog({value:initial,tables,tablesLoading,tablesError,busy,close,save}:{value:ReservationDraft;tables:Array<{id:string;name:string;seats:number;active:boolean}>;tablesLoading:boolean;tablesError:string;busy:boolean;close:()=>void;save:(v:ReservationDraft)=>void}){
  const{
    control,register,handleSubmit,
    formState:{errors},
  }=useForm<ReservationDraft>({
    defaultValues:initial,
    resolver:reservationResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });
  const guests=Number(useWatch({control,name:"guests"})||0);
  const eligibleTables=tables.filter(table=>table.id===initial.tableId||(table.active&&table.seats>=guests));

  return <div className="modal-backdrop modal-overlay-in" role="presentation">
    <Dialog onResponseClose={close} className="crud-modal reservation-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="reservation-title" aria-busy={busy}>

      <ModalHeader title="Datos de la reserva" titleId="reservation-title" icon="clock" subtitle={initial.id?"EDITAR RESERVA":"NUEVA RESERVA"} onClose={close} busy={busy} closeLabel="Cerrar"/>
      <ModalForm onSubmit={handleSubmit(save)} noValidate>
        <ModalBody><div className="form-grid reservation-form-grid">
          <label>
            Cliente
            <Input autoFocus autoComplete="name" maxLength={180} {...register("customerName")} aria-invalid={Boolean(errors.customerName)} placeholder="Nombre del cliente"/>
            {errors.customerName?.message&&<small className="field-error">{errors.customerName.message}</small>}
          </label>
          <label>
            Teléfono
            <Input type="tel" autoComplete="tel" maxLength={40} {...register("customerPhone")} aria-invalid={Boolean(errors.customerPhone)} placeholder="Ej. 999 999 999"/>
            {errors.customerPhone?.message&&<small className="field-error">{errors.customerPhone.message}</small>}
          </label>

          <label>
            Fecha y hora
            <Input type="datetime-local" {...register("startsAt")} aria-invalid={Boolean(errors.startsAt)}/>
            {errors.startsAt?.message&&<small className="field-error">{errors.startsAt.message}</small>}
          </label>
          <label>
            Personas
            <Input type="number" inputMode="numeric" min="1" max="100" {...register("guests")} aria-invalid={Boolean(errors.guests)}/>
            {errors.guests?.message&&<small className="field-error">{errors.guests.message}</small>}
          </label>

          <label>
            Mesa
            <Select {...register("tableId")}>
              <option value="">Asignar después</option>
              {eligibleTables.map(table=><option key={table.id} value={table.id}>{table.name} · {table.seats} personas</option>)}
            </Select>
            {tablesLoading&&<small className="reservation-table-state">Cargando mesas disponibles…</small>}
            {!tablesLoading&&tablesError&&<small className="reservation-table-state error">No pudimos cargar las mesas. Puedes guardar sin asignar.</small>}
            {!tablesLoading&&!tablesError&&!eligibleTables.length&&<small className="reservation-table-state">No hay mesas activas con capacidad suficiente.</small>}
          </label>
          <label>
            <FieldLabel hint="Tiempo durante el cual la mesa queda ocupada para evitar reservas solapadas.">Duración de mesa (min)</FieldLabel>
            <Input type="number" inputMode="numeric" min="15" max="360" step="15" {...register("durationMinutes")} aria-invalid={Boolean(errors.durationMinutes)}/>
            {errors.durationMinutes?.message&&<small className="field-error">{errors.durationMinutes.message}</small>}
          </label>

          <label className="reservation-notes">
            <span className="reservation-label">Notas <small>Opcional</small></span>
            <Textarea maxLength={500} rows={3} {...register("notes")} aria-invalid={Boolean(errors.notes)} placeholder="Preferencias, ocasión o indicaciones para el equipo"/>
            {errors.notes?.message&&<small className="field-error">{errors.notes.message}</small>}
          </label>
        </div></ModalBody>
        <ModalFooter>
          <Button kind="ghost" onClick={close} disabled={busy}>Cancelar</Button>
          <Button type="submit" disabled={busy}>{busy?"Guardando…":"Guardar"}</Button>
        </ModalFooter>
      </ModalForm>
      {busy&&<div className="modal-busy" role="status"><i/><span>Guardando…</span></div>}
    </Dialog>
  </div>;
}
