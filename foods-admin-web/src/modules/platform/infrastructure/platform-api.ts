import {apiFetch} from "@/shared/api/client";
import type {PlatformOnboardingContext,PlatformOnboardingDraft} from "../domain/types";

export async function getPlatformOnboardingContext():Promise<PlatformOnboardingContext>{
 const data=await apiFetch<Partial<PlatformOnboardingContext>>("settings");
 return {countryOptions:data.countryOptions??[],currencyOptions:data.currencyOptions??[]};
}
export async function createPlatformOrganization(draft:PlatformOnboardingDraft){
 const payload={
  ...draft,
  locationCode:draft.locationCode||draft.locationName.toUpperCase().replace(/[^A-Z0-9]/g,"").slice(0,8).padStart(3,"L"),
  taxRate:String(Number(draft.taxRate)/100),
  latitude:draft.latitude?Number(draft.latitude):null,
  longitude:draft.longitude?Number(draft.longitude):null,
 };
 const response=await fetch("/api/platform/organizations",{
  method:"POST",
  headers:{"Content-Type":"application/json"},
  body:JSON.stringify(payload),
 });
 const body=await response.json();
 if(!response.ok)throw new Error(body.message??"No pudimos registrar la empresa.");
 return body as {message:string};
}
