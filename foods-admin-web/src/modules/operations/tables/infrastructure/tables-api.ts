import {apiFetch} from "@/shared/api/client";
import type {List,RowDraft,Table,Zone,ZoneDraft} from "../domain/types";

export function listTables(input:{q:string;status:string;page:number;pageSize:number}){
 const params=new URLSearchParams({q:input.q,status:input.status,page:String(input.page),pageSize:String(input.pageSize)});
 return apiFetch<List<Table>>(`tables?${params.toString()}`);
}
export function listZones(page:number,pageSize:number){
 return apiFetch<List<Zone>>(`zones?page=${page}&pageSize=${pageSize}`);
}
export function listActiveZones(){
 return apiFetch<List<Zone>>("zones?status=active&page=1&pageSize=100");
}
export function createTables(items:RowDraft[]){
 return apiFetch<{items:Table[]}>("tables/batch",{method:"POST",body:JSON.stringify({items:items.map(row=>({name:row.name,seats:Number(row.seats)||2,zone:row.zone}))})});
}
export function deactivateTableOrZone(kind:"tables"|"zones",id:string){
 return apiFetch<void>(`${kind}/${id}`,{method:"DELETE"});
}
export function saveZone(draft:ZoneDraft){
 return apiFetch<Zone>(draft.id?`zones/${draft.id}`:"zones",{method:draft.id?"PATCH":"POST",body:JSON.stringify(draft)});
}
