"use client";
import {Button,EmptyState,ErrorState,Icon,Status,ModalHeader,ModalFooter,ModalWorkspace,ModalItemList,ModalItemNote,ModalItemComponents} from "@/design-system";
import {Dialog} from "@/design-system/dialog";
import type {SaleDetail} from "../domain/types";
import "./sales.css";

type Props={loading:boolean;error?:string;data?:SaleDetail;money:(value:string|number)=>string;date:(value:string)=>string;close:()=>void;retry:()=>void;ticket?:()=>void};
const channels:Record<string,string>={salon:"Salón",mostrador:"Mostrador",recojo:"Recojo",delivery:"Delivery",whatsapp:"WhatsApp"};
const paymentStates={paid:{label:"Pagada",tone:"green" as const},partial:{label:"Pago parcial",tone:"orange" as const},pending:{label:"Sin pagos",tone:"gray" as const}};

export function SaleDetailDialog({loading,error,data,money,date,close,retry,ticket}:Props){
  const status=data?paymentStates[data.paymentStatus]:null;
  return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal sales-detail-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="sales-detail-title" aria-busy={loading}>

    <ModalHeader title="Detalle de venta" titleId="sales-detail-title" icon="receipt" onClose={close} closeLabel="Cerrar" details={data&&!loading&&!error?<small>{data.order.code}</small>:undefined}/>
    {loading?<SaleDetailSkeleton/>
    :error?<div className="modal-body"><ErrorState title="No pudimos cargar la venta" text={error} onRetry={retry}/></div>
    :!data?<div className="modal-body"><EmptyState icon="receipt" title="No hay detalle disponible"/></div>
    :<ModalWorkspace summary={<section className="modal-summary" aria-labelledby="sales-detail-summary-title">
      <div className="modal-summary-heading"><h3 id="sales-detail-summary-title">Resumen de venta</h3>{status&&<Status tone={status.tone} emphasis>{status.label}</Status>}</div>
      <div className="modal-summary-lines">
        <div className="modal-summary-line"><span>Subtotal</span><b>{money(data.order.subtotal)}</b></div>
        <div className="modal-summary-line"><span>Pagado neto</span><b>{money(data.paidAmount)}</b></div>
        {Number(data.order.deliveryFee)>0&&<div className="modal-summary-line"><span>Delivery</span><b>{money(data.order.deliveryFee)}</b></div>}
        {Number(data.remainingAmount)>0&&<div className="modal-summary-line"><span>Saldo pendiente</span><b>{money(data.remainingAmount)}</b></div>}
      </div>
      <div className="modal-summary-total"><span>Total</span><strong>{money(data.order.total)}</strong></div>
    </section>}>
    <div className="modal-body sales-detail-body hover-scroll" tabIndex={0} aria-label="Datos, productos y pagos de la venta">
      <section className="sales-detail-context" aria-label="Datos de la venta"><dl className="modal-record-data">
        <div><dt>{data.order.tableName?"Mesa":data.order.customerName?"Cliente":"Venta"}</dt><dd>{data.order.tableName||data.order.customerName||"Venta directa"}{data.order.tableName&&data.order.customerName&&<small>{data.order.customerName}</small>}</dd></div>
        <div><dt>Canal</dt><dd>{channels[data.order.channel]??data.order.channel}</dd></div>
        {data.order.channel==="salon"&&<div><dt>Atendido por</dt><dd>{data.order.waiterName||"Sin asignar"}</dd></div>}
        <div><dt>Fecha del pedido</dt><dd>{date(data.order.createdAt)}</dd></div>
      </dl></section>
      <section className="sales-detail-section" aria-labelledby="sales-detail-products"><h3 id="sales-detail-products">Productos del pedido</h3>
        {data.order.items?.length?<ModalItemList className="sales-detail-lines">{data.order.items.map(item=><div className="modal-item-row" role="listitem" key={item.id}>
          <b className="modal-item-quantity">{Number(item.qty)}×</b><div className="modal-item-info"><span className="modal-item-name">{item.name}</span>
          {item.note&&<ModalItemNote>{item.note}</ModalItemNote>}
          <span className="modal-item-meta"><span>{money(item.unitPrice)} c/u</span></span>
          <ModalItemComponents label={`Detalle de ${item.name}`} items={[
            ...(item.selections??[]).map((selection,index)=>({key:`selection-${index}`,group:selection.groupName,name:selection.name,extra:Number(selection.surcharge)>0&&<small>+{money(selection.surcharge??0)}</small>})),
            ...(item.modifiers??[]).map((modifier,index)=>({key:`modifier-${index}`,group:modifier.groupName,name:modifier.name})),
          ]}/></div><strong className="modal-item-total">{money(Number(item.qty)*Number(item.unitPrice))}</strong>
        </div>)}</ModalItemList>:<p className="sales-detail-empty">No hay productos registrados.</p>}
      </section>
      <section className="sales-detail-section" aria-labelledby="sales-detail-payments"><h3 id="sales-detail-payments">Pagos registrados</h3>
        {data.payments.length?<ModalItemList className="sales-detail-payments">{data.payments.map(payment=><div className="modal-item-row" role="listitem" key={payment.id}>
          <span className="sales-detail-payment-icon"><Icon name="payment" size={18}/></span><div><b>{payment.methodName||payment.method}</b><small>{date(payment.createdAt)}</small>
          {payment.cashRegisterName&&<small>{payment.cashRegisterName}</small>}<small>Cobrado por: {payment.createdByName||"Sin información"}</small>{payment.reference&&<p>{payment.reference}</p>}</div>
          <div className="sales-detail-payment-value"><strong>{money(payment.netAmount)}</strong>{Number(payment.refundedAmount)>0&&<small>Cobrado {money(payment.amount)}<br/>Devuelto {money(payment.refundedAmount)}</small>}</div>
        </div>)}</ModalItemList>:<p className="sales-detail-empty">No hay pagos registrados.</p>}
      </section>
      {data.order.notes&&<section className="modal-detail-notes sales-detail-notes"><Icon name="edit" size={16}/><div><h3>Notas generales</h3><p>{data.order.notes}</p></div></section>}
    </div></ModalWorkspace>}
    {!loading&&!error&&data&&Number(data.paidAmount)>0&&ticket&&<ModalFooter><Button kind="secondary" icon="eye" onClick={ticket}>Ver resumen de pago</Button></ModalFooter>}
  </Dialog></div>;
}

function SaleDetailSkeleton(){return <ModalWorkspace summary={<section className="modal-summary sales-detail-loading" aria-hidden="true"><div className="modal-summary-heading"><b/></div><div className="modal-summary-lines">{Array.from({length:2},(_,index)=><div className="modal-summary-line" key={index}><i/><b/></div>)}</div><div className="modal-summary-total"><i/><b/></div></section>}>
<div className="modal-body sales-detail-body sales-detail-loading" aria-label="Cargando detalle de venta" role="status">
  <div className="sales-detail-context" aria-hidden="true"><dl className="modal-record-data">{Array.from({length:4},(_,index)=><div key={index}><i/><b/></div>)}</dl></div>
  {Array.from({length:2},(_,section)=><section className="sales-detail-section" aria-hidden="true" key={section}><h3><i/></h3><div className="modal-item-list sales-detail-lines">{Array.from({length:3},(_,index)=><div className="modal-item-row" key={index}><i className="modal-item-quantity"/><div><b/><small/></div><strong/></div>)}</div></section>)}
</div></ModalWorkspace>}
