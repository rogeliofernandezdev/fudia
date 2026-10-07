"use client";
import {Button,Icon,Status} from "@/design-system";
import {Dialog} from "@/design-system/dialog";
import type {SaleDetail} from "../domain/types";
import "./sales.css";

type Props={loading:boolean;error?:string;data?:SaleDetail;money:(value:string|number)=>string;date:(value:string)=>string;close:()=>void;retry:()=>void};
const channels:Record<string,string>={salon:"Salón",mostrador:"Mostrador",recojo:"Recojo",delivery:"Delivery",whatsapp:"WhatsApp"};
const paymentStates={paid:{label:"Pagada",tone:"green" as const},partial:{label:"Pago parcial",tone:"orange" as const},pending:{label:"Sin pagos",tone:"gray" as const}};

export function SaleDetailDialog({loading,error,data,money,date,close,retry}:Props){
  const status=data?paymentStates[data.paymentStatus]:null;
  return <div className="modal-backdrop modal-overlay-in"><Dialog className="crud-modal sales-detail-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="sales-detail-title" aria-busy={loading}>
    <div className="modal-accent"/>
    <header><span className="modal-title-icon"><Icon name="receipt" size={18}/></span><div><h2 id="sales-detail-title">Detalle de venta</h2>{data&&!loading&&!error&&<small>{data.order.code}</small>}</div><button type="button" aria-label="Cerrar" onClick={close}><Icon name="close"/></button></header>
    {loading?<SaleDetailSkeleton/>
    :error?<div className="sales-detail-state" role="alert"><Icon name="alert" size={24}/><b>No pudimos cargar la venta</b><p>{error}</p><Button kind="secondary" icon="refresh" onClick={retry}>Reintentar</Button></div>
    :!data?<div className="sales-detail-state"><Icon name="receipt" size={24}/><b>No hay detalle disponible</b></div>
    :<div className="sales-detail-body hover-scroll">
      <section className="sales-detail-context" aria-label="Datos de la venta"><dl>
        <div><dt>Cliente / mesa</dt><dd>{data.order.tableName||data.order.customerName||"Venta directa"}{data.order.tableName&&data.order.customerName&&<small>{data.order.customerName}</small>}</dd></div>
        <div><dt>Canal</dt><dd>{channels[data.order.channel]??data.order.channel}</dd></div>
        {data.order.channel==="salon"&&<div><dt>Atendido por</dt><dd>{data.order.waiterName||"Sin asignar"}</dd></div>}
        <div><dt>Fecha del pedido</dt><dd>{date(data.order.createdAt)}</dd></div>
      </dl>{status&&<Status tone={status.tone}>{status.label}</Status>}</section>
      <section className="sales-detail-section" aria-labelledby="sales-detail-products"><h3 id="sales-detail-products">Productos</h3>
        {data.order.items?.length?<div className="sales-detail-lines">{data.order.items.map(item=><article key={item.id}>
          <span className="sales-detail-qty">{Number(item.qty)}×</span><div><b>{item.name}</b><small>{money(item.unitPrice)} c/u</small>
          {item.selections?.map((selection,index)=><small key={`selection-${index}`}>{selection.groupName}: {selection.name}</small>)}
          {item.modifiers?.map((modifier,index)=><small key={`modifier-${index}`}>{modifier.groupName}: {modifier.name}</small>)}
          {item.note&&<p>{item.note}</p>}</div><strong>{money(Number(item.qty)*Number(item.unitPrice))}</strong>
        </article>)}</div>:<p className="sales-detail-empty">No hay productos registrados.</p>}
      </section>
      <dl className="sales-detail-totals" aria-label="Resumen de la venta">
        <div><dt>Subtotal</dt><dd>{money(data.order.subtotal)}</dd></div>
        {Number(data.order.deliveryFee)>0&&<div><dt>Delivery</dt><dd>{money(data.order.deliveryFee)}</dd></div>}
        <div className="sales-detail-total"><dt>Total</dt><dd>{money(data.order.total)}</dd></div>
        <div><dt>Pagado neto</dt><dd>{money(data.paidAmount)}</dd></div>
        {Number(data.remainingAmount)>0&&<div><dt>Saldo pendiente</dt><dd>{money(data.remainingAmount)}</dd></div>}
      </dl>
      <section className="sales-detail-section" aria-labelledby="sales-detail-payments"><h3 id="sales-detail-payments">Pagos registrados</h3>
        {data.payments.length?<div className="sales-detail-payments">{data.payments.map(payment=><article key={payment.id}>
          <span className="sales-detail-payment-icon"><Icon name="payment" size={18}/></span><div><b>{payment.methodName||payment.method}</b><small>{date(payment.createdAt)}</small>
          {payment.cashRegisterName&&<small>{payment.cashRegisterName}</small>}<small>Cobrado por: {payment.createdByName||"Sin información"}</small>{payment.reference&&<p>{payment.reference}</p>}</div>
          <div className="sales-detail-payment-value"><strong>{money(payment.netAmount)}</strong>{Number(payment.refundedAmount)>0&&<small>Cobrado {money(payment.amount)}<br/>Devuelto {money(payment.refundedAmount)}</small>}</div>
        </article>)}</div>:<p className="sales-detail-empty">No hay pagos registrados.</p>}
      </section>
      {data.order.notes&&<section className="sales-detail-section"><h3>Notas</h3><p className="sales-detail-notes">{data.order.notes}</p></section>}
    </div>}
  </Dialog></div>;
}

function SaleDetailSkeleton(){return <div className="sales-detail-body sales-detail-loading" aria-label="Cargando detalle de venta">
  <div className="sales-detail-context" aria-hidden="true"><dl>{Array.from({length:3},(_,index)=><div key={index}><i/><b/></div>)}</dl></div>
  {Array.from({length:2},(_,section)=><section className="sales-detail-section" aria-hidden="true" key={section}><h3><i/></h3><div className="sales-detail-lines">{Array.from({length:3},(_,index)=><article key={index}><i className="sales-detail-qty"/><div><b/><small/></div><strong/></article>)}</div>{section===0&&<dl className="sales-detail-totals">{Array.from({length:3},(_,index)=><div key={index}><i/><b/></div>)}</dl>}</section>)}
</div>}
