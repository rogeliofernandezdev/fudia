import {apiFetch} from "@/shared/api/client";
import type {SalesResponse} from "../domain/types";

export function listSales(input:{q:string;page:number;pageSize:number}){
  const params=new URLSearchParams({q:input.q,paymentStatus:"paid",page:String(input.page),pageSize:String(input.pageSize)});
  return apiFetch<SalesResponse>(`pos/orders?${params.toString()}`);
}
