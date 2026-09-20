import {apiFetch} from "@/shared/api/client";
import type {ComboDetail,ComboList,ProductList} from "../domain/catalog-types";

export function listOrderCategories(){
 return apiFetch<{items:{id:string;name:string}[]}>("categories?pageSize=100");
}
export function listProductsByCategory(categoryId:string){
 const params=new URLSearchParams({status:"active",categoryId,page:"1",pageSize:"100"});
 return apiFetch<ProductList>(`products?${params.toString()}`);
}
export function listAllProducts(){
 return apiFetch<ProductList>("products?status=active&page=1&pageSize=100");
}
export function searchProducts(q:string){
 const params=new URLSearchParams({status:"active",q:q.trim(),page:"1",pageSize:"20"});
 return apiFetch<ProductList>(`products?${params.toString()}`);
}
export function listSalonProducts(input:{categoryId:string;q:string;page:number;pageSize:number}){
 const params=new URLSearchParams({status:"active",page:String(input.page),pageSize:String(input.pageSize)});
 if(input.categoryId)params.set("categoryId",input.categoryId);
 if(input.q.trim())params.set("q",input.q.trim());
 return apiFetch<ProductList>(`products?${params.toString()}`);
}
export function listOrderCombos(input:{q:string;page:number;pageSize:number}){
 const params=new URLSearchParams({page:String(input.page),pageSize:String(input.pageSize)});
 if(input.q.trim())params.set("q",input.q.trim());
 return apiFetch<ComboList>(`order-combos?${params.toString()}`);
}
export function getOrderCombo(id:string){
 return apiFetch<ComboDetail>(`order-combos/${id}`);
}
