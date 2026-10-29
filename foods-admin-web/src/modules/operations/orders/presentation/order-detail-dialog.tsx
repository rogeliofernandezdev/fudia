"use client";
import "./order-detail.css";
import Link from "next/link";
import {useState} from "react";
import {Button,ErrorState,Icon,IconTile,Skeleton,Status,Tabs} from "@/design-system";
import {ModalHeader,ModalFooter,ModalWorkspace,ModalItemList,ModalItemNote,ModalItemComponents} from "@/design-system/modal";
import {useMoney,useSession} from "@/providers";
import {pageRoutes} from "@/shared/routing/page-routes";
import {canCancelOrder,canManageOrderService,nextOrderAction} from "../domain/order-actions";
import {canChargeAccount,canCollectOrder} from "../domain/service-flow";
import {deliveryPoint} from "../domain/delivery-point";
import {formatRegionalDateTime,formatRegionalNumber} from "@/shared/i18n/regional-format";
import type {Order,OrderItem,OrderItemSelection} from "../domain/types";
import type {ServiceItem} from "../domain/service-flow";
import {OrderAttribution,OrderAttributionSkeleton} from "./order-attribution";
import {OrderTimeline} from "./order-timeline";
import {channelIcons,isEditableOrder,openedAgo,orderStatus,registeredAt,serviceStatusTone,tableTitle} from "./order-meta";
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
  const{location,user,can,cashCollection}=useSession();
  const money=useMoney();
  const[tab,setTab]=useState<"products"|"tracking">("products");
  const table=variant==="table";
  const canManage=Boolean(hasPermission&&order&&canManageOrderService(order,user?.id));
  const meta=order?orderStatus(order.status):null;
  const action=order?nextOrderAction(order):null;
  const paid=Number(order?.paidAmount??0);
  const remaining=Number(order?.remainingAmount??order?.total??0);
  const hasPayments=paid>0.00001;
  const canCharge=Boolean(order&&canCollectOrder(order,user?.id,can("cash.manage"),cashCollection)&&canChargeAccount(order)&&remaining>0.00001);
  const canCancel=Boolean(order&&canCancelOrder(order));
  const editable=Boolean(edit&&order&&isEditableOrder(order)&&(!(order.serviceItems?.length)||order.status==="nuevo"));
  const actionIcon=action?.status==="entregado"?"availability":action?.status==="confirmado"?"chefHat":action?.icon;
  const items=order?.items??[];
  const point=order?deliveryPoint(order):null;
  // Si todas las preparaciones repiten el estado del pedido (Nuevo/Sin enviar, Preparando, Listo, Entregado), no se muestran por línea.
  // Confirmado se conserva: «Pendiente» indica que la comanda enviada aún no empieza a prepararse.
  const services=order?.serviceItems??[];
  const sharedService=Boolean(order&&order.status!=="confirmado"&&services.length>0&&services.every(service=>service.status===order.status));
  const title=table||order?.channel==="salon"?tableTitle(order?.tableName):order?(order.tableName||order.customerName||"Pedido"):"Pedido";
  const subtitle=order?(table?order.customerName:order.tableName&&order.customerName?order.customerName:""):"";
  const opened=order?(table?openedAgo(order.createdAt,location?.country,location?.timezone):formatRegionalDateTime(order.createdAt,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium"})):"";
  return <>

    <ModalHeader title={title} titleId={orderDetailTitleId} icon={table?"utensils":order?channelIcons[order.channel]??"receipt":"receipt"} onClose={close} closeLabel="Cerrar detalle"/>

    {error?(
      <div className="modal-body order-detail-body"><ErrorState title="No pudimos cargar el pedido" text={error} onRetry={retry} retrying={retrying}/></div>
    ):order&&meta&&<>
      <ModalWorkspace summary={
        <section className={`modal-summary order-detail-totals${remaining<=0.00001?" is-paid":""}`} aria-labelledby="order-detail-account-title">
          <div className="modal-summary-heading"><h3 id="order-detail-account-title" className="modal-section-title">Resumen de cuenta</h3>{order.channel==="salon"&&<Status tone={accountTone(order)} icon="receipt" emphasis>{accountLabel(order)}</Status>}</div>
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
          <section className="order-detail-hero" aria-label="Resumen del pedido">
            <div className="order-detail-identity">
              <div className="order-detail-identity-head"><div className="order-detail-status" aria-label="Estado del pedido"><Status tone={meta.tone}>{meta.label}</Status></div></div>
              <div className="order-detail-metadata">
                <div className="order-detail-meta-item"><small><Icon name="clock" size={16}/>{table?"Apertura":"Fecha del pedido"}</small><time dateTime={order.createdAt} aria-label={registeredAt(order.createdAt,location?.country,location?.timezone)}>{opened}</time></div>
                <OrderAttribution channel={order.channel} waiterName={order.waiterName} collectedByNames={order.collectedByNames}/>
              </div>
            </div>
            {(subtitle||order.customerName||order.customerPhone)&&<div className="order-detail-contact">
              <IconTile name="user" tone="teal"/>
              <div><b>{subtitle||order.customerName||"Cliente sin nombre"}</b><small>{order.customerPhone||"Sin teléfono registrado"}</small></div>
              {order.customerPhone&&<a className="button secondary order-detail-call" href={`tel:${order.customerPhone.replace(/\s+/g,"")}`} aria-label={`Llamar a ${order.customerPhone}`}><Icon name="phone" size={16}/><span>Llamar</span></a>}
            </div>}
            {/* Sin pestaña «Detalles»: mesa, cliente, teléfono y fecha ya están arriba; la entrega es el único dato propio. */}
            {(order.address||order.reference||point)&&<section className="order-detail-context" aria-label="Entrega">
              <div><span><Icon name="mapPin" size={16}/></span><p><small>Entrega</small><b>{order.address||"Dirección no registrada"}</b>{order.reference&&<em>{order.reference}</em>}
                {point&&<span className="order-detail-map">
                  <a className="order-detail-map-link" href={point.mapUrl} target="_blank" rel="noopener noreferrer" aria-label="Ver ubicación de entrega en Google Maps (se abre en otra pestaña)">Ver ubicación</a>
                  {point.distanceKm!==null&&<span className="order-detail-map-distance">{formatRegionalNumber(point.distanceKm,location?.country,{maximumFractionDigits:1})} km</span>}
                </span>}
              </p></div>
            </section>}
          </section>

          <Tabs className="order-detail-tabs" label="Información del pedido" value={tab} onChange={setTab} items={[
            {value:"products",label:"Productos",icon:"dish",count:items.length},
            {value:"tracking",label:"Seguimiento",icon:"route"},
          ]}/>

          <div className="order-detail-panel" role="tabpanel" aria-labelledby="order-detail-products-title" hidden={tab!=="products"}>
            <h3 id="order-detail-products-title" className="sr-only">Productos del pedido</h3>
            <section className="order-detail-consumption" aria-labelledby="order-detail-products-title">
              <ModalItemList className="order-detail-items">
                {items.map(item=><OrderDetailLine key={item.id} order={order} item={item} hideStatus={sharedService}/>)}
                {!items.length&&<div className="order-detail-empty" role="listitem"><Icon name="receipt" size={20}/><span>Sin ítems cargados aún.</span></div>}
              </ModalItemList>
            </section>
            {order.notes&&<section className="modal-detail-notes order-detail-notes" aria-labelledby="order-detail-notes-title"><span><Icon name="edit" size={16}/></span><div><h3 id="order-detail-notes-title">Notas generales</h3><p>{order.notes}</p></div></section>}
          </div>

          <div className="order-detail-panel" role="tabpanel" aria-label="Seguimiento" hidden={tab!=="tracking"}>
            <OrderTimeline order={order} country={location?.country} timeZone={location?.timezone}/>
          </div>
        </div>
      </div>
      </ModalWorkspace>

      {(canManage||canCharge)&&<ModalFooter className="order-detail-actions">
        {order.status==="entregado"&&!canCharge&&<p className="order-detail-notice is-success" role="status"><Icon name="circleCheck" size={18}/><span>{order.completedAt?(order.channel==="salon"?"Mesa liberada":"Pedido completado"):"Pedido entregado"}</span></p>}
        {(order.status==="nuevo"||order.status==="confirmado")&&hasPayments&&<p className="order-detail-notice" role="note"><Icon name="info" size={18}/><span>Devuelve los pagos en POS antes de cancelar.</span></p>}
        <div className="order-detail-buttons modal-action-layout" role="group" aria-label={table?"Acciones de la mesa":"Acciones del pedido"}>
          <OrderAccountActions order={order} onAdd={()=>add(order)} busy={busy}/>
          {canManage&&(editable||canCancel||(action&&canCharge))&&<div className="order-detail-secondary-actions modal-action-subgroup">
            {action&&canCharge&&<Button icon={actionIcon} kind="secondary" className="order-detail-outline" disabled={busy} aria-busy={busy} onClick={()=>advance(action.status)}>{action.label}</Button>}
            {editable&&edit&&<Button icon="edit" kind="secondary" className="order-detail-outline" disabled={busy} onClick={()=>edit(order)}>Editar comanda</Button>}
            {canCancel&&<Button action="cancel-record" className="order-detail-cancel" aria-label="Cancelar pedido" disabled={busy} onClick={()=>cancel(order)}>Cancelar</Button>}
          </div>}
          {canCharge?<Link href={`${pageRoutes.pos}?orderId=${order.id}`} className="button primary order-detail-pay modal-action-main" aria-disabled={busy} aria-busy={busy} tabIndex={busy?-1:undefined} onClick={event=>{if(busy)event.preventDefault();}}><Icon name="payment" size={18}/><span>Cobrar saldo</span></Link>
          :canManage&&action&&<Button icon={actionIcon} className="order-detail-primary modal-action-main" disabled={busy} aria-busy={busy} onClick={()=>advance(action.status)}>{action.label}</Button>}
        </div>
      </ModalFooter>}
    </>}
  </>;
}

/** Empareja cada selección del combo con su preparación (mismo producto) para presentarlas juntas en el bloque de componentes. */
function comboComponents(item:OrderItem,services:ServiceItem[]){
  const used=new Set<string>();
  return (item.selections??[]).map(selection=>{
    const service=services.find(candidate=>candidate.orderItemId===item.id&&candidate.productId===selection.productId&&!used.has(candidate.id));
    if(service)used.add(service.id);
    return {selection,service} as {selection:OrderItemSelection;service?:ServiceItem};
  });
}

/** Fila de producto: cantidad, nombre e importe; debajo precio c/u y, en combos, el bloque neutro de componentes a modo de carta (grupo | plato, destino si difiere y estado). */
function OrderDetailLine({order,item,hideStatus}:{order:Order;item:OrderItem;hideStatus:boolean}){
  const money=useMoney();
  const components=item.itemType==="combo"?comboComponents(item,order.serviceItems??[]):[];
  const shown=new Set(components.flatMap(component=>component.service?[component.service.id]:[]));
  const combo=components.length>0;
  // Si todos los componentes van al mismo destino, se indica una sola vez junto al precio en lugar de repetirlo por fila.
  const destinations=new Set(components.map(component=>component.service?.destinationLabel));
  const sharedDestination=combo&&destinations.size===1&&!destinations.has(undefined)?components[0].service?.destinationLabel:undefined;
  return <div className="modal-item-row order-detail-line" role="listitem">
    <b className="modal-item-quantity order-detail-qty">{Number(item.qty)}×</b>
    <div className="order-detail-line-info">
      <span>{item.name}</span>
      {item.note&&<ModalItemNote>{item.note}</ModalItemNote>}
      <OrderItemService order={order} itemId={item.id} lead={<><small>{money(item.unitPrice)} c/u</small>{sharedDestination&&<small>{sharedDestination}</small>}</>} hideStatus={hideStatus} exclude={shown}/>
      <ModalItemComponents label={`Componentes de ${item.name}`} items={components.map(({selection,service})=>({
        key:selection.groupId+selection.productId,
        group:selection.groupName,
        name:selection.name,
        extra:Number(selection.surcharge)>0&&<small>+{money(selection.surcharge)}</small>,
        side:<>{service&&!sharedDestination&&<span>{service.destinationLabel}</span>}{service&&!hideStatus&&<Status tone={serviceStatusTone(service.status)}>{service.statusLabel}</Status>}</>,
      }))}/>
    </div>
    <strong className="order-detail-line-total">{money(Number(item.qty)*Number(item.unitPrice))}</strong>
  </div>;
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
        <div className="order-detail-information modal-inline-metadata"><OrderAttributionSkeleton/><span className="modal-inline-date"><Skeleton width="var(--size-150)" height="var(--size-9)"/></span></div>
        <section className="order-detail-consumption">
          <div className="order-detail-section-head modal-section-rule"><Skeleton width="var(--size-156)" height="var(--size-13)"/><Skeleton width="var(--size-48)" height="var(--size-9)"/></div>
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
      <div className="order-detail-buttons modal-action-layout" aria-hidden="true"><Skeleton className="secondary"/><Skeleton className="primary modal-action-main"/></div>
    </ModalFooter>
  </>;
}
