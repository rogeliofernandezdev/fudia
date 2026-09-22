import {apiFetch} from "@/shared/api/client";
import type {OrganizationSubscription,PlatformModule,PlatformOnboardingContext,PlatformOnboardingDraft,SubscriptionPlan,SubscriptionPlanDraft} from "../domain/types";

async function platformFetch<T>(path:string,init?:RequestInit):Promise<T>{
 const response=await fetch(`/api/platform/${path}`,{
  ...init,
  headers:{"Content-Type":"application/json",...init?.headers},
 });
 if(response.status===204)return undefined as T;
 const body=await response.json().catch(()=>({}));
 if(!response.ok)throw new Error(body.message??"No pudimos completar la operación.");
 return body as T;
}

export async function listSubscriptionPlans():Promise<{items:SubscriptionPlan[]}>{
 return platformFetch<{items:SubscriptionPlan[]}>("plans");
}

export async function listReadyModules():Promise<PlatformModule[]>{
 const data=await apiFetch<{modules:PlatformModule[]}>("modules");
 return data.modules.filter(module=>module.availability==="ready");
}

export async function getPlatformOnboardingContext():Promise<PlatformOnboardingContext>{
 const[data,plans]=await Promise.all([
  apiFetch<Partial<PlatformOnboardingContext>>("settings"),
  listSubscriptionPlans(),
 ]);
 return {
  countryOptions:data.countryOptions??[],
  currencyOptions:data.currencyOptions??[],
  plans:plans.items.filter(plan=>plan.active&&plan.code!=="legacy"),
 };
}

export async function saveSubscriptionPlan(draft:SubscriptionPlanDraft):Promise<SubscriptionPlan>{
 const payload={
  ...draft,
  trialDays:Number(draft.trialDays)||0,
  maxLocations:draft.maxLocations.trim()?Number(draft.maxLocations):null,
  maxUsers:draft.maxUsers.trim()?Number(draft.maxUsers):null,
 };
 return platformFetch<SubscriptionPlan>(draft.id?`plans/${draft.id}`:"plans",{
  method:draft.id?"PATCH":"POST",
  body:JSON.stringify(payload),
 });
}

export async function createPlatformOrganization(draft:PlatformOnboardingDraft){
 const payload={
  ...draft,
  locationCode:draft.locationCode||draft.locationName.toUpperCase().replace(/[^A-Z0-9]/g,"").slice(0,8).padStart(3,"L"),
  taxRate:String(Number(draft.taxRate)/100),
  latitude:draft.latitude?Number(draft.latitude):null,
  longitude:draft.longitude?Number(draft.longitude):null,
 };
 return platformFetch<{organizationId:string;fiscalProfileId:string;locationId:string;administratorId:string}>("organizations",{
  method:"POST",
  body:JSON.stringify(payload),
 });
}

export async function getCurrentOrganizationSubscription(){
 return apiFetch<OrganizationSubscription>("subscription");
}

export async function changeOrganizationSubscription(input:{planId:string;billingCycle:"monthly"|"annual";status:OrganizationSubscription["status"];autoRenew:boolean;termsAccepted:boolean}){
 return platformFetch<OrganizationSubscription>("subscription",{method:"PATCH",body:JSON.stringify(input)});
}

export async function recordSubscriptionPayment(input:{amount:string;currency:string;status:"pending"|"paid"|"failed"|"refunded";provider:string;externalReference:string;paidAt:string}){
 return platformFetch<{id:string}>("subscription/payments",{method:"POST",body:JSON.stringify(input)});
}
