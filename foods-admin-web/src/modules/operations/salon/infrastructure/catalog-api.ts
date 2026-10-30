import {apiFetch} from "@/shared/api/client";
import type {ComboDetail,ComboList} from "../domain/catalog-types";

export function listOrderCategories(){
 return apiFetch<{items:{id:string;name:string}[]}>("categories?pageSize=100");
}
export async function listSalonProducts(input:{categoryId:string;q:string;page:number;pageSize:number;productIds?:string[]}){
 const params=new URLSearchParams({page:String(input.page),pageSize:String(input.pageSize),excludeCombos:"true"});
 if(input.categoryId)params.set("categoryId",input.categoryId);
 if(input.q.trim())params.set("q",input.q.trim());
 if(input.productIds?.length)params.set("productIds",input.productIds.join(","));
 const data=await apiFetch<{items:Array<{productId:string;name:string;price:string;categoryId:string|null;imageUrl:string|null;status:"available"|"low"|"sold_out"|"unavailable";remaining:number|null}>;total:number}>(`product-availability?${params.toString()}`);
 return{
  total:data.total,
  items:data.items.map(item=>({
   id:item.productId,
   name:item.name,
   price:item.price,
   categoryId:item.categoryId,
   imageUrl:item.imageUrl,
   availabilityStatus:item.status,
   remaining:item.remaining,
  })),
 };
}
/** Consulta acotada de los productos elegidos, independiente de filtros y páginas de la carta. */
export async function listSelectedProductAvailability(productIds:string[]){
 const batches=[];
 for(let index=0;index<productIds.length;index+=100)batches.push(productIds.slice(index,index+100));
 const pages=await Promise.all(batches.map(ids=>listSalonProducts({categoryId:"",q:"",page:1,pageSize:100,productIds:ids})));
 return {items:pages.flatMap(page=>page.items)};
}
export function listOrderCombos(input:{q:string;page:number;pageSize:number}){
 const params=new URLSearchParams({page:String(input.page),pageSize:String(input.pageSize)});
 if(input.q.trim())params.set("q",input.q.trim());
 return apiFetch<ComboList>(`order-combos?${params.toString()}`);
}
export function getOrderCombo(id:string){
 return apiFetch<ComboDetail>(`order-combos/${id}`);
}
