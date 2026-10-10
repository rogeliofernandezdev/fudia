"use client";
import "./order-detail.css";
import Link from "next/link";
import {Button,ErrorState,Icon,Skeleton,Status} from "@/design-system";
import {useMoney,useSession} from "@/providers";
import {pageRoutes} from "@/shared/routing/page-routes";
import {canCancelOrder,canManageOrderService,nextOrderAction} from "../domain/order-actions";
import {canChargeAccount} from "../domain/service-flow";
import type {Order} from "../domain/types";
import {OrderAttribution,OrderAttributionSkeleton} from "./order-attribution";
import {channelIcons,isEditableOrder,openedAgo,orderStatus,registeredAt,tableTitle} from "./order-meta";
import {OrderAccountActions,OrderItemService,accountLabel} from "./order-service-controls";

/**
 * Detalle compartido de Pedidos y Salón. `table` presenta la mesa (título «Mesa 04»,
 * tiempo de apertura y edición de comanda); `order` presenta el pedido con su fecha
 * de registro. Cabecera, cuenta y acciones son idénticas en ambos contextos.
 */
export type OrderDetailVariant="table"|"order";
export const orderDetailTitleId="order-detail-title";

type ContentProps<T extends Order>={
  variant:OrderDetailVariant;
  order?:T;
  error?:string;
  retrying?:boolean;
  canManage:boolean;
  busy:boolean;
  close:()=>void;
  retry:()=>void;
  advance:(status:string)=>void;
  add:(order:T)=>void;
  cancel:(order:T)=>void;
  edit?:(order:T)=>void;
};

