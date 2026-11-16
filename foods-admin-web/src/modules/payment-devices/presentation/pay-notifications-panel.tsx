"use client";

import {useEffect,useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,EmptyState,ErrorState,SearchField,Select,Status,TableSkeleton,Tag} from "@/design-system";
import {Dialog} from "@/design-system/dialog";
import {ModalHeader} from "@/design-system";
import {useFeedback} from "@/providers";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {payProviderLabel,payStatusMeta} from "../domain/pay-status";
import type {PayNotification} from "../domain/types";
import {associatePayNotification,dismissPayNotification,listPayNotifications,payProviderOptions,paySalesMethods,searchPayOrders} from "../infrastructure/pay-devices-api";

type Props={dateTime:(value:string)=>string};

/** Pagos detectados por los dispositivos: el cajero los revisa en Caja y los
 * asocia a un pedido; la asociación registra el cobro real con el flujo de
 * pagos existente. El aviso flotante global hace lo mismo desde cualquier
 * pantalla; esta lista es la consulta y la revisión posterior. */
export function PayNotificationsPanel({dateTime}:Props){
  const qc=useQueryClient();
  const {notify}=useFeedback();
  const [status,setStatus]=useState("");
  const [provider,setProvider]=useState("");
  const [associateTarget,setAssociateTarget]=useState<PayNotification|null>(null);
  const [dismissTarget,setDismissTarget]=useState<PayNotification|null>(null);

  const notifications=useQuery({
    queryKey:["pay-notifications",status,provider],
    queryFn:()=>listPayNotifications({status,provider}),
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
    <header className="pay-notifications-toolbar">
      <div><h2>Pagos detectados</h2><p>{pending?"Hay pagos por revisar.":"Los pagos que los dispositivos detecten aparecen aquí."}</p></div>
      <div className="pay-notifications-filters">
        <Select chip aria-label="Filtrar por estado" value={status} onChange={event=>setStatus(event.target.value)}>
          <option value="">Todos los estados</option>
          {Object.entries(payStatusMeta).map(([value,meta])=><option key={value} value={value}>{meta.label}</option>)}
        </Select>
        <Select chip aria-label="Filtrar por proveedor" value={provider} onChange={event=>setProvider(event.target.value)}>
          <option value="">Yape y Plin</option>
          {payProviderOptions.map(option=><option key={option} value={option}>{payProviderLabel[option]}</option>)}
        </Select>
      </div>
    </header>
    {notifications.isLoading?<TableSkeleton columns={["Proveedor","Remitente","Monto","Hora","Dispositivo","Estado","Acciones"]} rows={5} label="Cargando pagos detectados"/>
    :notifications.isError?<ErrorState title="No pudimos cargar los pagos detectados" text={notifications.error.message} onRetry={()=>void notifications.refetch()}/>
    :!items.length?<EmptyState icon="wallet" title="Aún no hay pagos detectados" text="Cuando un teléfono vinculado reciba un pago de Yape o Plin, aparecerá aquí para revisión."/>
    :<table className="pay-devices-table">
      <thead><tr><th>Proveedor</th><th>Remitente</th><th>Monto</th><th>Hora</th><th>Dispositivo</th><th>Estado</th><th className="pay-devices-actions">Acciones</th></tr></thead>
      <tbody>
        {items.map(item=>{
          const meta=payStatusMeta[item.status];
          return <tr key={item.id}>
            <td><Tag>{payProviderLabel[item.provider]}</Tag></td>
            <td><strong>{item.senderName||"Sin remitente"}</strong></td>
            <td>{item.currency} {item.amount}</td>
            <td>{dateTime(item.occurredAt)}</td>
            <td>{item.deviceName}</td>
            <td><Status tone={meta.tone} icon={item.status==="associated"?"circleCheck":undefined}>{meta.label}</Status></td>
            <td className="pay-devices-actions">
              {item.status==="detected"&&<Button kind="secondary" onClick={()=>setAssociateTarget(item)}>Asociar</Button>}
              {item.status==="associated"&&<Button kind="secondary" onClick={()=>setDismissTarget(item)}>Descartar</Button>}
            </td>
          </tr>;
        })}
      </tbody>
    </table>}
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
