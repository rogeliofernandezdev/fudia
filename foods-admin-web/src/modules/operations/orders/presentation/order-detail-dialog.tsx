"use client";
import "./order-detail.css";
import Link from "next/link";
import {Button,ErrorState,Icon,Skeleton,Status} from "@/design-system";
import {ModalHeader,ModalFooter,ModalWorkspace,ModalItemList} from "@/design-system/modal";
import {useMoney,useSession} from "@/providers";
import {pageRoutes} from "@/shared/routing/page-routes";
import {canCancelOrder,canManageOrderService,nextOrderAction} from "../domain/order-actions";
import {canChargeAccount} from "../domain/service-flow";
import type {Order} from "../domain/types";
import {OrderAttribution,OrderAttributionSkeleton} from "./order-attribution";
import {channelIcons,isEditableOrder,openedAgo,orderStatus,registeredAt,tableTitle} from "./order-meta";
import {OrderAccountActions,OrderItemService,accountLabel,accountTone} from "./order-service-controls";

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
  const title=table||order?.channel==="salon"?tableTitle(order?.tableName):order?(order.tableName||order.customerName||"Pedido"):"Pedido";
  const subtitle=order?(table?order.customerName:order.tableName&&order.customerName?order.customerName:""):"";
  const opened=order?(table?openedAgo(order.createdAt,location?.country,location?.timezone):registeredAt(order.createdAt,location?.country,location?.timezone)):"";
  return <>

    <ModalHeader title={title} titleId={orderDetailTitleId} icon={table?"utensils":order?channelIcons[order.channel]??"receipt":"receipt"} onClose={close} closeLabel="Cerrar detalle"/>

    {error?(
      <div className="modal-body order-detail-body"><ErrorState title="No pudimos cargar el pedido" text={error} onRetry={retry} retrying={retrying}/></div>
    ):order&&meta&&<>
      <ModalWorkspace summary={
        <section className={`modal-summary order-detail-totals${remaining<=0.00001?" is-paid":""}`} aria-labelledby="order-detail-account-title">
          <div className="modal-summary-heading"><h3 id="order-detail-account-title">Resumen de cuenta</h3>{order.channel==="salon"&&<Status tone={accountTone(order)} icon="receipt" emphasis>{accountLabel(order)}</Status>}</div>
          <div className="modal-summary-lines">
            <div className="modal-summary-line"><span>Subtotal</span><b>{money(order.subtotal)}</b></div>
            {Number(order.deliveryFee)>0&&<div className="modal-summary-line"><span>Delivery</span><b>{money(order.deliveryFee)}</b></div>}
            <div className={`modal-summary-line order-detail-paid${hasPayments?" has-payment":""}`}><span>Pagado</span><b>{money(paid)}</b></div>
          </div>
          <div className="modal-summary-total"><span>{remaining>0.00001?"Saldo pendiente":"Total pagado"}</span><strong>{money(remaining>0.00001?remaining:order.total)}</strong></div>
        </section>
      }>
      <div className="modal-body order-detail-body" tabIndex={0} aria-label="Productos y datos del pedido">
        <div className="order-detail-content">
          <div className="order-detail-information" aria-label="Datos del pedido">
            <OrderAttribution channel={order.channel} waiterName={order.waiterName} collectedByNames={order.collectedByNames}/>
            <span className="order-detail-opened"><Icon name="clock" size={14}/><time dateTime={order.createdAt}>{opened}</time></span>
            {subtitle&&<span>{subtitle}</span>}
          </div>
          <header className="order-detail-section-head"><h3 id="order-detail-products-title">Productos del pedido</h3><div className="order-detail-status" aria-label="Estado del pedido"><Status tone={meta.tone}>{meta.label}</Status></div></header>
          {(order.customerPhone||order.address||order.reference)&&<section className="order-detail-context" aria-label="Contacto y entrega">
            {order.customerPhone&&<div><span><Icon name="users" size={16}/></span><p><small>CONTACTO</small><b>{order.customerPhone}</b></p></div>}
            {(order.address||order.reference)&&<div><span><Icon name="truck" size={16}/></span><p><small>ENTREGA</small><b>{order.address||"Dirección no registrada"}</b>{order.reference&&<em>{order.reference}</em>}</p></div>}
          </section>}

          <section className="order-detail-consumption" aria-labelledby="order-detail-products-title">
            <ModalItemList className="order-detail-items">
              {items.map(item=><div className="modal-item-row order-detail-line" role="listitem" key={item.id}>
                <b className="modal-item-quantity order-detail-qty">{Number(item.qty)}×</b>
                <div className="order-detail-line-info">
                  <span>{item.name}</span>
                  <small>{money(item.unitPrice)} c/u</small>
                  {item.itemType==="combo"&&(item.selections??[]).length>0&&<div className="order-detail-selections">{(item.selections??[]).map(selection=><small key={selection.groupId+selection.productId}><b>{selection.groupName}:</b> {selection.name}{Number(selection.surcharge)>0?` (+${money(selection.surcharge)})`:""}</small>)}</div>}
                  {item.note&&<em>{item.note}</em>}
                  <OrderItemService order={order} itemId={item.id}/>
                </div>
                <strong className="order-detail-line-total">{money(Number(item.qty)*Number(item.unitPrice))}</strong>
              </div>)}
              {!items.length&&<div className="order-detail-empty" role="listitem"><Icon name="receipt" size={20}/><span>Sin ítems cargados aún.</span></div>}
            </ModalItemList>
          </section>
          {order.notes&&<section className="modal-detail-notes order-detail-notes" aria-labelledby="order-detail-notes-title"><span><Icon name="edit" size={16}/></span><div><h3 id="order-detail-notes-title">Notas generales</h3><p>{order.notes}</p></div></section>}

        </div>
      </div>
      </ModalWorkspace>

      {(canManage||canCharge)&&<ModalFooter className="order-detail-actions">
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
      </ModalFooter>}
    </>}
  </>;
}

