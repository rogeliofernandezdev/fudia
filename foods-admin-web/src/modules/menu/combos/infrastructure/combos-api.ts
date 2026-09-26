import {apiFetch} from "@/shared/api/client";
import type {Combo,ComboDetail,Draft,Product} from "../domain/types";

export function listCombos(page:number,pageSize:number,q:string,status:string){
  const params=new URLSearchParams({page:String(page),pageSize:String(pageSize),q,status});
  return apiFetch<{items:Combo[];total:number;page:number;pageSize:number}>(`combos?${params.toString()}`);
}
export function listComboProducts(){
  return apiFetch<{items:Product[]}>("products?page=1&pageSize=100&status=active");
}
export function getCombo(id:string){return apiFetch<ComboDetail>(`combos/${id}`)}
export function saveCombo(value:Draft,editingId:string|null){
  return apiFetch<{id:string}>(editingId?`combos/${editingId}`:"combos",{
    method:editingId?"PATCH":"POST",
    body:JSON.stringify({
      ...value,
      availableFrom:value.availableFrom?new Date(value.availableFrom).toISOString():null,
      availableUntil:value.availableUntil?new Date(value.availableUntil).toISOString():null,
      availableDays:value.availableDays.length?value.availableDays:null,
      groups:value.groups.map(group=>({...group,options:group.options.map(option=>({...option,surcharge:option.surcharge.trim()||"0",quota:option.quota?Number(option.quota):null}))}))
    })
  });
}
export function setComboActive(id:string,active:boolean){
  return apiFetch<void>(`combos/${id}/status`,{method:"PATCH",body:JSON.stringify({active})});
}
