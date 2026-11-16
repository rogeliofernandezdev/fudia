"use client";

import {useEffect,useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,SearchField,Select,Status,Tag} from "@/design-system";
import {Dialog} from "@/design-system/dialog";
import {ModalHeader} from "@/design-system";
import {useFeedback} from "@/providers";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {payProviderLabel,payStatusMeta} from "../domain/pay-status";
import type {PayNotification,PayOrderRef} from "../domain/types";
import {associatePayNotification,payOrderDetail,paySalesMethods,searchPayOrders} from "../infrastructure/pay-devices-api";
import {usePayCollection} from "./pay-collection-context";

type Props={notification:PayNotification|null;onClose:()=>void;onDone:()=>void};

const channelLabel=(channel:string)=>({salon:"Mesa",delivery:"Delivery",mostrador:"Venta rápida",recojo:"Recojo",whatsapp:"WhatsApp"}[channel]??channel);

/**
 * Detalle del pago detectado (§2) y asociación (§3-§7): el pedido en cobro se
 * sugiere, nunca se aplica solo; el cajero puede cambiar de pedido, revisar el
 * detalle y confirmar. Un importe superior al saldo se registra como parcial
 * del saldo y la diferencia queda explicada.
 */
export function PayNotificationDetail({notification,onClose,onDone}:Props){
  if(!notification)return null;
  // Cada notificación monta su propio cuerpo: al cambiar, el estado se reinicia.
  return <DetailBody key={notification.id} notification={notification} onClose={onClose} onDone={onDone}/>;
}