/** Skeleton con la geometría final: cabecera, atribución, productos, cuenta y acciones. */
export function OrderDetailSkeleton({variant,close}:{variant:OrderDetailVariant;close:()=>void}){
  return <>

    <ModalHeader title={<><span className="sr-only">{variant==="table"?"Cargando detalle de la mesa":"Cargando detalle del pedido"}</span><Skeleton width="var(--size-138)" height="var(--size-16)"/></>} titleId={orderDetailTitleId}
      iconSlot={<Skeleton width="var(--size-44)" height="var(--size-44)" radius="var(--radius-tile)" className="order-detail-skeleton-icon"/>} onClose={close} closeLabel="Cerrar detalle"/>
    <ModalWorkspace summary={
      <section className="modal-summary" aria-hidden="true">
        <div className="modal-summary-heading"><Skeleton width="var(--size-138)" height="var(--size-13)"/><Skeleton width="var(--size-108)" height="var(--size-24)" radius="var(--radius-pill)"/></div>
        <div className="modal-summary-lines">
          {Array.from({length:2},(_,index)=><div className="modal-summary-line" key={index}><Skeleton width="var(--size-72)" height="var(--size-9)"/><Skeleton width="var(--size-68)" height="var(--size-16)"/></div>)}
        </div>
        <div className="modal-summary-total"><Skeleton width="var(--size-116)" height="var(--size-10)"/><Skeleton width="var(--size-108)" height="var(--size-24)"/></div>
      </section>
    }>
    <div className="modal-body order-detail-body">
      <div className="order-detail-content" aria-hidden="true">
        <div className="order-detail-information"><OrderAttributionSkeleton/><Skeleton width="var(--size-150)" height="var(--size-9)"/></div>
        <section className="order-detail-consumption">
          <div className="order-detail-section-head"><Skeleton width="var(--size-156)" height="var(--size-13)"/><Skeleton width="var(--size-48)" height="var(--size-9)"/></div>
          <ModalItemList className="order-detail-items">
            {Array.from({length:3},(_,index)=><div className="modal-item-row order-detail-line" role="listitem" key={index}>
              <Skeleton width="var(--size-32)" height="var(--size-32)" radius="var(--radius-pill)" className="order-detail-skeleton-qty"/>
              <span className="order-detail-skeleton-copy"><Skeleton width="min(var(--size-220),72%)" height="var(--size-11)"/><Skeleton width="var(--size-92)" height="var(--size-9)"/></span>
              <Skeleton width="var(--size-66)" height="var(--size-10)"/>
            </div>)}
          </ModalItemList>
        </section>
      </div>
    </div>
    </ModalWorkspace>
    <ModalFooter className="order-detail-actions order-detail-skeleton-actions">
      <div className="order-detail-buttons" aria-hidden="true"><Skeleton className="secondary"/><Skeleton className="primary"/></div>
    </ModalFooter>
  </>;
}
