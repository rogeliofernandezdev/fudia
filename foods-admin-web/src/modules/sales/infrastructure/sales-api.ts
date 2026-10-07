import {apiFetch} from "@/shared/api/client";
import type {SaleDetail,SalesResponse} from "../domain/types";

export function listSales(input:{q:string;page:number;pageSize:number}){
  const params=new URLSearchParams({q:input.q,paymentStatus:"paid",page:String(input.page),pageSize:String(input.pageSize)});
  return apiFetch<SalesResponse>(`pos/orders?${params.toString()}`);
}

export function getSaleDetail(id:string){
  return apiFetch<SaleDetail>(`pos/orders/${encodeURIComponent(id)}`);
}
