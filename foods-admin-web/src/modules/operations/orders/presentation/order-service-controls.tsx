"use client";
import {useMutation,useQueryClient} from "@tanstack/react-query";
import {Button,Status} from "@/design-system";
import {useSession,useFeedback} from "@/providers";
import {canManageOrderService} from "../domain/order-actions";
import {canAddConsumption,canCloseAccount,type TableAccount} from "../domain/service-flow";
import {closeOrderAccount} from "../infrastructure/service-api";
import "./order-service.css";

function useServiceMutation(){
 const qc=useQueryClient();const{notify}=useFeedback();
 return useMutation({mutationFn:({orderId}:{orderId:string})=>closeOrderAccount(orderId),onSuccess:order=>{
  qc.setQueryData(["order",order.id],order);
  for(const key of ["order","orders","salon-floor","pos-orders","pos-order","kitchen-tickets","dashboard","sales"])void qc.invalidateQueries({queryKey:[key]});
 },onError:error=>notify({tone:"danger",title:"No se pudo actualizar",message:error.message})});
}

export function OrderItemService({order,itemId}:{order:TableAccount;itemId:string}){
 const items=(order.serviceItems??[]).filter(item=>item.orderItemId===itemId);
 if(!items.length)return null;
 return <div className="order-service-lines">{items.map(item=><div className="order-service-line" key={item.id}>
  <div>{items.length>1&&<span>{item.name}</span>}<small>{item.destinationLabel}</small><Status tone={item.status==="entregado"?"gray":item.status==="listo"?"green":item.status==="preparando"?"orange":"blue"}>{item.statusLabel}</Status></div>
 </div>)}</div>;
}

export function OrderAccountActions({order,onAdd,busy=false}:{order:TableAccount;onAdd?:()=>void;busy?:boolean}){
 const{user,can}=useSession();const mutation=useServiceMutation();
 const manage=can("orders.manage")&&canManageOrderService(order,user?.id)&&!order.completedAt&&order.status!=="cancelado";
 if(order.channel!=="salon"||!manage)return null;
 return <>
  {onAdd&&canAddConsumption(order)&&<Button kind="secondary" icon="plus" disabled={busy||mutation.isPending} onClick={onAdd}>Agregar productos</Button>}
  {canCloseAccount(order)&&<Button icon="receipt" disabled={busy||mutation.isPending} aria-busy={mutation.isPending} onClick={()=>mutation.mutate({orderId:order.id})}>{mutation.isPending?"Cerrando…":"Cerrar cuenta"}</Button>}
 </>;
}

export function accountLabel(order:TableAccount){return order.completedAt?"Finalizada":order.accountState==="paid"?"Pagada":order.billClosedAt?"Por cobrar":"Cuenta abierta"}
