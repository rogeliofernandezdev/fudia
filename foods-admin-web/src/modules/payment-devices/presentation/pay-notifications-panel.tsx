"use client";

import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {IconButton,ConfirmDialog,EmptyState,ErrorState,SearchField,Select,Pagination,Status,TableSkeleton} from "@/design-system";
import {DataTable} from "@/design-system/data-table";
import {useFeedback,useMoney,useSession} from "@/providers";
import {useDebouncedValue} from "@/shared/hooks/use-debounced-value";
import {payProviderLabel,payStatusMeta} from "../domain/pay-status";
import type {PayNotification} from "../domain/types";
import {PayProviderTag} from "./pay-status-tag";
import {dismissPayNotification,listPayNotificationPage,payProviderOptions,payStatusOptions} from "../infrastructure/pay-devices-api";
import {PayNotificationDetail} from "./pay-notification-detail";

type Props={dateTime:(value:string,part?:"date"|"time")=>string};

/** Pestaña «Pagos detectados» de Caja: el cajero los revisa (filtros y
 * paginación en el servidor) y los asocia a un pedido; la asociación registra el cobro real con el flujo de
 * pagos existente. El aviso flotante global hace lo mismo desde cualquier
 * pantalla; esta lista es la consulta y la revisión posterior. */
const notificationColumns=["Proveedor","Remitente","Monto","Recepción","Estado","Acciones"];

