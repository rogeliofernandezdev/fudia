import {apiFetch} from "@/shared/api/client";
import type {AvailabilityResponse,CategoryOption} from "../domain/types";

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

export function updateAvailability(productId:string,payload:{status:"available"|"sold_out";dailyQuota:number|null;note:string}){
  return apiFetch<void>(`product-availability/${productId}`,{
    method:"PATCH",
    body:JSON.stringify(payload),
  });
}
