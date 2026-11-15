"use client";
import {useState} from "react";
import {createPortal} from "react-dom";
import {useMutation,useQueryClient} from "@tanstack/react-query";
import {ConfirmDialog,IconButton} from "@/design-system";
import {useFeedback,useSession} from "@/providers";
import {canRetryDispatch,canUseOwnDelivery,isDispatchStatus} from "../domain/dispatch-status";
import {retryDeliveryDispatch,switchToOwnDelivery} from "../infrastructure/delivery-api";

type DispatchActionOrder={id:string;code:string;channel:string;status:string;completedAt?:string;deliveryDispatchStatus?:string};

/** Acciones de reparto en la fila de Pedidos; no consulta el detalle ni el despacho por cada fila. */
export function OrderDispatchActions({order,busy=false}:{order:DispatchActionOrder;busy?:boolean}){
  const {can}=useSession();
  const {notify}=useFeedback();
  const client=useQueryClient();
  const [confirmOwn,setConfirmOwn]=useState(false);
  const status=order.deliveryDispatchStatus;
  const allowed=can("orders.manage")&&order.channel==="delivery"&&!order.completedAt&&order.status!=="entregado"&&order.status!=="cancelado";
  const canRetry=allowed&&isDispatchStatus(status)&&canRetryDispatch(status);
  const canOwn=allowed&&isDispatchStatus(status)&&canUseOwnDelivery(status);
  const refresh=()=>Promise.all(["delivery-dispatch","order","orders","pos-order","pos-orders","sales","dashboard","staff-notices"].map(key=>client.invalidateQueries({queryKey:[key]})));
  const retry=useMutation({
    mutationFn:()=>retryDeliveryDispatch(order.id),
    onSuccess:()=>{notify({tone:"success",title:"Pedido ofrecido de nuevo",message:"Las centrales activas reciben el pedido en su panel de reparto."});return refresh();},
    onError:error=>{void refresh();notify({tone:"danger",title:"No se pudo reintentar",message:error.message});},
  });
  const own=useMutation({
    mutationFn:()=>switchToOwnDelivery(order.id),
    onSuccess:()=>{setConfirmOwn(false);notify({tone:"success",title:"Delivery propio",message:"Ninguna central recogerá este pedido; despáchalo con tu repartidor."});return refresh();},
    onError:error=>{setConfirmOwn(false);void refresh();notify({tone:"danger",title:"No se pudo cambiar",message:error.message});},
  });
  const pending=busy||retry.isPending||own.isPending;
  if(!canRetry&&!canOwn)return null;
  return <div className="order-dispatch-row-actions" role="group" aria-label={`Acciones de reparto de ${order.code}`} onClick={event=>event.stopPropagation()}>
    {canRetry&&<IconButton icon="refresh" label="Reintentar centrales" accessibleLabel={`Reintentar centrales para ${order.code}`} disabled={pending} aria-busy={retry.isPending} onClick={()=>{if(canRetry&&!pending)retry.mutate();}}/>}
    {canOwn&&<IconButton icon="store" label="Usar delivery propio" accessibleLabel={`Usar delivery propio en ${order.code}`} disabled={pending} onClick={()=>setConfirmOwn(true)}/>}
    {confirmOwn&&canOwn&&createPortal(<ConfirmDialog open tone="danger" title="Usar delivery propio" subject={{label:"Pedido",name:order.code}}
      description={status==="accepted"?"La central que aceptó recibirá el aviso y ya no recogerá este pedido. Despáchalo con tu repartidor.":"Ninguna central recogerá este pedido. Despáchalo con tu repartidor."}
      confirmLabel="Usar propio" pending={own.isPending} confirmDisabled={pending||!canOwn} onCancel={()=>setConfirmOwn(false)} onConfirm={()=>{if(canOwn&&!pending)own.mutate();}}/>,document.body)}
  </div>;
}
