import {apiFetch} from "@/shared/api/client";
import type {AvailabilityResponse,AvailabilityHistoryResponse,CategoryOption} from "../domain/types";

export type AvailabilityQuery={
  search:string;
  categoryId:string;
  page:number;
  pageSize:number;
};

export function listAvailability(query:AvailabilityQuery){
  const params=new URLSearchParams({
    q:query.search,
    categoryId:query.categoryId,
    page:String(query.page),
    pageSize:String(query.pageSize),
  });
  return apiFetch<AvailabilityResponse>(`product-availability?${params.toString()}`);
}

export function listAvailabilityCategories(){
  return apiFetch<{items:CategoryOption[]}>("categories?page=1&pageSize=100");
}

export function listAvailabilityHistory(productId:string,page:number,pageSize:number){
 const params=new URLSearchParams({page:String(page),pageSize:String(pageSize)});
 return apiFetch<AvailabilityHistoryResponse>(`product-availability/${productId}/history?${params.toString()}`);
}

export function updateAvailability(productId:string,payload:{status:"available"|"sold_out";portionQuantity:number|null;note:string;reason:string}){
  return apiFetch<void>(`product-availability/${productId}`,{
    method:"PATCH",
    body:JSON.stringify(payload),
  });
}