export function OrderDetailContent<T extends Order>({variant,order,error,retrying=false,canManage:hasPermission,busy,close,retry,advance,add,cancel,edit}:ContentProps<T>){
  const{location,user,can}=useSession();
  const money=useMoney();
  const table=variant==="table";
  const canManage=Boolean(hasPermission&&order&&canManageOrderService(order,user?.id));
  const meta=order?orderStatus(order.status):null;
  const action=order?nextOrderAction(order):null;
  const paid=Number(order?.paidAmount??0);
  const remaining=Number(order?.remainingAmount??order?.total??0);
  const hasPayments=paid>0.00001;
  const canCharge=Boolean(can("cash.manage")&&order&&canChargeAccount(order)&&remaining>0.00001);
  const canCancel=Boolean(order&&canCancelOrder(order));
  const editable=Boolean(edit&&order&&isEditableOrder(order)&&(!(order.serviceItems?.length)||order.status==="nuevo"));
  const actionIcon=action?.status==="entregado"?"availability":action?.status==="confirmado"?"chefHat":action?.icon;
  const items=order?.items??[];
  const itemCount=items.reduce((sum,item)=>sum+Number(item.qty||0),0);
  const title=table?tableTitle(order?.tableName):order?(order.tableName||order.customerName||"Pedido"):"Pedido";
  const subtitle=order?(table?order.customerName:order.tableName&&order.customerName?order.customerName:""):"";
  const opened=order?(table?openedAgo(order.createdAt,location?.country,location?.timezone):registeredAt(order.createdAt,location?.country,location?.timezone)):"";
  return <>
    <div className="modal-accent" aria-hidden="true"/>
    <header>
      <span className="modal-title-icon"><Icon name={table?"utensils":order?channelIcons[order.channel]??"receipt":"receipt"} size={18}/></span>
      <div className="order-detail-heading">
        <small>DETALLE DEL PEDIDO</small>
        <h2 id={orderDetailTitleId}>{title}</h2>
        {order&&<p>{subtitle&&<><span>{subtitle}</span><span aria-hidden="true"> · </span></>}<span className="order-detail-opened"><Icon name="clock" size={14}/><time dateTime={order.createdAt}>{opened}</time></span></p>}
        {order&&meta&&<div className="order-detail-status"><Status tone={meta.tone}>{meta.label}</Status>{order.channel==="salon"&&<Status tone="blue">{accountLabel(order)}</Status>}</div>}
      </div>
      <button type="button" aria-label="Cerrar detalle" onClick={close}><Icon name="close"/></button>
    </header>

    {error?(
      <div className="order-detail-body"><ErrorState title="No pudimos cargar el pedido" text={error} onRetry={retry} retrying={retrying}/></div>
    ):order&&meta&&<>
      <div className="order-detail-body">
        <div className="order-detail-content">
          <OrderAttribution channel={order.channel} waiterName={order.waiterName} collectedByNames={order.collectedByNames}/>

          {(order.customerPhone||order.address||order.reference)&&<section className="order-detail-context" aria-label="Contacto y entrega">
            {order.customerPhone&&<div><span><Icon name="users" size={16}/></span><p><small>CONTACTO</small><b>{order.customerPhone}</b></p></div>}
            {(order.address||order.reference)&&<div><span><Icon name="truck" size={16}/></span><p><small>ENTREGA</small><b>{order.address||"Dirección no registrada"}</b>{order.reference&&<em>{order.reference}</em>}</p></div>}
          </section>}

          <section className="order-detail-consumption" aria-labelledby="order-detail-products-title">
            <header className="order-detail-section-head"><h3 id="order-detail-products-title">Productos del pedido</h3><span>{itemCount} unidad{itemCount===1?"":"es"}</span></header>
            <div className="order-detail-items">
              {items.map(item=><div className="order-detail-line" key={item.id}>
                <b className="order-detail-qty">{Number(item.qty)}×</b>
                <div className="order-detail-line-info">
                  <span>{item.name}</span>
                  <small>{money(item.unitPrice)} c/u</small>
                  {item.itemType==="combo"&&(item.selections??[]).length>0&&<div className="order-detail-selections">{(item.selections??[]).map(selection=><small key={selection.groupId+selection.productId}><b>{selection.groupName}:</b> {selection.name}{Number(selection.surcharge)>0?` (+${money(selection.surcharge)})`:""}</small>)}</div>}
                  {item.note&&<em>{item.note}</em>}
                  <OrderItemService order={order} itemId={item.id}/>
                </div>
                <strong className="order-detail-line-total">{money(Number(item.qty)*Number(item.unitPrice))}</strong>
              </div>)}
              {!items.length&&<div className="order-detail-empty"><Icon name="receipt" size={20}/><span>Sin ítems cargados aún.</span></div>}
            </div>
            {order.notes&&<div className="order-detail-notes"><span><Icon name="edit" size={16}/></span><div><b>Notas generales</b><p>{order.notes}</p></div></div>}
          </section>

          <section className={`order-detail-totals${remaining<=0.00001?" is-paid":""}`} aria-labelledby="order-detail-account-title">
            <h3 id="order-detail-account-title" className="sr-only">Resumen de cuenta</h3>
            <div className="order-detail-subtotal"><span>Subtotal</span><b>{money(order.subtotal)}</b></div>
            {Number(order.deliveryFee)>0&&<div className="order-detail-subtotal"><span>Delivery</span><b>{money(order.deliveryFee)}</b></div>}
            <div className={`order-detail-subtotal order-detail-paid${hasPayments?" has-payment":""}`}><span>Pagado</span><b>{money(paid)}</b></div>
            <div className="order-detail-grand"><span>{remaining>0.00001?"Saldo pendiente":"Total pagado"}</span><strong>{money(remaining>0.00001?remaining:order.total)}</strong></div>
          </section>
        </div>
      </div>

      {(canManage||canCharge)&&<footer className="order-detail-actions">
        {order.status==="entregado"&&!canCharge&&<p className="order-detail-notice is-success" role="status"><Icon name="circleCheck" size={18}/><span>{order.completedAt?(order.channel==="salon"?"Mesa liberada":"Pedido completado"):"Pedido entregado"}</span></p>}
        {(order.status==="nuevo"||order.status==="confirmado")&&hasPayments&&<p className="order-detail-notice" role="note"><Icon name="info" size={18}/><span>Devuelve los pagos en POS antes de cancelar.</span></p>}
        <div className="order-detail-buttons" role="group" aria-label={table?"Acciones de la mesa":"Acciones del pedido"}>
          <OrderAccountActions order={order} onAdd={()=>add(order)} busy={busy}/>
          {canManage&&(editable||canCancel||(action&&canCharge))&&<div className="order-detail-secondary-actions">
            {action&&canCharge&&<Button icon={actionIcon} kind="secondary" className="order-detail-outline" disabled={busy} aria-busy={busy} onClick={()=>advance(action.status)}>{action.label}</Button>}
            {editable&&edit&&<Button icon="edit" kind="secondary" className="order-detail-outline" disabled={busy} onClick={()=>edit(order)}>Editar comanda</Button>}
            {canCancel&&<Button icon="cancel" kind="ghost" className="order-detail-cancel" disabled={busy} onClick={()=>cancel(order)}>Cancelar pedido</Button>}
          </div>}
          {canCharge?<Link href={`${pageRoutes.pos}?orderId=${order.id}`} className="button primary order-detail-pay" aria-disabled={busy} aria-busy={busy} tabIndex={busy?-1:undefined} onClick={event=>{if(busy)event.preventDefault();}}><Icon name="payment" size={18}/><span>Cobrar saldo</span></Link>
          :canManage&&action&&<Button icon={actionIcon} className="order-detail-primary" disabled={busy} aria-busy={busy} onClick={()=>advance(action.status)}>{action.label}</Button>}
        </div>
      </footer>}
    </>}
  </>;
}

