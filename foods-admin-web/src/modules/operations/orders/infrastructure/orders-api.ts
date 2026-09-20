import {apiFetch} from "@/shared/api/client";
import type {Order,OrdersResponse} from "../domain/types";

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
