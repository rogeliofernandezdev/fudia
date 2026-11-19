"use client";

import {useEffect,useRef,useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,EmptyState,ErrorState,Icon,Modal,ModalBody,ModalFooter,SearchField,Skeleton,Status,Tag} from "@/design-system";
import {DetailFacts} from "@/design-system/detail-view";
import {useFeedback,useMoney,useSession} from "@/providers";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {payProviderLabel,payStatusMeta} from "../domain/pay-status";
import type {PayNotification,PayOrderRef} from "../domain/types";
import {PayProviderTag} from "./pay-status-tag";
import {associatePayNotification,payOrderDetail,paySalesMethods,searchPayOrders} from "../infrastructure/pay-devices-api";
import {usePayCollection} from "./pay-collection-context";

type PayNotificationMode="detail"|"associate";
type Props={notification:PayNotification|null;mode?:PayNotificationMode;onClose:()=>void;onDone:()=>void};

const channelLabel=(channel:string)=>({salon:"Mesa",delivery:"Delivery",mostrador:"Venta rápida",recojo:"Recojo",whatsapp:"WhatsApp"}[channel]??channel);

/** Dos intenciones con destinos distintos: «detail» es una consulta de solo
 * lectura (sin acciones: asociar tiene su propio botón en la tabla y en el
 * aviso) y «associate» es la tarea de cobro, donde revisar el pedido y su
 * saldo siempre precede al registro. */
export function PayNotificationDetail({notification,mode="detail",onClose,onDone}:Props){
  if(!notification)return null;
  return <DetailBody key={`${notification.id}:${mode}`} notification={notification} mode={mode} onClose={onClose} onDone={onDone}/>;
}

