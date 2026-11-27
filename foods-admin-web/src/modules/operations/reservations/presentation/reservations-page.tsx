"use client";
import {DataTable} from "@/design-system/data-table";
import {Dialog} from "@/design-system/dialog";
import "./reservations.css";
import {useState} from "react";
import {useForm,useWatch} from "react-hook-form";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,EmptyState,ErrorState,FormField,FormSection,Icon,IconButton,Input,MoneyInput,PageHeader,Pagination,RowActionButton,Select,Status,TableSkeleton,Tabs,Textarea,ModalHeader,ModalFooter,ModalForm,ModalBody} from "@/design-system";
import {useCurrency,useFeedback,useMoney,useSession} from "@/providers";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {listTables} from "../../tables/infrastructure/tables-api";
import {parseReservationDate,reservationResolver} from "../domain/reservation-schema";
import type {Reservation,ReservationDepositDecision,ReservationDraft,ReservationStatus} from "../domain/types";
import {chargeReservationDeposit,listReservations,refundReservationDeposit,saveReservation,setReservationStatus} from "../infrastructure/reservations-api";
import {ReservationDepositDialog} from "./reservation-deposit-dialog";

const columns=["Cliente","Fecha y hora","Personas","Duración","Mesa","Estado","Acciones"];
/** Valores del local (Locales) para una reserva nueva: 15 min de tolerancia y sin garantía si la lista aún no los trae. */
const fallbackDefaults={toleranceMinutes:15,depositAmount:"0.00"};
/** «0», «0.00» o vacío significan sin garantía. */
function hasDeposit(amount:string){return Number(amount)>0}
/** Matriz de estados de Reservas (2026-11-27): Pendiente → Confirmada → En mesa → Asistió (al terminar su tiempo), o No asistió / Cancelada. */
const statusMeta:Record<ReservationStatus,{label:string;tone:"green"|"blue"|"sky"|"orange"|"red"|"gray"}>={
  pending:{label:"Pendiente",tone:"orange"},confirmed:{label:"Confirmada",tone:"blue"},seated:{label:"En mesa",tone:"sky"},attended:{label:"Asistió",tone:"green"},
  cancelled:{label:"Cancelada",tone:"red"},no_show:{label:"No asistió",tone:"gray"},
};
/** Fecha y hora locales del navegador para los campos «Fecha» y «Hora». */
function localParts(value:Date){
  if(Number.isNaN(value.getTime()))return {date:"",time:""};
  const pad=(n:number)=>String(n).padStart(2,"0");
  return {date:`${value.getFullYear()}-${pad(value.getMonth()+1)}-${pad(value.getDate())}`,time:`${pad(value.getHours())}:${pad(value.getMinutes())}`};
}
/** «Garantía S/ 50.00 · pagada con Yape»: el estado del cobro bajo el cliente; sin garantía, el teléfono. */
function depositText(item:Reservation,money:(value:number)=>string){
  const amount=`Garantía ${money(Number(item.depositAmount))}`;
  switch(item.depositStatus){
    case "pending":return `${amount} · por cobrar`;
    case "paid":return `${amount} · pagada con ${item.depositMethodName}`;
    case "applied":return `${amount} · descontada`;
    case "retained":return [amount,"retenida",item.depositDocument].filter(Boolean).join(" · ");
    case "refunded":return `${amount} · devuelta`;
    default:return "";
  }
}
function blankDraft(defaults:{toleranceMinutes:number;depositAmount:string}):ReservationDraft{
  return {customerName:"",customerPhone:"",date:localParts(new Date()).date,time:"",guests:"2",durationMinutes:"90",toleranceMinutes:String(defaults.toleranceMinutes),depositEnabled:hasDeposit(defaults.depositAmount),depositAmount:hasDeposit(defaults.depositAmount)?defaults.depositAmount:"",tableId:"",notes:""};
}