function DetailBody({notification,onClose,onDone}:{notification:PayNotification;onClose:()=>void;onDone:()=>void}){
  const qc=useQueryClient();
  const {notify}=useFeedback();
  const {open}=usePayCollection();
  const [method,setMethod]=useState("");
  const [view,setView]=useState<"suggestion"|"orders"|"detail">("suggestion");
  const [q,setQ]=useState("");
  const [selected,setSelected]=useState<PayOrderRef|null>(null);

  const methods=useQuery({queryKey:["pay-sales-methods"],queryFn:paySalesMethods,enabled:Boolean(notification)});
  // Medio por defecto: el del catálogo que corresponda a Yape o Plin; el
  // cajero puede cambiarlo antes de asociar.
  const effectiveMethod=method||(methods.data?.items.find(item=>/yape|plin/i.test(`${item.code} ${item.name}`))?.code??"");

  // Escenario B: pedidos con saldo pendiente; los de monto exacto primero.
  const pending=useQuery({
    queryKey:["pay-pending-orders",notification?.id,q],
    queryFn:()=>searchPayOrders(q),
    enabled:Boolean(notification)&&view!=="suggestion",
  });
  const ordered=(pending.data?.items??[])
    .filter(item=>!item.paymentStatus||item.paymentStatus!=="paid")
    .sort((a,b)=>{
      const target=Number(notification?.amount??0);
      const exact=(item:PayOrderRef)=>Number(item.total)===target?0:1;
      return exact(a)-exact(b)||Math.abs(Number(a.total??0)-target)-Math.abs(Number(b.total??0)-target);
    });
  const suggestions=ordered.slice(0,8);

  const detail=useQuery({
    queryKey:["pay-order-detail",selected?.id],
    queryFn:()=>payOrderDetail(selected!.id),
    enabled:Boolean(notification&&selected),
  });

  const close=()=>{onClose();setMethod("");setSelected(null);setQ("");setView("orders")};
  const associate=useMutation({
    mutationFn:(amount?:number)=>associatePayNotification(notification!.id,selected!.id,effectiveMethod,amount),
    onSuccess:(result:PayNotification)=>{
      notify({tone:"success",title:"Pago asociado",message:`${payProviderLabel[result.provider]} · ${result.currency} ${result.amount} quedó asociado al pedido ${result.orderId}.`});
      void qc.invalidateQueries({queryKey:["pay-notifications"]});
      void qc.invalidateQueries({queryKey:["pay-pending-orders"]});
      onDone();
    },
    onError:(error:Error)=>notify({tone:"danger",title:"No se pudo asociar",message:error.message}),
  });

  const meta=payStatusMeta[notification.status];
  const payable=Boolean(effectiveMethod)&&Boolean(selected);
  const remaining=detail.data?Number(detail.data.remainingAmount):null;
  const amount=Number(notification.amount);
  const over=remaining!==null&&amount>remaining;

  return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={onClose} className="crud-modal pay-detail-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="pay-detail-title" aria-busy={associate.isPending}>
    <ModalHeader title="Pago detectado" titleId="pay-detail-title" onClose={onClose} closeLabel="Cerrar"/>
    <div className="pay-detail-body">
      <dl className="pay-detail-facts">
        <div><dt>Medio de pago</dt><dd><Tag>{payProviderLabel[notification.provider]}</Tag></dd></div>
        <div><dt>Remitente</dt><dd>{notification.senderName||"Sin remitente"}</dd></div>
        <div><dt>Monto recibido</dt><dd><strong>{notification.currency} {notification.amount}</strong></dd></div>
        <div><dt>Fecha y hora</dt><dd>{formatRegionalDateTime(notification.occurredAt)}</dd></div>
        <div><dt>Número de operación</dt><dd>{notification.eventKey}</dd></div>
        <div><dt>Estado</dt><dd><Status tone={meta.tone}>{meta.label}</Status></dd></div>
      </dl>

      <section aria-label="Asociar pago a pedido" className="pay-associate">
        <h3>Asociar pago a pedido</h3>
        <label className="pay-detail-field">Medio de pago con el que se registra el cobro
          <Select value={effectiveMethod} onChange={event=>setMethod(event.target.value)}>
            <option value="">Elegir medio…</option>
            {methods.data?.items.map(item=><option key={item.code} value={item.code}>{item.name}</option>)}
          </Select>
        </label>

        {view==="suggestion"&&open&&<>
          <p className="form-help">Estás cobrando este pedido; Fudia lo sugiere, la asociación nunca es automática.</p>
          <div className="pay-order-options">
            <li className="pay-order-option">
              <span><strong>{open.code}</strong> · saldo {open.pending}</span>
            </li>
          </div>
          <div className="pay-detail-actions">
            <Button disabled={!payable} loading={associate.isPending} onClick={()=>associate.mutate()}>Confirmar asociación</Button>
            <Button kind="secondary" onClick={()=>setView("orders")}>Cambiar pedido</Button>
          </div>
        </>}

        {view!=="detail"&&(view==="orders"||!open)&&<>
          <SearchField value={q} onChange={event=>setQ(event.target.value)} placeholder="Buscar por pedido, mesa o cliente…" label="Buscar pedidos pendientes de cobro"/>
          <p className="form-help">Pedidos con saldo pendiente (mesas, delivery y venta rápida); los de monto exacto primero.</p>
          {pending.isFetching&&!pending.data&&<p className="form-help">Buscando…</p>}
          <ul className="pay-order-options">
            {suggestions.length?suggestions.map(item=><li key={item.id} className="pay-order-option">
              <span><strong>{item.code}</strong> {channelLabel(item.channel??"")} · {item.customerName??""} {item.total?`· ${item.total}`:""}</span>
              <Button kind="secondary" onClick={()=>{setSelected(item);setView("detail")}}>Elegir</Button>
            </li>):<li className="pay-order-option"><span>{pending.isFetching?"Buscando…":"Sin pedidos pendientes de cobro."}</span></li>}
          </ul>
        </>}

        {view==="detail"&&selected&&<>
          {detail.isLoading&&<p className="form-help">Cargando el detalle del pedido…</p>}
          {detail.isError&&<p className="field-error" role="alert">{detail.error.message}</p>}
          {detail.data&&<>
            <dl className="pay-detail-facts">
              <div><dt>Pedido</dt><dd><strong>{detail.data.order.code}</strong> · {channelLabel(detail.data.order.channel)}</dd></div>
              {detail.data.order.customerName&&<div><dt>Cliente</dt><dd>{detail.data.order.customerName}</dd></div>}
              <div><dt>Total del pedido</dt><dd>{detail.data.order.total??"—"}</dd></div>
              <div><dt>Pagos registrados</dt><dd>{detail.data.paidAmount}</dd></div>
              <div><dt>Saldo pendiente</dt><dd><strong>{detail.data.remainingAmount}</strong></dd></div>
            </dl>
            {over&&<p className="field-error" role="alert">El pago supera el saldo pendiente en {notification.currency} {(amount-remaining!).toFixed(2)}. Se registrará solo el saldo y la diferencia quedará sin aplicar.</p>}
            <div className="pay-detail-actions">
              <Button kind="secondary" onClick={()=>{setSelected(null);setView("orders")}}>Volver a pedidos</Button>
              <Button disabled={!payable} loading={associate.isPending}
                onClick={()=>associate.mutate(over&&remaining!==null?remaining:undefined)}>Confirmar asociación</Button>
            </div>
          </>}
        </>}
      </section>
    </div>
  </Dialog></div>;
}
