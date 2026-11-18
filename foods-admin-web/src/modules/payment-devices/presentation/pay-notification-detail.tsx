"use client";

import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,EmptyState,ErrorState,FormDisclosure,Modal,ModalBody,ModalFooter,SearchField,Select,Skeleton,Status} from "@/design-system";
import {DetailFacts,DetailHero} from "@/design-system/detail-view";
import {useFeedback,useMoney,useSession} from "@/providers";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {payProviderLabel,payStatusMeta} from "../domain/pay-status";
import type {PayNotification,PayOrderRef} from "../domain/types";
import {PayProviderTag} from "./pay-status-tag";
import {associatePayNotification,payOrderDetail,paySalesMethods,searchPayOrders} from "../infrastructure/pay-devices-api";
import {usePayCollection} from "./pay-collection-context";

type Props={notification:PayNotification|null;onClose:()=>void;onDone:()=>void};

const channelLabel=(channel:string)=>({salon:"Mesa",delivery:"Delivery",mostrador:"Venta rápida",recojo:"Recojo",whatsapp:"WhatsApp"}[channel]??channel);

/** Revisar el pedido y su saldo siempre precede al registro del cobro. */
export function PayNotificationDetail({notification,onClose,onDone}:Props){
  if(!notification)return null;
  return <DetailBody key={notification.id} notification={notification} onClose={onClose} onDone={onDone}/>;
}