/** Skeleton con la geometría final: cabecera, atribución, productos, cuenta y acciones. */
export function OrderDetailSkeleton({variant,close}:{variant:OrderDetailVariant;close:()=>void}){
  return <>
    <div className="modal-accent" aria-hidden="true"/>
    <header>
      <Skeleton width="var(--size-32)" height="var(--size-32)" radius="var(--radius-tile)" className="order-detail-skeleton-icon"/>
      <div className="order-detail-heading">
        <h2 id={orderDetailTitleId} className="sr-only">{variant==="table"?"Cargando detalle de la mesa":"Cargando detalle del pedido"}</h2>
        <span className="order-detail-skeleton-copy" aria-hidden="true">
          <Skeleton width="var(--size-92)" height="var(--size-9)"/>
          <Skeleton width="var(--size-138)" height="var(--size-16)"/>
          <Skeleton width="var(--size-150)" height="var(--size-24)" radius="var(--radius-pill)"/>
        </span>
      </div>
      <button type="button" aria-label="Cerrar detalle" onClick={close}><Icon name="close"/></button>
    </header>
    <div className="order-detail-body" role="status" aria-label={variant==="table"?"Cargando detalle de la mesa":"Cargando detalle del pedido"}>
      <div className="order-detail-content" aria-hidden="true">
        <OrderAttributionSkeleton/>
        <section className="order-detail-consumption">
          <div className="order-detail-section-head"><Skeleton width="var(--size-156)" height="var(--size-13)"/><Skeleton width="var(--size-48)" height="var(--size-9)"/></div>
          <div className="order-detail-items">
            {Array.from({length:3},(_,index)=><div className="order-detail-line" key={index}>
              <Skeleton width="var(--size-32)" height="var(--size-32)" radius="var(--radius-pill)" className="order-detail-skeleton-qty"/>
              <span className="order-detail-skeleton-copy"><Skeleton width="min(var(--size-220),72%)" height="var(--size-11)"/><Skeleton width="var(--size-92)" height="var(--size-9)"/></span>
              <Skeleton width="var(--size-66)" height="var(--size-10)"/>
            </div>)}
          </div>
        </section>
        <section className="order-detail-totals">
          {Array.from({length:2},(_,index)=><div className="order-detail-subtotal" key={index}><Skeleton width="var(--size-72)" height="var(--size-9)"/><Skeleton width="var(--size-68)" height="var(--size-16)"/></div>)}
          <div className="order-detail-grand"><Skeleton width="var(--size-116)" height="var(--size-10)"/><Skeleton width="var(--size-108)" height="var(--size-24)"/></div>
        </section>
      </div>
    </div>
    <footer className="order-detail-actions order-detail-skeleton-actions" aria-hidden="true">
      <div className="order-detail-buttons"><Skeleton className="secondary"/><Skeleton className="primary"/></div>
    </footer>
  </>;
}
