"use client";
import {useMutation,useQueryClient} from "@tanstack/react-query";
import {Button,Status} from "@/design-system";
import {useFeedback} from "@/providers";
import {canRetryDispatch,canUseOwnDelivery,dispatchStatusMeta,isDispatchStatus} from "../domain/dispatch-status";
import {retryDeliveryDispatch,switchToOwnDelivery} from "../infrastructure/delivery-api";
import "./delivery.css";

type DispatchOrder={id:string;deliveryDispatchStatus?:string;deliveryPartnerName?:string;deliveryPartnerFee?:string};

/**
 * Despacho del pedido a las centrales de Delivery Fudia dentro del bloque de
 * entrega del detalle: estado, central y, si ninguna aceptó, Reintentar o
 * Usar delivery propio. Sin despacho no se muestra nada.
 */
export function OrderDispatchBlock({order,canManage,money,disabled}:{order:DispatchOrder;canManage:boolean;money:(value:string)=>string;disabled?:boolean}){
  const{notify}=useFeedback();
  const client=useQueryClient();
  const refresh=()=>{
    for(const key of ["order","orders","pos-order","pos-orders","staff-notices"])void client.invalidateQueries({queryKey:[key]});
  };
  const retry=useMutation({
    mutationFn:()=>retryDeliveryDispatch(order.id),
    onSuccess:()=>{refresh();notify({tone:"success",title:"Pedido ofrecido de nuevo",message:"La 1.ª central lo recibe ahora por WhatsApp."});},
    onError:e=>notify({tone:"danger",title:"No se pudo reintentar",message:e.message}),
  });
  const own=useMutation({
    mutationFn:()=>switchToOwnDelivery(order.id),
    onSuccess:()=>{refresh();notify({tone:"success",title:"Delivery propio",message:"Ninguna central recogerá este pedido; despáchalo con tu repartidor."});},
    onError:e=>notify({tone:"danger",title:"No se pudo cambiar",message:e.message}),
  });
  const status=order.deliveryDispatchStatus;
  if(!isDispatchStatus(status))return null;
  const meta=dispatchStatusMeta[status];
  const busy=disabled||retry.isPending||own.isPending;
  const partner=status==="own"||status==="cancelled"?"":order.deliveryPartnerName;
  return <div className="order-dispatch" aria-label="Despacho a centrales">
    <div className="order-dispatch-state">
      <Status tone={meta.tone} icon={meta.icon}>{meta.label}</Status>
      {partner&&<b>{partner}</b>}
    </div>
    {order.deliveryPartnerFee&&<p className="order-dispatch-fee">Envío {money(order.deliveryPartnerFee)}: lo cobra la central al cliente.</p>}
    {canManage&&(canRetryDispatch(status)||canUseOwnDelivery(status))&&<div className="order-dispatch-actions">
      {canRetryDispatch(status)&&<Button kind="secondary" icon="refresh" disabled={busy} loading={retry.isPending} onClick={()=>retry.mutate()}>Reintentar centrales</Button>}
      {canUseOwnDelivery(status)&&<Button kind="ghost" icon="store" disabled={busy} loading={own.isPending} onClick={()=>own.mutate()}>Usar delivery propio</Button>}
    </div>}
  </div>;
}
