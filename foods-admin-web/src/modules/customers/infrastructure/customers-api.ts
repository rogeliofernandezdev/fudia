import {apiFetch} from "@/shared/api/client";
import type {Customer,CustomerDraft,CustomersResponse} from "../domain/types";

export type CustomersQuery={
  q:string;
  status:string;
  segment:string;
  page:number;
  pageSize:number;
};

export function listCustomers(query:CustomersQuery){
  const params=new URLSearchParams({
    q:query.q,
    status:query.status,
    segment:query.segment,
    page:String(query.page),
    pageSize:String(query.pageSize),
  });
  return apiFetch<CustomersResponse>(`customers?${params.toString()}`);
}

export function getCustomer(id:string){
  return apiFetch<Customer>(`customers/${id}`);
}

export function saveCustomer(draft:CustomerDraft){
  return apiFetch<Customer>(draft.id?`customers/${draft.id}`:"customers",{
    method:draft.id?"PATCH":"POST",
    body:JSON.stringify(draft),
  });
}

export function setCustomerActive(id:string,active:boolean){
  return apiFetch<void>(`customers/${id}/status`,{
    method:"PATCH",
    body:JSON.stringify({active}),
  });
}