export function ReservationsPage(){
  const qc=useQueryClient();const{notify}=useFeedback();const{can,location}=useSession();
  const canManage=can("reservations.manage");
  const[q,setQ]=useState("");const[status,setStatus]=useState("");const[page,setPage]=useState(1);const[size,setSize]=useState(10);
  const[draft,setDraft]=useState<ReservationDraft|null>(null);
  const[statusTarget,setStatusTarget]=useState<{item:Reservation;status:ReservationStatus}|null>(null);
  const[depositDecision,setDepositDecision]=useState<ReservationDepositDecision>("refund");
  const[seatTarget,setSeatTarget]=useState<Reservation|null>(null);const[seatTable,setSeatTable]=useState("");
  const[chargeTarget,setChargeTarget]=useState<Reservation|null>(null);
  const[refundTarget,setRefundTarget]=useState<Reservation|null>(null);
  const money=useMoney();
  const list=useQuery({queryKey:["reservations",q,status,page,size],queryFn:()=>listReservations({q,status,page,pageSize:size}),meta:{live:["reservations"],poll:30000}});
  const tables=useQuery({queryKey:["reservation-tables"],queryFn:()=>listTables({q:"",status:"active",page:1,pageSize:100})});
  const save=useMutation({mutationFn:saveReservation,onSuccess:()=>{setDraft(null);void qc.invalidateQueries({queryKey:["reservations"]});notify({tone:"success",title:"Reserva guardada",message:"La reserva quedó disponible para el equipo del local."})},onError:e=>notify({tone:"danger",title:"No se pudo guardar",message:e.message})});
  const refreshCash=()=>{for(const key of ["reservations","cash-shift","cash-registers","pos-order","sales","dashboard"])void qc.invalidateQueries({queryKey:[key]})};
  const change=useMutation({mutationFn:({id,status,deposit,tableId}:{id:string;status:ReservationStatus;deposit?:ReservationDepositDecision;tableId?:string})=>setReservationStatus(id,status,{deposit,tableId}),onSuccess:(_,variables)=>{setStatusTarget(null);setSeatTarget(null);refreshCash();notify({tone:"success",title:"Reserva actualizada",message:variables.deposit==="refund"?"La garantía se devolvió desde tu turno de caja.":variables.deposit==="retain"?"La garantía quedó en el local con su comprobante.":"El estado se guardó correctamente."})},onError:e=>notify({tone:"danger",title:"No se pudo actualizar",message:e.message})});
  const charge=useMutation({mutationFn:({id,draft}:{id:string;draft:Parameters<typeof chargeReservationDeposit>[1]})=>chargeReservationDeposit(id,draft),onSuccess:receipt=>{setChargeTarget(null);refreshCash();notify({tone:"success",title:"Garantía cobrada",message:`Se registró ${money(Number(receipt.amount))} con ${receipt.methodName} en tu turno de caja.`})},onError:e=>notify({tone:"danger",title:"No se pudo cobrar la garantía",message:e.message})});
  const refund=useMutation({mutationFn:(id:string)=>refundReservationDeposit(id),onSuccess:()=>{setRefundTarget(null);refreshCash();notify({tone:"success",title:"Garantía devuelta",message:"La devolución quedó registrada en tu turno de caja."})},onError:e=>notify({tone:"danger",title:"No se pudo devolver la garantía",message:e.message})});
  const depositHeld=statusTarget?.status==="cancelled"&&statusTarget.item.depositStatus==="paid";
  const date=(value:string)=>formatRegionalDateTime(value,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium",timeStyle:"short"});
  const time=(value:string)=>formatRegionalDateTime(value,{country:location?.country,timeZone:location?.timezone},{timeStyle:"short"});
  const waitUntil=(item:Reservation)=>{const start=parseReservationDate(item.startsAt).getTime();return Number.isNaN(start)?item.startsAt:new Date(start+item.toleranceMinutes*60000).toISOString()};
  const create=()=>setDraft(blankDraft(list.data?{toleranceMinutes:list.data.defaultToleranceMinutes,depositAmount:list.data.defaultDepositAmount}:fallbackDefaults));
  const edit=(item:Reservation)=>setDraft({id:item.id,customerName:item.customerName,customerPhone:item.customerPhone,...localParts(parseReservationDate(item.startsAt)),guests:String(item.guests),durationMinutes:String(item.durationMinutes),toleranceMinutes:String(item.toleranceMinutes),depositEnabled:hasDeposit(item.depositAmount),depositAmount:hasDeposit(item.depositAmount)?item.depositAmount:"",depositLocked:item.depositStatus!=="none"&&item.depositStatus!=="pending",tableId:item.tableId??"",notes:item.notes});
  const items=list.data?.items??[];
  return <><PageHeader title="Reservas" description="Agenda clientes, asigna mesas y controla asistencia por local." action={canManage?<Button icon="plus" onClick={create}>Nueva reserva</Button>:undefined}/>
    <section className="panel management standardized-management">
      <div className="toolbar"><label><Icon name="search" size={18}/><Input value={q} onChange={e=>{setQ(e.target.value);setPage(1)}} placeholder="Buscar cliente, teléfono o mesa..."/></label><Select chip value={status} onChange={e=>{setStatus(e.target.value);setPage(1)}}><option value="">Todos los estados</option>{Object.entries(statusMeta).map(([value,meta])=><option value={value} key={value}>{meta.label}</option>)}</Select></div>
      {list.isLoading?<TableSkeleton columns={columns} label="Cargando reservas"/>:list.isError?<ErrorState title="No pudimos cargar las reservas" text={list.error.message} onRetry={()=>void list.refetch()} retrying={list.isFetching}/>:!items.length?(q||status?<EmptyState icon="search" title="Sin coincidencias" text="No hay reservas que coincidan con la búsqueda o el estado."/>:<EmptyState icon="clock" title="Aún no hay reservas" text="Registra la primera reserva del local para comenzar." action={canManage?<Button icon="plus" onClick={create}>Nueva reserva</Button>:undefined}/>):<DataTable label="Listado de reservas" containerClassName="hover-scroll" className="reservations-table"><thead><tr>{columns.map(column=><th key={column}>{column}</th>)}</tr></thead><tbody>{items.map((item,i)=>{const meta=statusMeta[item.status];return <tr className={i%2?"alternate":""} key={item.id}><td><span className="row-icon"><Icon name="users"/></span><b>{item.customerName}</b><small>{depositText(item,money)||item.customerPhone||item.notes||"Sin teléfono"}</small></td><td>{date(item.startsAt)}{item.status==="pending"||item.status==="confirmed"?<small>Espera hasta {time(waitUntil(item))}</small>:item.expired?<small>Cancelada automáticamente</small>:null}</td><td>{item.guests}</td><td>{item.durationMinutes} min</td><td>{item.tableName||"Por asignar"}</td><td><Status tone={meta.tone}>{meta.label}</Status></td><td><div className="table-actions">{canManage&&<>
        {item.status==="pending"&&<IconButton icon="check" label="Confirmar reserva" className="reservation-action-confirm" onClick={()=>setStatusTarget({item,status:"confirmed"})}/>}
        {item.status==="confirmed"&&<IconButton icon="tables" label="Ubicar en mesa" className="reservation-action-seat" onClick={()=>{setSeatTable(item.tableId??"");setSeatTarget(item)}}/>}
        {item.depositStatus==="pending"&&["pending","confirmed"].includes(item.status)&&<IconButton icon="wallet" label="Cobrar garantía" className="reservation-action-deposit" onClick={()=>setChargeTarget(item)}/>}
        {(item.status==="seated"||item.status==="attended")&&item.depositStatus==="paid"&&<IconButton icon="undo" label="Devolver garantía" className="reservation-action-deposit" onClick={()=>setRefundTarget(item)}/>}
        {["pending","confirmed"].includes(item.status)&&<RowActionButton action="edit" onClick={()=>edit(item)}/>}
        {["pending","confirmed"].includes(item.status)&&<RowActionButton action="cancel" label="Cancelar reserva" onClick={()=>{setDepositDecision("refund");setStatusTarget({item,status:"cancelled"})}}/>}
      </>}</div></td></tr>})}</tbody></DataTable>}
      {!list.isLoading&&!list.isError&&<Pagination page={page} size={size} total={list.data?.total??0} onPage={setPage} onSize={v=>{setSize(v);setPage(1)}}/>}
    </section>
    {draft&&<ReservationDialog value={draft} tables={tables.data?.items??[]} tablesLoading={tables.isLoading} tablesError={tables.isError?tables.error.message:""} busy={save.isPending} close={()=>setDraft(null)} save={v=>save.mutate(v)}/>}
    <ConfirmDialog open={Boolean(statusTarget)} title={depositHeld?"Cancelar reserva":"Actualizar reserva"} description={statusTarget?(depositHeld?`${statusTarget.item.customerName} pagó una garantía de ${money(Number(statusTarget.item.depositAmount))} con ${statusTarget.item.depositMethodName}.`:`Cambiar ${statusTarget.item.customerName} a “${statusMeta[statusTarget.status].label}”.`):""} confirmLabel="Confirmar" pending={change.isPending} onCancel={()=>setStatusTarget(null)} onConfirm={()=>statusTarget&&change.mutate({id:statusTarget.item.id,status:statusTarget.status,...(depositHeld?{deposit:depositDecision}:{})})}>
      {depositHeld&&<div className="reservation-cancel-deposit">
        <Tabs label="Qué hacer con la garantía" value={depositDecision} onChange={setDepositDecision} items={[{value:"refund",label:"Devolver",icon:"undo"},{value:"retain",label:"Retener",icon:"wallet"}]}/>
        <p>{depositDecision==="refund"?"Sale de tu turno de caja con el mismo medio.":"Queda en el local como venta, con su boleta."}</p>
      </div>}
    </ConfirmDialog>
    <ConfirmDialog open={Boolean(seatTarget)} tone="success" title="Ubicar en mesa" description={seatTarget?`Elige la mesa donde se sentó ${seatTarget.customerName}.`:""} confirmLabel="Ubicar en mesa" pending={change.isPending} confirmDisabled={!seatTable} onCancel={()=>setSeatTarget(null)} onConfirm={()=>seatTarget&&seatTable&&change.mutate({id:seatTarget.id,status:"seated",tableId:seatTable})}>
      {seatTarget&&<FormField label="Mesa" help={tables.isLoading?"Cargando mesas…":tables.isError?"No pudimos cargar las mesas.":undefined}><Select value={seatTable} onChange={e=>setSeatTable(e.target.value)}>
        <option value="">Elige una mesa</option>
        {(tables.data?.items??[]).filter(table=>table.id===seatTarget.tableId||(table.active&&table.seats>=seatTarget.guests)).map(table=><option key={table.id} value={table.id}>{table.name} · {table.seats} personas</option>)}
      </Select></FormField>}
    </ConfirmDialog>
    {chargeTarget&&<ReservationDepositDialog reservation={chargeTarget} busy={charge.isPending} formatMoney={money} formatDate={date} close={()=>setChargeTarget(null)} save={draft=>charge.mutate({id:chargeTarget.id,draft})}/>}
    <ConfirmDialog open={Boolean(refundTarget)} title="Devolver garantía" description={refundTarget?`Se devuelven ${money(Number(refundTarget.depositAmount))} a ${refundTarget.customerName} desde tu turno de caja.`:""} confirmLabel="Devolver garantía" pending={refund.isPending} onCancel={()=>setRefundTarget(null)} onConfirm={()=>refundTarget&&refund.mutate(refundTarget.id)}/>
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
  const currency=useCurrency();
  const guests=Number(useWatch({control,name:"guests"})||0);
  const depositEnabled=useWatch({control,name:"depositEnabled"});
  const eligibleTables=tables.filter(table=>table.id===initial.tableId||(table.active&&table.seats>=guests));
  const tablesHelp=tablesLoading?"Cargando mesas disponibles…":tablesError?"No pudimos cargar las mesas. Puedes guardar sin asignar.":!eligibleTables.length?"No hay mesas activas con capacidad suficiente.":undefined;

  return <div className="modal-backdrop modal-overlay-in" role="presentation">
    <Dialog onResponseClose={close} className="crud-modal reservation-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="reservation-title" aria-busy={busy}>
      <ModalHeader title={initial.id?"Editar reserva":"Nueva reserva"} titleId="reservation-title" onClose={close} busy={busy} closeLabel="Cerrar"/>
      <ModalForm onSubmit={handleSubmit(save)} noValidate>
        <ModalBody><div className="form-sections-flow">
          <FormSection icon="user" title="Cliente">
            <FormField label="Nombre" error={errors.customerName?.message}><Input autoFocus autoComplete="name" maxLength={180} {...register("customerName")} placeholder="Ej. Ana Pérez"/></FormField>
            <FormField label="Teléfono" error={errors.customerPhone?.message}><Input type="tel" inputMode="tel" autoComplete="tel" maxLength={40} {...register("customerPhone")} placeholder="Ej. 999 999 999"/></FormField>
          </FormSection>
          <FormSection icon="calendar" tone="teal" title="Fecha y mesa" className="reservation-schedule">
            <FormField className="reservation-date" label="Fecha" icon={false} error={errors.date?.message}><Input type="date" {...register("date")}/></FormField>
            <FormField className="reservation-time" label="Hora" icon={false} error={errors.time?.message}><Input type="time" step="300" {...register("time")}/></FormField>
            <FormField label="Personas" error={errors.guests?.message}><Input type="number" inputMode="numeric" min="1" max="100" {...register("guests")}/></FormField>
            <FormField label="Mesa" help={tablesHelp}><Select {...register("tableId")}>
              <option value="">Asignar después</option>
              {eligibleTables.map(table=><option key={table.id} value={table.id}>{table.name} · {table.seats} personas</option>)}
            </Select></FormField>
            <FormField label="Duración de mesa (min)" error={errors.durationMinutes?.message}><Input type="number" inputMode="numeric" min="15" max="360" step="1" {...register("durationMinutes")}/></FormField>
            <FormField label="Tolerancia (min)" icon="hourglass" help="Luego pasa a «No asistió»." error={errors.toleranceMinutes?.message}><Input type="number" inputMode="numeric" min="0" max="240" step="1" {...register("toleranceMinutes")}/></FormField>
          </FormSection>
          <FormSection icon="wallet" tone="brand" title="Garantía">
            <label className={depositEnabled?"switch-row":"switch-row span-2"}><input type="checkbox" disabled={initial.depositLocked} {...register("depositEnabled")}/><span/><b>Cobrar garantía</b></label>
            {depositEnabled&&<FormField label="Importe" help={initial.depositLocked?"Ya se cobró en Caja; el importe no cambia.":"Se cobra en Caja y se descuenta de la cuenta."} error={errors.depositAmount?.message}><MoneyInput symbol={currency.symbol} position={currency.position} placeholder="0.00" disabled={initial.depositLocked} {...register("depositAmount")}/></FormField>}
          </FormSection>
          <FormSection icon="note" tone="neutral" title="Información adicional">
            <FormField className="span-2" label="Notas" optional error={errors.notes?.message}><Textarea maxLength={500} rows={3} {...register("notes")} placeholder="Preferencias, ocasión o indicaciones para el equipo"/></FormField>
          </FormSection>
        </div></ModalBody>
        <ModalFooter>
          <Button kind="ghost" onClick={close} disabled={busy}>Cancelar</Button>
          <Button type="submit" icon="check" disabled={busy} loading={busy}>{busy?"Guardando…":"Guardar"}</Button>
        </ModalFooter>
      </ModalForm>
    </Dialog>
  </div>;
}
