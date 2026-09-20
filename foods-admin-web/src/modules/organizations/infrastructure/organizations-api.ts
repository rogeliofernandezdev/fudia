import {apiFetch} from "@/shared/api/client";
import type {Currency,List,Location,LocationDraft,Organization,ProfileDraft,ProfilesResponse,Rate,RateDraft} from "../domain/types";

export function getOrganization(){return apiFetch<Organization>("organization")}
export function updateOrganization(value:Organization){
 return apiFetch<Organization>("organization",{method:"PATCH",body:JSON.stringify(value)});
}
export function listLocations(page:number,pageSize:number){
 return apiFetch<List<Location>>(`locations?page=${page}&pageSize=${pageSize}`);
}
export function listProfiles(page:number,pageSize:number){
 return apiFetch<ProfilesResponse>(`fiscal-profiles?page=${page}&pageSize=${pageSize}`);
}
export function saveLocation(value:LocationDraft){
 const payload={...value,code:value.code||value.name.toUpperCase().replace(/[^A-Z0-9]/g,"").slice(0,8).padStart(3,"L")};
 return apiFetch<Location|void>(value.id?`locations/${value.id}`:"locations",{method:value.id?"PATCH":"POST",body:JSON.stringify(payload)});
}
export function deactivateLocation(id:string){return apiFetch<void>(`locations/${id}`,{method:"DELETE"})}
export function listRates(page:number,pageSize:number){
 return apiFetch<List<Rate>&{currencyOptions:Currency[]}>(`exchange-rates?page=${page}&pageSize=${pageSize}`);
}
export function saveProfile(value:ProfileDraft){
 return apiFetch<void>(value.id?`fiscal-profiles/${value.id}`:"fiscal-profiles",{method:value.id?"PATCH":"POST",body:JSON.stringify({...value,taxRate:String(Number(value.taxPercent)/100)})});
}
export function deactivateProfile(id:string){return apiFetch<void>(`fiscal-profiles/${id}`,{method:"DELETE"})}
export function createRate(value:RateDraft){
 return apiFetch<Rate>("exchange-rates",{method:"POST",body:JSON.stringify({...value,effectiveAt:new Date(value.effectiveAt).toISOString(),providerReference:value.providerReference||null})});
}
export function deleteRate(id:string){return apiFetch<void>(`exchange-rates/${id}`,{method:"DELETE"})}