export function PayNotificationsPanel({dateTime}:Props){
  const qc=useQueryClient();
  const {notify}=useFeedback();
  const money=useMoney();
  // Asociar y descartar exigen cash.manage en el API; quien solo ve Caja
  // (cash.read: Auditor, Contador) consulta el pago sin esas acciones.
  const canManage=useSession().can("cash.manage");
  const [status,setStatus]=useState("");
  const [provider,setProvider]=useState("");
  const [search,setSearch]=useState("");
  // Busca mientras escribe, en el servidor, con una sola consulta cuando deja de teclear.
  const q=useDebouncedValue(search.trim());
  const [page,setPage]=useState(1);
  const [size,setSize]=useState(20);
  const [dismissTarget,setDismissTarget]=useState<PayNotification|null>(null);
  const [detail,setDetail]=useState<PayNotification|null>(null);
  const [detailMode,setDetailMode]=useState<"detail"|"associate">("detail");
  const openPayment=(item:PayNotification,mode:"detail"|"associate")=>{setDetailMode(mode);setDetail(item)};

  const notifications=useQuery({
    queryKey:["pay-notifications",status,provider,q,page,size],
    queryFn:()=>listPayNotificationPage({status,provider,q,page,size}),
    placeholderData:previous=>previous,
    meta:{live:["pay"],poll:query=>(query.state.data as {items?:Array<{status:string}>}|undefined)?.items?.some(item=>item.status==="detected")?10000:30000},
  });

  const done=(message:string)=>{
    void qc.invalidateQueries({queryKey:["pay-notifications"]});
    notify({tone:"success",title:"Pago detectado actualizado",message});
  };
  const dismiss=useMutation({
    mutationFn:(id:string)=>dismissPayNotification(id),
    onSuccess:()=>{setDismissTarget(null);done("Quedó descartado con su traza.")},
    onError:(error:Error)=>notify({tone:"danger",title:"No se pudo descartar",message:error.message}),
  });

  const items=notifications.data?.items??[];
  const pending=items.some(item=>item.status==="detected");
  return <section className="pay-notifications-section" aria-label="Pagos detectados">
    <div className="pay-notifications-toolbar">
      <div className="pay-notifications-filters">
        <SearchField className="pay-notifications-search" value={search} onChange={event=>{setSearch(event.target.value);setPage(1)}} placeholder="Buscar remitente o monto" label="Buscar pagos por remitente o monto"/>
        <Select chip aria-label="Filtrar por estado" value={status} onChange={event=>{setStatus(event.target.value);setPage(1)}}>
          <option value="">Todos los estados</option>
          {payStatusOptions.map(value=><option key={value} value={value}>{payStatusMeta[value].label}</option>)}
        </Select>
        <Select chip aria-label="Filtrar por proveedor" value={provider} onChange={event=>{setProvider(event.target.value);setPage(1)}}>
          <option value="">Yape y Plin</option>
          {payProviderOptions.map(option=><option key={option} value={option}>{payProviderLabel[option]}</option>)}
        </Select>
      </div>
      {canManage&&<p>{pending?"Hay pagos por revisar: asócialos a su pedido.":"Asociar un pago registra el cobro en su pedido."}</p>}
    </div>
    {notifications.isLoading?<TableSkeleton columns={notificationColumns} rows={5} label="Cargando pagos detectados"/>
    :notifications.isError?<ErrorState className="pay-empty" title="No pudimos cargar los pagos detectados" text={notifications.error.message} onRetry={()=>void notifications.refetch()}/>
    :!items.length?<EmptyState className="pay-empty" icon="wallet" title={status||provider||q?"No hay pagos con estos filtros":"Aún no hay pagos detectados"} text={q?"Prueba con otro remitente o monto, o quita los filtros.":status||provider?"Prueba con otro estado o proveedor para encontrar el pago.":"Cuando un teléfono vinculado reciba un pago de Yape o Plin, aparecerá aquí para revisión."}/>
    :<DataTable label="Pagos detectados" containerClassName="hover-scroll pay-notifications-table" stickyActions>
      <thead><tr>{notificationColumns.map(column=><th scope="col" className={column==="Monto"?"pay-amount":column==="Proveedor"?"pay-col-provider":column==="Estado"?"pay-col-state":column==="Acciones"?"pay-col-actions":undefined} key={column}>{column==="Proveedor"?<span className="pay-col-provider-label">{column}</span>:column}</th>)}</tr></thead>
      <tbody>
        {items.map((item,index)=>{
          const meta=payStatusMeta[item.status];
          return <tr className={index%2?"alternate":""} key={item.id}>
            <td><PayProviderTag provider={item.provider} appearance="circle"/></td>
            <td className="pay-notification-sender"><strong>{item.senderName||"Sin remitente"}</strong>{item.operationNumber&&<small>Operación {item.operationNumber}</small>}</td>
            <td className="pay-amount">{money(item.amount)}</td>
            <td className="pay-notification-received"><time dateTime={item.occurredAt}>{dateTime(item.occurredAt,"date")}<small>{dateTime(item.occurredAt,"time")}</small></time></td>
            <td><Status tone={meta.tone} icon={item.status==="associated"?"circleCheck":undefined}>{meta.label}</Status></td>
            <td><div className="table-actions">
              <IconButton icon="eye" label={`Ver el detalle del pago de ${item.senderName||payProviderLabel[item.provider]}`} onClick={()=>openPayment(item,"detail")}/>
              {/* Asociar es la tarea pendiente del pago: azul sólido para distinguirla de Ver (2026-11-21). */}
              {canManage&&item.status==="detected"&&<IconButton icon="link" className="solid" label={`Asociar el pago de ${item.senderName||payProviderLabel[item.provider]} a un pedido`} onClick={()=>openPayment(item,"associate")}/>}
              {canManage&&item.status==="associated"&&<IconButton icon="cancel" className="danger" label={`Descartar el pago de ${item.senderName||payProviderLabel[item.provider]}`} onClick={()=>setDismissTarget(item)}/>}
            </div></td>
          </tr>;
        })}
      </tbody>
    </DataTable>}
    <PayNotificationDetail notification={detail} mode={detailMode} onClose={()=>setDetail(null)} onDone={()=>{setDetail(null);void notifications.refetch()}}/>
    {!notifications.isLoading&&!notifications.isError&&<Pagination page={page} size={size} total={notifications.data?.total??0} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>}
    <ConfirmDialog open={Boolean(dismissTarget)} tone="danger" title="Descartar el pago detectado"
      description="El evento queda registrado con su traza, pero ya no aparecerá como pendiente de revisión."
      subject={dismissTarget?{label:"Pago",name:`${payProviderLabel[dismissTarget.provider]} · ${dismissTarget.currency} ${dismissTarget.amount}`}:undefined}
      confirmLabel="Descartar" pending={dismiss.isPending} onCancel={()=>setDismissTarget(null)}
      onConfirm={()=>dismissTarget&&dismiss.mutate(dismissTarget.id)}/>
  </section>;
}
