"use client";
import {useState} from "react";
import {Button,DetailContact,DetailTimeline,EmptyState,ErrorState,Icon,Skeleton,Status,Tabs,ModalHeader,ModalFooter,ModalWorkspace,ModalItemList,ModalItemNote,ModalItemComponents,type IconName} from "@/design-system";
import {Dialog} from "@/design-system/dialog";
import type {SaleDetail} from "../domain/types";
import "./sales.css";

type Props={loading:boolean;error?:string;data?:SaleDetail;money:(value:string|number)=>string;date:(value:string)=>string;close:()=>void;retry:()=>void;ticket?:()=>void};
const channels:Record<string,string>={salon:"Salón",mostrador:"Mostrador",recojo:"Recojo",delivery:"Delivery",whatsapp:"WhatsApp"};
const channelIcons:Record<string,IconName>={salon:"utensils",mostrador:"store",recojo:"box",delivery:"truck",whatsapp:"share"};
const paymentStates={paid:{label:"Pagada",tone:"green" as const},partial:{label:"Pago parcial",tone:"orange" as const},pending:{label:"Sin pagos",tone:"gray" as const}};
const titleId="sales-detail-title";

/**
 * Detalle de venta con el patrón de Detalle de pedido: cabecera con la mesa o el cliente,
 * identidad (estado y fecha), tarjeta de cliente, pestañas y resumen fijo al pie.
 */
export function SaleDetailDialog({loading,error,data,money,date,close,retry,ticket}:Props){
  const status=data?paymentStates[data.paymentStatus]:null;
  const tableName=data?.order.tableName?.trim();
  const title=tableName?(/^mesa\b/i.test(tableName)?tableName:`Mesa ${tableName}`):data?.order.customerName||(data?"Venta directa":"Detalle de venta");
  const icon:IconName=tableName?"utensils":data?channelIcons[data.order.channel]??"receipt":"receipt";
  const paid=Number(data?.paidAmount??0);
  const remaining=Number(data?.remainingAmount??0);
  const[tab,setTab]=useState<"products"|"payments">("products");
  return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal sales-detail-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby={titleId} aria-busy={loading}>
    {loading?<SaleDetailSkeleton close={close} withFooter={Boolean(ticket)}/>:<>
    <ModalHeader title={title} titleId={titleId} icon={icon} onClose={close} closeLabel="Cerrar detalle"/>
    {error?<div className="modal-body sales-detail-body"><ErrorState title="No pudimos cargar la venta" text={error} onRetry={retry}/></div>
    :!data?<div className="modal-body sales-detail-body"><EmptyState icon="receipt" title="No hay detalle disponible"/></div>
    :<ModalWorkspace summary={<section className={`modal-summary sales-detail-totals${remaining<=0.00001?" is-paid":""}`} aria-labelledby="sales-detail-summary-title">
      <div className="modal-summary-heading"><h3 id="sales-detail-summary-title" className="modal-section-title">Resumen de venta</h3></div>
      <div className="modal-summary-lines">
        <div className="modal-summary-line"><span>Subtotal</span><b>{money(data.order.subtotal)}</b></div>
        {Number(data.order.deliveryFee)>0&&<div className="modal-summary-line"><span>Delivery</span><b>{money(data.order.deliveryFee)}</b></div>}
        <div className={`modal-summary-line sales-detail-paid${paid>0.00001?" has-payment":""}`}><span>Pagado neto</span><b>{money(paid)}</b></div>
      </div>
      <div className="modal-summary-total"><span>{remaining>0.00001?"Saldo pendiente":"Total pagado"}</span><strong>{money(remaining>0.00001?remaining:data.order.total)}</strong></div>
    </section>}>
    <div className="modal-body sales-detail-body" tabIndex={0} aria-label="Productos, pagos y datos de la venta">
      <div className="sales-detail-content">
        <section className="sales-detail-hero" aria-label="Resumen de la venta">
          <div className="sales-detail-identity">
            {status&&<div className="sales-detail-status" aria-label="Estado de pago"><Status tone={status.tone}>{status.label}</Status></div>}
            <p><Icon name="clock" size={14}/><time dateTime={data.order.createdAt}>{date(data.order.createdAt)}</time></p>
            <p><Icon name={channelIcons[data.order.channel]??"receipt"} size={14}/><span>{[channels[data.order.channel]??data.order.channel,data.order.channel==="salon"&&`Atendido por ${data.order.waiterName||"sin asignar"}`].filter(Boolean).join(" · ")}</span></p>
          </div>
          {data.order.customerName&&<DetailContact name={data.order.customerName}/>}
        </section>

        <Tabs className="sales-detail-tabs" label="Información de la venta" value={tab} onChange={setTab} items={[
          {value:"products",label:"Productos",icon:"dish",count:data.order.items?.length??0},
          {value:"payments",label:"Pagos",icon:"payment",count:data.payments.length},
        ]}/>

        <section className="detail-panel sales-detail-panel" role="tabpanel" aria-labelledby="sales-detail-products" hidden={tab!=="products"}><h3 id="sales-detail-products" className="sr-only">Productos del pedido</h3>
          {data.order.items?.length?<ModalItemList className="sales-detail-lines">{data.order.items.map(item=><div className="modal-item-row" role="listitem" key={item.id}>
            <b className="modal-item-quantity">{Number(item.qty)}×</b><div className="modal-item-info"><span className="modal-item-name">{item.name}</span>
            {item.note&&<ModalItemNote>{item.note}</ModalItemNote>}
            <span className="modal-item-meta"><span>{money(item.unitPrice)} c/u</span></span>
            <ModalItemComponents label={`Detalle de ${item.name}`} items={[
              ...(item.selections??[]).map((selection,index)=>({key:`selection-${index}`,group:selection.groupName,name:selection.name,extra:Number(selection.surcharge)>0&&<small>+{money(selection.surcharge??0)}</small>})),
              ...(item.modifiers??[]).map((modifier,index)=>({key:`modifier-${index}`,group:modifier.groupName,name:modifier.name})),
            ]}/></div><strong className="modal-item-total">{money(Number(item.qty)*Number(item.unitPrice))}</strong>
          </div>)}</ModalItemList>:<p className="sales-detail-empty">No hay productos registrados.</p>}
          {data.order.notes&&<section className="modal-detail-notes sales-detail-notes" aria-labelledby="sales-detail-notes-title"><span><Icon name="edit" size={16}/></span><div><h3 id="sales-detail-notes-title">Notas generales</h3><p>{data.order.notes}</p></div></section>}
        </section>

        <section className="detail-panel sales-detail-panel" role="tabpanel" aria-labelledby="sales-detail-payments" hidden={tab!=="payments"}><h3 id="sales-detail-payments" className="sr-only">Pagos registrados</h3>
          {data.payments.length?<DetailTimeline label="Pagos de la venta" steps={data.payments.map(payment=>({
            key:payment.id,
            icon:"payment",
            title:payment.methodName||payment.method,
            time:date(payment.createdAt),
            description:[payment.cashRegisterName,`Cobrado por: ${payment.createdByName||"Sin información"}`,payment.reference].filter(Boolean).join(" · "),
            aside:<>{money(payment.netAmount)}{Number(payment.refundedAmount)>0&&<small>Cobrado {money(payment.amount)} · Devuelto {money(payment.refundedAmount)}</small>}</>,
          }))}/>:<p className="sales-detail-empty">No hay pagos registrados.</p>}
        </section>      </div>
    </div></ModalWorkspace>}
    {!error&&data&&paid>0&&ticket&&<ModalFooter><Button kind="secondary" icon="eye" onClick={ticket}>Ver resumen de pago</Button></ModalFooter>}
    </>}
  </Dialog></div>;
}

