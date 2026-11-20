"use client";
import {useQuery} from "@tanstack/react-query";
import {DetailFacts,ErrorState,Skeleton,Status} from "@/design-system";
import {useSession} from "@/providers";
import {isPermissionError} from "@/shared/api/client";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {formatPartnerPhone,pickupAlertLabels,waitMinutes} from "../domain/board";
import {dispatchStatusMeta} from "../domain/dispatch-status";
import {getDeliveryDispatch} from "../infrastructure/delivery-api";
import "./delivery.css";

type DispatchOrder={id:string;code:string;status:string;completedAt?:string;deliveryPartnerFee?:string};

/** Seguimiento de solo lectura; las acciones de reparto viven en la fila de Pedidos. */
export function OrderDispatchBlock({order,money}:{order:DispatchOrder;money:(value:string)=>string}){
  const {location}=useSession();
  const queryKey=["delivery-dispatch",order.id,order.status];
  const closed=Boolean(order.completedAt)||order.status==="entregado"||order.status==="cancelado";
  const dispatch=useQuery({queryKey,queryFn:()=>getDeliveryDispatch(order.id),retry:false,meta:{live:["delivery","orders"],poll:query=>closed||isPermissionError(query.state.error)?false:10_000}});
  const data=dispatch.data;
  const meta=data?dispatchStatusMeta[data.status]:null;
  const assigned=data?.status==="accepted"||data?.status==="picked_up"||data?.status==="delivered";
  const date=(value:string)=>formatRegionalDateTime(value,{country:location?.country,timeZone:location?.timezone});
  const pickup=data?.pickupAlert?pickupAlertLabels[data.pickupAlert]:null;
  const issue=data?.failedDelivery?"Entrega fallida":data?.issueReportedAt?"No pudo entregar":"";
  return <section className="order-dispatch" aria-label="Reparto">
    <h3 className="modal-section-title">Reparto</h3>
    {dispatch.isLoading?<div className="order-dispatch-loading" role="status" aria-label="Cargando reparto"><Skeleton width="var(--size-138)" height="var(--size-24)"/><Skeleton height="var(--size-44)"/></div>:
      dispatch.isError?<ErrorState title="No pudimos cargar el reparto" text={dispatch.error.message} retrying={dispatch.isFetching} onRetry={isPermissionError(dispatch.error)?undefined:()=>void dispatch.refetch()}/>:
      !data?<p className="order-dispatch-note">Este pedido aún no tiene despacho a centrales registrado.</p>:
      meta&&<>
        <div className="order-dispatch-state"><Status tone={meta.tone} icon={meta.icon}>{meta.label}</Status></div>
        {assigned&&<DetailFacts items={[
          ...(data.partnerName?[{label:"Central",value:<>{data.partnerName}{data.partnerPhone&&<a href={`tel:+${data.partnerPhone}`}>{formatPartnerPhone(data.partnerPhone)}</a>}</>}]:[]),
          {label:"Motorizado",value:data.courierName||"Pendiente de asignación"},
        ]}/>}
        {(data.acceptedAt||data.pickedUpAt||data.deliveredAt)&&<DetailFacts label="Seguimiento del reparto" items={[
          ...(data.acceptedAt?[{label:"Aceptado",value:<time dateTime={data.acceptedAt}>{date(data.acceptedAt)}</time>}]:[]),
          ...(data.pickedUpAt?[{label:"Recogido",value:<time dateTime={data.pickedUpAt}>{date(data.pickedUpAt)}</time>}]:[]),
          ...(data.deliveredAt?[{label:"Entregado",value:<time dateTime={data.deliveredAt}>{date(data.deliveredAt)}</time>}]:[]),
        ]}/>}
        {issue?<div role="status"><Status tone={data.failedDelivery?"red":"orange"} icon="alert">{issue}</Status></div>:
          pickup&&<div role="status"><Status tone={pickup.tone} icon="clock">{pickup.label} · {waitMinutes(data.pickupWaitSeconds??0)}</Status></div>}
        {assigned&&order.deliveryPartnerFee&&<p className="order-dispatch-note">Envío {money(order.deliveryPartnerFee)}: lo cobra la central al cliente.</p>}
      </>}
  </section>;
}
