import {apiFetch} from "@/shared/api/client";
import type {PaymentMethod,PaymentMethodDraft,PaymentMethodsResponse} from "../domain/payment-method";

export function listPaymentMethods(input:{q?:string;status?:string;page?:number;pageSize?:number}={}){
 const params=new URLSearchParams({
  q:input.q??"",
  status:input.status??"",
  page:String(input.page??1),
  pageSize:String(input.pageSize??20),
 });
 return apiFetch<PaymentMethodsResponse>("payment-methods?"+params.toString());
}

export function savePaymentMethod(value:PaymentMethodDraft,editingCode?:string){
 const path=editingCode?"payment-methods/"+encodeURIComponent(editingCode):"payment-methods";
 const payload=editingCode?{...value,code:undefined}:value;
 return apiFetch<PaymentMethod>(path,{method:editingCode?"PATCH":"POST",body:JSON.stringify(payload)});
}

export function setPaymentMethodActive(code:string,active:boolean){
 return apiFetch<void>("payment-methods/"+encodeURIComponent(code)+"/status",{method:"PATCH",body:JSON.stringify({active})});
}