/** Skeleton con la geometría final: cabecera, identidad, tarjeta, pestañas, productos, resumen y acción. */
function SaleDetailSkeleton({close,withFooter}:{close:()=>void;withFooter:boolean}){
  return <>
    <ModalHeader title={<><span className="sr-only">Cargando detalle de venta</span><Skeleton width="var(--size-138)" height="var(--size-16)"/></>} titleId={titleId}
      iconSlot={<Skeleton width="var(--size-44)" height="var(--size-44)" radius="var(--radius-tile)" className="sales-detail-skeleton-icon"/>} onClose={close} closeLabel="Cerrar detalle"/>
    <ModalWorkspace summary={
      <section className="modal-summary" aria-hidden="true">
        <div className="modal-summary-heading"><Skeleton width="var(--size-138)" height="var(--size-13)"/></div>
        <div className="modal-summary-lines">
          {Array.from({length:2},(_,index)=><div className="modal-summary-line" key={index}><Skeleton width="var(--size-72)" height="var(--size-9)"/><Skeleton width="var(--size-68)" height="var(--size-16)"/></div>)}
        </div>
        <div className="modal-summary-total"><Skeleton width="var(--size-116)" height="var(--size-10)"/><Skeleton width="var(--size-108)" height="var(--size-24)"/></div>
      </section>
    }>
    <div className="modal-body sales-detail-body" role="status" aria-label="Cargando detalle de venta">
      <div className="sales-detail-content" aria-hidden="true">
        <div className="sales-detail-hero">
          <div className="sales-detail-identity"><Skeleton width="var(--size-72)" height="var(--size-24)" radius="var(--radius-pill)"/><Skeleton width="var(--size-150)" height="var(--size-9)"/></div>
          <Skeleton width="100%" height="var(--size-68)" radius="var(--radius-card)"/>
        </div>
        <Skeleton className="sales-detail-tabs" width="min(var(--size-320),100%)" height="var(--size-44)" radius="var(--radius-pill)"/>
        <ModalItemList className="sales-detail-panel sales-detail-lines">
          {Array.from({length:3},(_,index)=><div className="modal-item-row" role="listitem" key={index}>
            <Skeleton width="var(--size-32)" height="var(--size-32)" radius="var(--radius-pill)"/>
            <span className="sales-detail-skeleton-copy"><Skeleton width="min(var(--size-220),72%)" height="var(--size-11)"/><Skeleton width="var(--size-92)" height="var(--size-9)"/></span>
            <Skeleton width="var(--size-66)" height="var(--size-10)"/>
          </div>)}
        </ModalItemList>
      </div>
    </div>
    </ModalWorkspace>
    {withFooter&&<ModalFooter className="sales-detail-skeleton-actions"><Skeleton className="sales-detail-skeleton-action"/></ModalFooter>}
  </>;
}
