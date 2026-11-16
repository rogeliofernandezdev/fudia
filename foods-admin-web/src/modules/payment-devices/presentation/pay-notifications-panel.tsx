"use client";

import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {IconButton,Button,ConfirmDialog,EmptyState,ErrorState,SearchField,Select,Pagination,Status,TableSkeleton,Tag} from "@/design-system";
import {DataTable} from "@/design-system/data-table";
import {Dialog} from "@/design-system/dialog";
import {ModalHeader} from "@/design-system";
import {useFeedback} from "@/providers";
import {payProviderLabel,payStatusMeta} from "../domain/pay-status";
import type {PayNotification} from "../domain/types";
import {associatePayNotification,dismissPayNotification,listPayNotificationPage,payProviderOptions,paySalesMethods,searchPayOrders} from "../infrastructure/pay-devices-api";

type Props={dateTime:(value:string)=>string};

/** Pestaña «Pagos» de Pagos detectados: el cajero los revisa (filtros y
 * paginación en el servidor) y los asocia a un pedido; la asociación registra el cobro real con el flujo de
 * pagos existente. El aviso flotante global hace lo mismo desde cualquier
 * pantalla; esta lista es la consulta y la revisión posterior. */
const notificationColumns=["Proveedor","Remitente","Monto","Hora","Dispositivo","Estado","Acciones"];

export function PayNotificationsPanel({dateTime}:Props){
  const qc=useQueryClient();
  const {notify}=useFeedback();
  const [status,setStatus]=useState("");
  const [provider,setProvider]=useState("");
  const [page,setPage]=useState(1);
  const [size,setSize]=useState(20);
  const [associateTarget,setAssociateTarget]=useState<PayNotification|null>(null);
  const [dismissTarget,setDismissTarget]=useState<PayNotification|null>(null);

  const notifications=useQuery({
    queryKey:["pay-notifications",status,provider,page,size],
    queryFn:()=>listPayNotificationPage({status,provider,page,size}),
    placeholderData:previous=>previous,
    refetchInterval:query=>query.state.data?.items.some(item=>item.status==="detected")?10000:30000,
  });

  const done=(message:string)=>{
    void qc.invalidateQueries({queryKey:["pay-notifications"]});
    notify({tone:"success",title:"Pago detectado actualizado",message});
  };
  const associate=useMutation({
    mutationFn:({id,orderId,method}:{id:string;orderId:string;method:string})=>associatePayNotification(id,orderId,method),
    onSuccess:result=>{setAssociateTarget(null);done(`Quedó asociado al pedido y el cobro quedó registrado (${result.currency} ${result.amount}).`)},
    onError:(error:Error)=>notify({tone:"danger",title:"No se pudo asociar",message:error.message}),
  });
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
        <Select chip aria-label="Filtrar por estado" value={status} onChange={event=>{setStatus(event.target.value);setPage(1)}}>
          <option value="">Todos los estados</option>
          {Object.entries(payStatusMeta).map(([value,meta])=><option key={value} value={value}>{meta.label}</option>)}
        </Select>
        <Select chip aria-label="Filtrar por proveedor" value={provider} onChange={event=>{setProvider(event.target.value);setPage(1)}}>
          <option value="">Yape y Plin</option>
          {payProviderOptions.map(option=><option key={option} value={option}>{payProviderLabel[option]}</option>)}
        </Select>
      </div>
      <p>{pending?"Hay pagos por revisar: asócialos a su pedido.":"Asociar un pago registra el cobro en su pedido."}</p>
    </div>
    {notifications.isLoading?<TableSkeleton columns={notificationColumns} rows={5} label="Cargando pagos detectados"/>
    :notifications.isError?<ErrorState className="pay-empty" title="No pudimos cargar los pagos detectados" text={notifications.error.message} onRetry={()=>void notifications.refetch()}/>
    :!items.length?<EmptyState className="pay-empty" icon="wallet" title="Aún no hay pagos detectados" text="Cuando un teléfono vinculado reciba un pago de Yape o Plin, aparecerá aquí para revisión."/>
    :<DataTable label="Pagos detectados" containerClassName="hover-scroll pay-notifications-table">
      <thead><tr>{notificationColumns.map(column=><th scope="col" className={column==="Monto"?"pay-amount":undefined} key={column}>{column}</th>)}</tr></thead>
      <tbody>
        {items.map((item,index)=>{
          const meta=payStatusMeta[item.status];
          return <tr className={index%2?"alternate":""} key={item.id}>
            <td><Tag>{payProviderLabel[item.provider]}</Tag></td>
            <td><strong>{item.senderName||"Sin remitente"}</strong></td>
            <td className="pay-amount">{item.currency} {item.amount}</td>
            <td>{dateTime(item.occurredAt)}</td>
            <td>{item.deviceName}</td>
            <td><Status tone={meta.tone} icon={item.status==="associated"?"circleCheck":undefined}>{meta.label}</Status></td>
            <td><div className="table-actions">
              {item.status==="detected"&&<IconButton icon="link" label={`Asociar el pago de ${item.senderName||payProviderLabel[item.provider]} a un pedido`} onClick={()=>setAssociateTarget(item)}/>}
              {item.status==="associated"&&<IconButton icon="cancel" className="danger" label={`Descartar el pago de ${item.senderName||payProviderLabel[item.provider]}`} onClick={()=>setDismissTarget(item)}/>}
            </div></td>
          </tr>;
        })}
      </tbody>
    </DataTable>}
    {!notifications.isLoading&&!notifications.isError&&<Pagination page={page} size={size} total={notifications.data?.total??0} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>}
    <AssociateDialog target={associateTarget} pending={associate.isPending} onCancel={()=>setAssociateTarget(null)}
      onConfirm={(id,orderId,method)=>associate.mutate({id,orderId,method})}/>
    <ConfirmDialog open={Boolean(dismissTarget)} tone="danger" title="Descartar el pago detectado"
      description="El evento queda registrado con su traza, pero ya no aparecerá como pendiente de revisión."
      subject={dismissTarget?{label:"Pago",name:`${payProviderLabel[dismissTarget.provider]} · ${dismissTarget.currency} ${dismissTarget.amount}`}:undefined}
      confirmLabel="Descartar" pending={dismiss.isPending} onCancel={()=>setDismissTarget(null)}
      onConfirm={()=>dismissTarget&&dismiss.mutate(dismissTarget.id)}/>
  </section>;
}

