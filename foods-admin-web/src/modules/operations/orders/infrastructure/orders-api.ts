import {apiFetch} from "@/shared/api/client";
import type {DeliveryQuote,Order,OrdersResponse} from "../domain/types";
import {manualOrderPayload,type ManualOrderDraft} from "../domain/manual-order-schema";

export function createManualOrder(draft:ManualOrderDraft){
  return apiFetch<Order>("orders",{method:"POST",body:JSON.stringify(manualOrderPayload(draft))});
}

/** El punto viaja en el cuerpo, nunca en la URL. */
export function quoteDelivery(point:{latitude:number;longitude:number}){
  return apiFetch<DeliveryQuote>("delivery-quote",{method:"POST",body:JSON.stringify(point)});
}

export type OrdersQuery={
  q:string;
  channel:string;
  status:string;
  page:number;
  pageSize:number;
};

export function listOrders(query:OrdersQuery){
  const params=new URLSearchParams({
    q:query.q,
    channel:query.channel,
    status:query.status,
    page:String(query.page),
    pageSize:String(query.pageSize),
  });
  return apiFetch<OrdersResponse>(`orders?${params.toString()}`);
}

export function getOrder(id:string){
  return apiFetch<Order>(`orders/${id}`);
}

export function updateOrderStatus(id:string,status:string){
  return apiFetch<Order>(`orders/${id}/status`,{
    method:"PATCH",
    body:JSON.stringify({status}),
  });
}
