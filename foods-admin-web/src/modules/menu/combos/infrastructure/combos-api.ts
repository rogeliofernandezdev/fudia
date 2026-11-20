import {apiFetch} from "@/shared/api/client";
import type {Combo,ComboDetail,Draft,Product} from "../domain/types";
import {zonedInputToIso} from "../domain/sale-window";

export function listCombos(page:number,pageSize:number,q:string,status:string){
  const params=new URLSearchParams({page:String(page),pageSize:String(pageSize),q,status});
  return apiFetch<{items:Combo[];total:number;page:number;pageSize:number}>(`combos?${params.toString()}`);
}
export async function listComboProducts(){
  const pageOf=(page:number)=>apiFetch<{items:Product[];total:number}>(`products?page=${page}&pageSize=100&status=active`);
  const first=await pageOf(1);
  const items:Product[]=[...first.items];
  if(!first.items.length||items.length>=first.total)return {items};
  if(Number.isFinite(first.total)){
    // Con el total conocido, las páginas restantes se piden a la vez en vez de una
    // tras otra (cada petición cruza el BFF y el API).
    const rest=await Promise.all(Array.from({length:Math.ceil(first.total/100)-1},(_,index)=>pageOf(index+2)));
    for(const result of rest)items.push(...result.items);
    return {items};
  }
  for(let page=2;;page++){
    const result=await pageOf(page);
    items.push(...result.items);
    if(!result.items.length||result.items.length<100)return {items};
  }
}
/** Solo las opciones elegidas, para refrescar sus cantidades sin volver a descargar toda la carta (lotes de 100). */
export async function listComboProductsByIds(ids:string[]){
  const batches:string[][]=[];
  for(let index=0;index<ids.length;index+=100)batches.push(ids.slice(index,index+100));
  const pages=await Promise.all(batches.map(batch=>apiFetch<{items:Product[]}>(`products?${new URLSearchParams({page:"1",pageSize:"100",status:"active",productIds:batch.join(",")}).toString()}`)));
  return {ids,items:pages.flatMap(page=>page.items)};
}
export function getCombo(id:string){return apiFetch<ComboDetail>(`combos/${id}`)}
/** `timeZone` es la zona del local activo: las fechas de venta se interpretan en su hora, no en la del navegador. */
export function saveCombo(value:Draft,editingId:string|null,timeZone?:string|null){
  return apiFetch<{id:string}>(editingId?`combos/${editingId}`:"combos",{
    method:editingId?"PATCH":"POST",
    body:JSON.stringify({
      ...value,
      availableFrom:zonedInputToIso(value.availableFrom,timeZone),
      availableUntil:zonedInputToIso(value.availableUntil,timeZone),
      availableDays:value.availableDays.length?value.availableDays:null,
      groups:value.groups.map(group=>({...group,options:group.options.map(option=>({...option,surcharge:option.surcharge.trim()||"0",quota:option.quota?Number(option.quota):null}))}))
    })
  });
}
export function setComboActive(id:string,active:boolean){
  return apiFetch<void>(`combos/${id}/status`,{method:"PATCH",body:JSON.stringify({active})});
}