/** Asociar: elige el medio de pago del catálogo de la empresa, busca el pedido
 * con saldo pendiente y registra el cobro real. */
function AssociateDialog({target,pending,onCancel,onConfirm}:{target:PayNotification|null;pending:boolean;onCancel:()=>void;onConfirm:(id:string,orderId:string,method:string)=>void}){
  const [q,setQ]=useState("");
  const [method,setMethod]=useState("");
  const [order,setOrder]=useState<{id:string;code:string}|null>(null);
  const methods=useQuery({queryKey:["pay-sales-methods"],queryFn:paySalesMethods,enabled:Boolean(target)});
  const orders=useQuery({queryKey:["pay-associate-orders",q],queryFn:()=>searchPayOrders(q),enabled:Boolean(target)&&q.trim().length>=2,gcTime:0});
  // Medio por defecto: el del catálogo que corresponda a Yape o Plin.
  const effectiveMethod=method||(methods.data?.items.find(item=>/yape|plin/i.test(`${item.code} ${item.name}`))?.code??"");
  const close=()=>{onCancel();setMethod("");setOrder(null);setQ("")};
  if(!target)return null;
  return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal pay-order-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="pay-associate-title" aria-busy={pending}>
    <ModalHeader title="Asociar el pago a un pedido" titleId="pay-associate-title" onClose={close} closeLabel="Cerrar"/>
    <div className="pay-order-search">
      <p className="form-help">Pago: <strong>{payProviderLabel[target.provider]} · {target.currency} {target.amount}</strong> de {target.senderName||"remitente desconocido"}. La asociación registra el cobro real.</p>
      <label className="pay-detail-field">Medio de pago
        <Select value={effectiveMethod} onChange={event=>setMethod(event.target.value)}>
          <option value="">Elegir medio…</option>
          {methods.data?.items.map(item=><option key={item.code} value={item.code}>{item.name}</option>)}
        </Select>
      </label>
      <SearchField value={q} onChange={event=>{setQ(event.target.value);setOrder(null)}} placeholder="Buscar por código PED-…, mesa o cliente" label="Buscar pedido"/>
      {orders.data&&<ul className="pay-order-options">
        {orders.data.items.length?orders.data.items.map(item=><li key={item.id} className="pay-order-option">
          <span><strong>{item.code}</strong> {item.customerName??""}{item.total?` · ${item.total}`:""}</span>
          <Button kind="secondary" disabled={pending||!effectiveMethod} onClick={()=>{setOrder({id:item.id,code:item.code});onConfirm(target.id,item.id,effectiveMethod)}}>Asociar</Button>
        </li>):<li className="pay-order-option"><span>{orders.isFetching?"Buscando…":"Sin pedidos para esa búsqueda."}</span></li>}
      </ul>}
      {order&&<p className="form-help" role="status">Asociando a {order.code}…</p>}
    </div>
  </Dialog></div>;
}
