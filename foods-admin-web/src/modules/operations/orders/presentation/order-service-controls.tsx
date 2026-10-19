"use client";
import type {ReactNode} from "react";
import {useMutation,useQueryClient} from "@tanstack/react-query";
import {Button,Status} from "@/design-system";
import {useSession,useFeedback} from "@/providers";
import {canManageOrderService} from "../domain/order-actions";
import {canAddConsumption,canCloseAccount,type TableAccount} from "../domain/service-flow";
import {closeOrderAccount} from "../infrastructure/service-api";
import {serviceStatusTone} from "./order-meta";
import "./order-service.css";

function useServiceMutation(){
 const qc=useQueryClient();const{notify}=useFeedback();
 return useMutation({mutationFn:({orderId}:{orderId:string})=>closeOrderAccount(orderId),onSuccess:order=>{
  qc.setQueryData(["order",order.id],order);
  for(const key of ["order","orders","salon-floor","pos-orders","pos-order","kitchen-tickets","dashboard","sales"])void qc.invalidateQueries({queryKey:[key]});
 },onError:error=>notify({tone:"danger",title:"No se pudo actualizar",message:error.message})});
}

/** Destino y estado por producto. `lead` (precio unitario) comparte la fila con el destino; el estado queda alineado a la derecha y se omite con `hideStatus` cuando todas las preparaciones comparten estado y el estado del pedido ya lo expresa. `exclude` omite preparaciones ya presentadas en la línea de componentes del combo. */
export function OrderItemService({order,itemId,lead,hideStatus=false,exclude}:{order:TableAccount;itemId:string;lead?:ReactNode;hideStatus?:boolean;exclude?:ReadonlySet<string>}){
 const items=(order.serviceItems??[]).filter(item=>item.orderItemId===itemId&&!exclude?.has(item.id));
 if(!items.length)return lead?<div className="order-service-lines"><div className="order-service-line"><div>{lead}</div></div></div>:null;
 const single=items.length===1;
 return <div className="order-service-lines">
  {!single&&lead&&<div className="order-service-line"><div>{lead}</div></div>}
  {items.map(item=><div className="order-service-line" key={item.id}>
   <div>{single&&lead}{!single&&<span>{item.name}</span>}<small>{item.destinationLabel}</small></div>{!hideStatus&&<Status tone={serviceStatusTone(item.status)}>{item.statusLabel}</Status>}
  </div>)}
 </div>;
}

export function OrderAccountActions({order,onAdd,busy=false}:{order:TableAccount;onAdd?:()=>void;busy?:boolean}){
 const{user,can}=useSession();const mutation=useServiceMutation();
 const manage=can("orders.manage")&&canManageOrderService(order,user?.id)&&!order.completedAt&&order.status!=="cancelado";
 if(order.channel!=="salon"||!manage)return null;
 return <>
  {onAdd&&canAddConsumption(order)&&<Button kind="secondary" icon="plus" disabled={busy||mutation.isPending} onClick={onAdd}>Agregar productos</Button>}
  {canCloseAccount(order)&&<Button icon="receipt" disabled={busy||mutation.isPending} loading={mutation.isPending} onClick={()=>mutation.mutate({orderId:order.id})}>{mutation.isPending?"Cerrando…":"Cerrar cuenta"}</Button>}
 </>;
}

export function accountLabel(order:TableAccount){return order.completedAt?"Finalizada":order.accountState==="paid"?"Pagada":order.billClosedAt?"Por cobrar":"Cuenta abierta"}
export function accountTone(order:TableAccount):"sky"|"green"|"orange"|"gray"{return order.completedAt?"gray":order.accountState==="paid"?"green":order.billClosedAt?"orange":"sky"}