function DetailBody({notification,onClose,onDone}:{notification:PayNotification;onClose:()=>void;onDone:()=>void}){
  const qc=useQueryClient();
  const {notify}=useFeedback();
  const money=useMoney();
  const {location}=useSession();
  const {open}=usePayCollection();
  const [view,setView]=useState<"suggestion"|"orders"|"detail">(open?"suggestion":"orders");
  const [q,setQ]=useState("");
  const [selected,setSelected]=useState<PayOrderRef|null>(null);
  const [sourceOpen,setSourceOpen]=useState(false);
  const canAssociate=notification.status==="detected";
  const dateTime=(value:string)=>formatRegionalDateTime(value,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium",timeStyle:"short"});

  const methods=useQuery({queryKey:["pay-sales-methods"],queryFn:paySalesMethods,enabled:canAssociate});
  // El código estable del catálogo identifica al proveedor, aunque se renombre.
  const detectedMethod=methods.data?.items.find(item=>item.code===notification.provider);
  const effectiveMethod=detectedMethod?.code??"";
  const pending=useQuery({
    queryKey:["pay-pending-orders",notification.id,q],
    queryFn:()=>searchPayOrders(q),
    enabled:canAssociate&&(view==="orders"||view==="suggestion"&&!open),
  });
  const suggestions=(pending.data?.items??[])
    .filter(item=>item.paymentStatus!=="paid")
    .sort((a,b)=>Math.abs(Number(a.total??0)-Number(notification.amount))-Math.abs(Number(b.total??0)-Number(notification.amount)))
    .slice(0,8);

  const detail=useQuery({
    queryKey:["pay-order-detail",selected?.id],
    queryFn:()=>payOrderDetail(selected!.id),
    enabled:canAssociate&&Boolean(selected),
  });
  const remaining=detail.data?Number(detail.data.remainingAmount):null;
  const amount=Number(notification.amount);
  const over=remaining!==null&&amount>remaining;
  const payable=canAssociate&&Boolean(selected)&&!detail.isFetching&&!detail.isError&&remaining!==null&&remaining>0
    &&!methods.isFetching&&!methods.isError&&Boolean(methods.data?.items.some(item=>item.code===effectiveMethod));
  const choose=(order:PayOrderRef)=>{setSelected(order);setView("detail")};
  const back=()=>{setSelected(null);setView("orders")};
  const associate=useMutation({
    mutationFn:()=>{
      if(!payable||!selected)throw new Error("Revisa el pedido y el medio de pago antes de confirmar.");
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

  return <Modal title="Pago detectado" className="pay-detail-modal" busy={associate.isPending} onClose={onClose}>
    <ModalBody className="pay-detail-body">
      <DetailHero title={<PayProviderTag provider={notification.provider}/>} status={<Status tone={meta.tone}>{meta.label}</Status>}/>
      <dl className="pay-detail-amount"><dt>Monto recibido</dt><dd>{money(notification.amount)}</dd></dl>
      <DetailFacts label="Datos del pago" items={[
        {label:"Remitente",value:notification.senderName||"Sin remitente",wide:true},
        {label:"Número de operación",value:notification.operationNumber||"No informado",wide:true},
        {label:"Fecha y hora",value:<time dateTime={notification.occurredAt}>{dateTime(notification.occurredAt)}</time>},
      ]}/>

      {canAssociate?<section aria-label="Asociar pago a pedido" className="pay-associate">
        <div className="pay-associate-heading">
          <h3>Asociar pago a pedido</h3>
          <p className="form-help">La asociación registra el cobro real. Revisa el pedido antes de confirmar.</p>
        </div>
        {methods.isLoading?<div role="status" aria-label="Cargando medios de pago"><Skeleton height="var(--control-height)"/></div>
        :methods.isError?<ErrorState className="pay-detail-state" title="No pudimos cargar los medios de pago" text={methods.error.message} onRetry={()=>void methods.refetch()}/>
        :!detectedMethod?<EmptyState className="pay-detail-state" icon="wallet" title={`${payProviderLabel[notification.provider]} no está disponible para ventas`} text={`Activa ${payProviderLabel[notification.provider]} en Configuración para asociar este pago.`}/>
        :<label className="pay-detail-field">Medio de pago
          <Select value={effectiveMethod} disabled>
            <option value={detectedMethod.code}>{detectedMethod.name}</option>
          </Select>
        </label>}

        {view==="suggestion"&&open&&<>
          <p className="form-help">Estás cobrando este pedido; la asociación nunca es automática.</p>
          <ul className="pay-order-options"><li className="pay-order-option">
            <span><strong>{open.code}</strong><small>Saldo pendiente: {money(open.pending)}</small></span>
            <Button kind="secondary" onClick={()=>choose({id:open.id,code:open.code})}>Revisar pedido</Button>
          </li></ul>
          <Button kind="secondary" onClick={()=>setView("orders")}>Cambiar pedido</Button>
        </>}

        {view!=="detail"&&(view==="orders"||!open)&&<>
          <SearchField value={q} onChange={event=>setQ(event.target.value)} placeholder="Buscar por pedido, mesa o cliente…" label="Buscar pedidos pendientes de cobro"/>
          {pending.isLoading?<OrderLoading label="Buscando pedidos pendientes"/>
          :pending.isError?<ErrorState className="pay-detail-state" title="No pudimos cargar los pedidos" text={pending.error.message} onRetry={()=>void pending.refetch()}/>
          :suggestions.length?<ul className="pay-order-options">
            {suggestions.map(item=><li key={item.id} className="pay-order-option">
              <span><strong>{item.code}</strong><small>{[channelLabel(item.channel??""),item.customerName,item.total?`Total: ${money(item.total)}`:null].filter(Boolean).join(" · ")}</small></span>
              <Button kind="secondary" onClick={()=>choose(item)}>Revisar</Button>
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
              {label:"Saldo pendiente",value:<strong>{money(detail.data.remainingAmount)}</strong>},
            ]}/>
            {remaining===0&&<p className="form-help" role="status">Este pedido ya está pagado. Elige otro pedido para asociar el pago.</p>}
            {over&&remaining!==0&&<p className="form-help" role="status">El pago supera el saldo pendiente en {money(amount-remaining!)}. Se registrará solo el saldo y la diferencia quedará sin aplicar.</p>}
          </>}
        </div>}
      </section>:<p className="pay-detail-footer-note">{notification.status==="dismissed"?"Este pago fue descartado y se conserva en el historial.":"Este pago ya fue asociado a un pedido."}</p>}
      <FormDisclosure title="Origen del aviso" summary="" open={sourceOpen} onToggle={()=>setSourceOpen(value=>!value)}>
        <DetailFacts label="Origen del aviso" items={[{label:"Teléfono de origen",value:notification.deviceName}]}/>
      </FormDisclosure>
    </ModalBody>
    {canAssociate&&view==="detail"&&<ModalFooter>
      <div className="modal-action-layout">
        <Button action="back" onClick={back} disabled={associate.isPending}>Volver a pedidos</Button>
        <Button disabled={!payable} loading={associate.isPending} onClick={()=>associate.mutate()}>Confirmar asociación</Button>
      </div>
    </ModalFooter>}
  </Modal>;
}

function OrderLoading({label}:{label:string}){
  return <div className="pay-order-loading" role="status" aria-label={label} aria-busy="true">
    <Skeleton height="var(--size-20)" width="70%"/><Skeleton height="var(--size-12)" width="90%"/>
    <Skeleton height="var(--size-20)" width="60%"/><Skeleton height="var(--size-12)" width="80%"/>
  </div>;
}