function DetailBody({notification,mode,onClose,onDone}:Omit<Props,"notification"|"mode">&{notification:PayNotification;mode:PayNotificationMode}){
  const qc=useQueryClient();
  const {notify}=useFeedback();
  const money=useMoney();
  const {location}=useSession();
  const {open}=usePayCollection();
  const [view,setView]=useState<"suggestion"|"orders"|"detail">(open?"suggestion":"orders");
  const [q,setQ]=useState("");
  const [selected,setSelected]=useState<PayOrderRef|null>(null);
  const associationHeading=useRef<HTMLHeadingElement>(null);
  const associating=notification.status==="detected"&&mode==="associate";
  useEffect(()=>{if(associating)associationHeading.current?.focus({preventScroll:true})},[associating,view]);
  const dateTime=(value:string)=>formatRegionalDateTime(value,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium",timeStyle:"short"});

  const methods=useQuery({queryKey:["pay-sales-methods"],queryFn:paySalesMethods,enabled:associating});
  // El código estable del catálogo identifica al proveedor, aunque se renombre.
  // El logo del resumen ya dice el medio: no se repite en un selector bloqueado.
  const detectedMethod=methods.data?.items.find(item=>item.code===notification.provider);
  const effectiveMethod=detectedMethod?.code??"";
  const methodReady=Boolean(detectedMethod)&&!methods.isError;
  const pending=useQuery({
    queryKey:["pay-pending-orders",notification.id,q],
    queryFn:()=>searchPayOrders(q),
    enabled:associating&&methodReady&&(view==="orders"||view==="suggestion"&&!open),
  });
  const amount=Number(notification.amount);
  const suggestions=(pending.data?.items??[])
    .filter(item=>item.paymentStatus!=="paid")
    .sort((a,b)=>Math.abs(Number(a.total??0)-amount)-Math.abs(Number(b.total??0)-amount))
    .slice(0,8);

  const detail=useQuery({
    queryKey:["pay-order-detail",selected?.id],
    queryFn:()=>payOrderDetail(selected!.id),
    enabled:associating&&Boolean(selected),
  });
  const remaining=detail.data?Number(detail.data.remainingAmount):null;
  const over=remaining!==null&&amount>remaining;
  const payable=associating&&Boolean(selected)&&!detail.isFetching&&!detail.isError&&remaining!==null&&remaining>0
    &&!methods.isFetching&&methodReady;
  const choose=(order:PayOrderRef)=>{setSelected(order);setView("detail")};
  const back=()=>{setSelected(null);setView("orders")};
  const associate=useMutation({
    mutationFn:()=>{
      if(!payable||!selected)throw new Error("Revisa el pedido antes de confirmar.");
      return associatePayNotification(notification.id,selected.id,effectiveMethod,over&&remaining!==null?remaining:undefined);
    },
    onSuccess:(result:PayNotification)=>{
      notify({tone:"success",title:"Pago asociado",message:`${payProviderLabel[result.provider]} · ${money(result.amount)}. El cobro quedó registrado en el pedido.`});
      for(const key of ["pay-notifications","pay-pending-orders","pay-notice-feed"])void qc.invalidateQueries({queryKey:[key]});
      onDone();
    },
    onError:(error:Error)=>notify({tone:"danger",title:"No se pudo asociar",message:error.message}),
  });
  const meta=payStatusMeta[notification.status];
  const exact=(total?:string|null)=>total!==undefined&&total!==null&&Number(total)===amount;

  return <Modal title={associating?"Asociar pago a pedido":"Detalle del pago"} className="pay-detail-modal" busy={associate.isPending} onClose={onClose}>
    <ModalBody className="pay-detail-body">
      <section className="pay-detail-summary" aria-label="Resumen del pago">
        <PayProviderTag provider={notification.provider}/>
        <dl className="pay-detail-amount"><dt>Monto recibido</dt><dd>{money(notification.amount)}</dd></dl>
        <Status tone={meta.tone} icon={notification.status==="associated"?"circleCheck":undefined}>{meta.label}</Status>
      </section>
      <DetailFacts label="Datos del pago" items={associating?[
        {label:"Remitente",value:notification.senderName||"Sin remitente"},
        {label:"Teléfono de origen",value:notification.deviceName||"No informado"},
      ]:[
        {label:"Remitente",value:notification.senderName||"Sin remitente"},
        {label:"Número de operación",value:notification.operationNumber||"No informado"},
        {label:"Teléfono de origen",value:notification.deviceName||"No informado"},
        {label:"Fecha y hora",value:<time dateTime={notification.occurredAt}>{dateTime(notification.occurredAt)}</time>},
      ]}/>

      {associating?<section aria-label="Asociar pago a pedido" className="pay-associate">
        <div className="pay-associate-heading">
          <h3 ref={associationHeading} tabIndex={-1}>{view==="detail"?"Revisa el pedido":"Elige el pedido"}</h3>
          <p className="form-help">{view==="detail"?"La asociación registra el cobro real en este pedido."
            :view==="suggestion"&&open?"Estás cobrando este pedido; la asociación nunca es automática."
            :"Los pedidos con el mismo monto aparecen primero."}</p>
        </div>

        {methods.isLoading?<OrderLoading label="Preparando la asociación"/>
        :methods.isError?<ErrorState className="pay-detail-state" title="No pudimos cargar los medios de pago" text={methods.error.message} onRetry={()=>void methods.refetch()}/>
        :!detectedMethod?<EmptyState className="pay-detail-state" icon="wallet" title={`${payProviderLabel[notification.provider]} no está disponible para ventas`} text={`Activa ${payProviderLabel[notification.provider]} en Configuración para asociar este pago.`}/>
        :<>
          {view==="suggestion"&&open&&<>
            <ul className="pay-order-options">
              <li><OrderOption code={open.code} meta={`Saldo pendiente: ${money(open.pending)}`} exact={Number(open.pending)===amount} onClick={()=>choose({id:open.id,code:open.code})}/></li>
            </ul>
            <Button kind="secondary" icon="search" onClick={()=>setView("orders")}>Buscar otro pedido</Button>
          </>}

          {view!=="detail"&&(view==="orders"||!open)&&<>
            <SearchField value={q} onChange={event=>setQ(event.target.value)} placeholder="Buscar por pedido, mesa o cliente…" label="Buscar pedidos pendientes de cobro"/>
            {pending.isLoading?<OrderLoading label="Buscando pedidos pendientes"/>
            :pending.isError?<ErrorState className="pay-detail-state" title="No pudimos cargar los pedidos" text={pending.error.message} onRetry={()=>void pending.refetch()}/>
            :suggestions.length?<ul className="pay-order-options">
              {suggestions.map(item=><li key={item.id}>
                <OrderOption code={item.code} meta={[channelLabel(item.channel??""),item.customerName].filter(Boolean).join(" · ")} total={item.total?money(item.total):undefined} exact={exact(item.total)} onClick={()=>choose(item)}/>
              </li>)}
            </ul>:<EmptyState className="pay-detail-state" icon="receipt" title={q.trim()?"No encontramos pedidos":"Sin pedidos pendientes de cobro"} text={q.trim()?"Prueba con otro pedido, mesa o cliente.":"El pago seguirá disponible para asociarlo cuando haya un pedido por cobrar."}/>}
          </>}

          {view==="detail"&&selected&&<div className="pay-detail-selection">
            {detail.isLoading?<OrderLoading label="Cargando el detalle del pedido"/>
            :detail.isError?<ErrorState className="pay-detail-state" title="No pudimos cargar el pedido" text={detail.error.message} onRetry={()=>void detail.refetch()}/>
            :detail.data&&<>
              <DetailFacts label="Pedido seleccionado" items={[
                {label:"Pedido",value:<><strong>{detail.data.order.code}</strong> · {channelLabel(detail.data.order.channel)}</>},
                ...(detail.data.order.customerName?[{label:"Cliente",value:detail.data.order.customerName}]:[]),
                {label:"Total del pedido",value:money(detail.data.order.total)},
                {label:"Pagos registrados",value:money(detail.data.paidAmount)},
              ]}/>
              <div className="pay-balance" role="group" aria-label="Saldo del pedido">
                <span>Saldo pendiente</span><strong>{money(detail.data.remainingAmount)}</strong>
              </div>
              {remaining===0&&<p className="form-help" role="status">Este pedido ya está pagado. Elige otro pedido para asociar el pago.</p>}
              {over&&remaining!==0&&<p className="form-help" role="status">El pago supera el saldo pendiente en {money(amount-remaining!)}. Se registrará solo el saldo y la diferencia quedará sin aplicar.</p>}
            </>}
          </div>}
        </>}
      </section>:notification.status!=="detected"&&<p className="pay-detail-footer-note">{notification.status==="dismissed"?"Este pago fue descartado y se conserva en el historial.":"Este pago ya fue asociado a un pedido."}</p>}
    </ModalBody>
    {associating&&view==="detail"&&<ModalFooter>
      <div className="modal-action-layout">
        <Button action="back" onClick={back} disabled={associate.isPending}>Volver a pedidos</Button>
        <Button disabled={!payable} loading={associate.isPending} onClick={()=>associate.mutate()}>Asociar pago</Button>
      </div>
    </ModalFooter>}
  </Modal>;
}

/** Fila completa pulsable: el pedido se revisa antes de confirmar, sin un
 * botón «Revisar» repetido en cada fila. */
function OrderOption({code,meta,total,exact,onClick}:{code:string;meta?:string;total?:string;exact:boolean;onClick:()=>void}){
  return <button type="button" className="pay-order-option" aria-label={`Revisar el pedido ${code}`} onClick={onClick}>
    <span className="pay-order-main">
      <span className="pay-order-code"><strong>{code}</strong>{exact&&<Tag icon="check">Monto exacto</Tag>}</span>
      {meta&&<small>{meta}</small>}
    </span>
    <span className="pay-order-side">
      {total&&<strong className="pay-order-total">{total}</strong>}
      <Icon name="chevron" size={16} className="pay-order-chevron"/>
    </span>
  </button>;
}

function OrderLoading({label}:{label:string}){
  return <div className="pay-order-loading" role="status" aria-label={label} aria-busy="true">
    <Skeleton height="var(--size-56)"/><Skeleton height="var(--size-56)"/><Skeleton height="var(--size-56)"/>
  </div>;
}
