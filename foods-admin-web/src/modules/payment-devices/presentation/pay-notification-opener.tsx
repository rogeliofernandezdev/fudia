"use client";

import {useQuery,useQueryClient} from "@tanstack/react-query";
import {ErrorState,Modal,ModalBody,Skeleton} from "@/design-system";
import {getPayNotification} from "../infrastructure/pay-devices-api";
import {PayNotificationDetail} from "./pay-notification-detail";

/** Abre un pago detectado por su id, como lo hace el aviso de la campana: si
 * sigue detectado va directo a «Asociar pago a pedido»; si ya se asoció o se
 * descartó, muestra su detalle. Es el mismo modal del aviso flotante y de Caja. */
export function PayNotificationOpener({id,onClose}:{id:string|null;onClose:()=>void}){
  if(!id)return null;
  return <Opener key={id} id={id} onClose={onClose}/>;
}

function Opener({id,onClose}:{id:string;onClose:()=>void}){
  const qc=useQueryClient();
  const payment=useQuery({
    queryKey:["pay-notification",id],
    queryFn:()=>getPayNotification(id),
    gcTime:0,
    retry:1,
    refetchOnWindowFocus:false,
  });
  if(payment.data)return <PayNotificationDetail notification={payment.data} mode="associate" onClose={onClose}
    onDone={()=>{onClose();void qc.invalidateQueries({queryKey:["staff-notices"]})}}/>;
  return <Modal title="Asociar pago a pedido" className="pay-detail-modal" onClose={onClose}>
    <ModalBody className="pay-detail-body">
      {payment.isError
        ?<ErrorState className="pay-detail-state" title="No pudimos abrir el pago" text={payment.error.message} onRetry={()=>void payment.refetch()} retrying={payment.isFetching}/>
        :<div className="pay-order-loading" role="status" aria-label="Cargando el pago" aria-busy="true">
          <Skeleton height="var(--size-72)" radius="var(--radius-card)"/><Skeleton height="var(--size-56)"/><Skeleton height="var(--size-56)"/><Skeleton height="var(--size-56)"/>
        </div>}
    </ModalBody>
  </Modal>;
}
