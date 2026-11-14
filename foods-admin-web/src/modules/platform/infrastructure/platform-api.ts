import {ApiClientError,apiFetch} from "@/shared/api/client";
import {expireBrowserSession} from "@/shared/session/expire-session";
import type {SpecialTaxRegime,SpecialTaxStatus} from "@/shared/fiscal/special-tax";
import type {ConciergeLimitOverride,ConciergeUsageDetail,ConciergeUsagePage,ConciergeUsageSettings,ConciergeUsageSettingsInput,Country,Currency,OrganizationSubscription,PlatformModule,PlatformOnboardingContext,PlatformOnboardingDraft,PlatformOrganizationDetail,PlatformOrganizationFilters,PlatformOrganizationPage,PlatformSubscriptionPlan,SubscriptionPlanDraft,PlatformWhatsAppChannel,PlatformWhatsAppChannelDraft,WhatsAppProviderOption,WhatsAppConnection,WhatsAppChannelHistory,DeliveryPartner,DeliveryPartnerDraft,OrganizationDelivery} from "../domain/types";
import {optionalLimit} from "../domain/concierge-usage";

async function platformFetch<T>(path:string,init?:RequestInit):Promise<T>{
 const response=await fetch(`/api/platform/${path}`,{
  ...init,
  headers:{"Content-Type":"application/json",...init?.headers},
 });
 if(response.status===401)expireBrowserSession();
 if(response.status===204)return undefined as T;
 const raw=await response.text();
 let body:({message?:string;code?:string;correlationId?:string}&Record<string,unknown>);
 try{body=raw?JSON.parse(raw):{}}catch{
  throw new ApiClientError(
   "No pudimos interpretar la respuesta del servidor.",
   "invalid_server_response",
   response.status,
   response.headers.get("X-Request-ID")??undefined,
  );
 }
 if(!response.ok)throw new ApiClientError(
  body.message??"No pudimos completar la operación.",
  body.code??"unknown_error",
  response.status,
  body.correlationId??response.headers.get("X-Request-ID")??undefined,
 );
 return body as T;
}

export async function listSubscriptionPlans():Promise<{items:PlatformSubscriptionPlan[]}>{
 return platformFetch<{items:PlatformSubscriptionPlan[]}>("plans");
}

export async function listReadyModules():Promise<PlatformModule[]>{
 const data=await apiFetch<{modules:PlatformModule[]}>("modules");
 return data.modules.filter(module=>module.availability==="ready");
}

export async function getPlatformOnboardingContext():Promise<PlatformOnboardingContext>{
 const[data,plans]=await Promise.all([
  platformFetch<Pick<PlatformOnboardingContext,"countryOptions"|"currencyOptions"|"specialTaxRegimeOptions"|"tipOptions"|"maxMenuPages">>("onboarding/catalogs"),
  listSubscriptionPlans(),
 ]);
 return {
  maxMenuPages:data.maxMenuPages,
  tipOptions:data.tipOptions,
  countryOptions:data.countryOptions??[],
  currencyOptions:data.currencyOptions??[],
  specialTaxRegimeOptions:data.specialTaxRegimeOptions??[],
  plans:plans.items.filter(plan=>plan.active&&plan.code!=="legacy"),
 };
}

export async function getPlatformGlobalCatalogs():Promise<{countryOptions:Country[];currencyOptions:Currency[]}>{
 return platformFetch<{countryOptions:Country[];currencyOptions:Currency[]}>("onboarding/catalogs?scope=all");
}

export async function saveSubscriptionPlan(draft:SubscriptionPlanDraft):Promise<PlatformSubscriptionPlan>{
 const payload={
  ...draft,
  trialDays:Number(draft.trialDays)||0,
  maxLocations:optionalLimit(draft.maxLocations),
  maxUsers:optionalLimit(draft.maxUsers),
  maxConciergeConversations:optionalLimit(draft.maxConciergeConversations),
 };
 return platformFetch<PlatformSubscriptionPlan>(draft.id?`plans/${draft.id}`:"plans",{
  method:draft.id?"PATCH":"POST",
  body:JSON.stringify(payload),
 });
}

export async function createPlatformOrganization(draft:PlatformOnboardingDraft){
 const payload={
  ...draft,
  districtCode:draft.country==="PE"?draft.districtCode:"",
  locationCode:draft.locationCode||draft.locationName.toUpperCase().replace(/[^A-Z0-9]/g,"").slice(0,8).padStart(3,"L"),
  taxRate:String(Number(draft.taxRate)/100),
  specialTaxRegime:draft.specialTaxRegime||null,
  latitude:draft.latitude?Number(draft.latitude):null,
  longitude:draft.longitude?Number(draft.longitude):null,
 };
 return platformFetch<{organizationId:string;fiscalProfileId:string;locationId:string;administratorId:string}>("organizations",{
  method:"POST",
  body:JSON.stringify(payload),
 });
}

export async function listPlatformOrganizations(filters:PlatformOrganizationFilters):Promise<PlatformOrganizationPage>{
 const params=new URLSearchParams({page:String(filters.page),pageSize:String(filters.pageSize)});
 if(filters.q.trim())params.set("q",filters.q.trim());
 if(filters.planId)params.set("planId",filters.planId);
 if(filters.standing)params.set("standing",filters.standing);
 return platformFetch<PlatformOrganizationPage>(`organizations?${params}`);
}

export async function getPlatformOrganization(organizationId:string):Promise<PlatformOrganizationDetail>{
 return platformFetch<PlatformOrganizationDetail>(`organizations/${encodeURIComponent(organizationId)}`);
}

export async function getCurrentOrganizationSubscription(){
 return apiFetch<OrganizationSubscription>("subscription");
}

/** Sin `organizationId` opera sobre la empresa activa de la sesión. */
const subscriptionPath=(organizationId?:string)=>organizationId?`organizations/${encodeURIComponent(organizationId)}/subscription`:"subscription";

export async function getOrganizationSubscription(organizationId:string){
 return platformFetch<OrganizationSubscription>(subscriptionPath(organizationId));
}

export async function changeOrganizationSubscription(input:{planId:string;billingCycle:"monthly"|"annual";status:OrganizationSubscription["status"];autoRenew:boolean;termsAccepted:boolean},organizationId?:string){
 return platformFetch<OrganizationSubscription>(subscriptionPath(organizationId),{method:"PATCH",body:JSON.stringify(input)});
}

export async function recordSubscriptionPayment(input:{amount:string;currency:string;status:"pending"|"paid"|"failed"|"refunded";provider:string;externalReference:string;paidAt:string},organizationId?:string){
 return platformFetch<{id:string}>(`${subscriptionPath(organizationId)}/payments`,{method:"POST",body:JSON.stringify(input)});
}

export async function listPlatformWhatsAppChannels(){return platformFetch<{items:PlatformWhatsAppChannel[];providerOptions:WhatsAppProviderOption[]}>("whatsapp-channels",{cache:"no-store"});}
export async function listDeliveryPartners(){return platformFetch<{items:DeliveryPartner[]}>("delivery-partners");}
export async function saveDeliveryPartner(draft:DeliveryPartnerDraft,id?:string){return platformFetch<DeliveryPartner>(id?`delivery-partners/${encodeURIComponent(id)}`:"delivery-partners",{method:id?"PATCH":"POST",body:JSON.stringify({name:draft.name,whatsappPhone:draft.whatsappPhone,active:draft.active,coverageCodes:draft.coverageCodes})});}
/** Reemplaza el enlace privado del panel de la central; el anterior deja de abrir. */
export async function rotateDeliveryPartnerPanel(id:string){return platformFetch<{partnerId:string;panelToken:string}>(`delivery-partners/${encodeURIComponent(id)}/panel-link`,{method:"POST"});}
export async function reorderDeliveryPartners(partnerIds:string[]){return platformFetch<{items:DeliveryPartner[]}>("delivery-partners/order",{method:"PUT",body:JSON.stringify({partnerIds})});}
export async function getOrganizationDelivery(organizationId:string){return platformFetch<OrganizationDelivery>(`organizations/${encodeURIComponent(organizationId)}/delivery`);}
export async function setOrganizationDelivery(organizationId:string,enabled:boolean){return platformFetch<OrganizationDelivery>(`organizations/${encodeURIComponent(organizationId)}/delivery`,{method:"PUT",body:JSON.stringify({enabled})});}
export async function savePlatformWhatsAppChannel(draft:PlatformWhatsAppChannelDraft,id?:string){return platformFetch<PlatformWhatsAppChannel>(id?`whatsapp-channels/${id}`:"whatsapp-channels",{method:id?"PATCH":"POST",body:JSON.stringify(draft)});}

/** Uso de Fudia Concierge: solo Plataforma. Las empresas nunca consultan su tope ni su consumo. */
export async function getConciergeUsageSettings(){return platformFetch<ConciergeUsageSettings>("concierge-usage/settings");}
export async function updateConciergeUsageSettings(input:ConciergeUsageSettingsInput){return platformFetch<ConciergeUsageSettings>("concierge-usage/settings",{method:"PATCH",body:JSON.stringify(input)});}

/** El contrato admite hasta 100 empresas por consulta; `summary` siempre cuenta toda la cartera. */
export async function listConciergeUsage(organizationIds:string[]){
 const params=new URLSearchParams();
 organizationIds.slice(0,100).forEach(id=>params.append("organizationId",id));
 const query=params.toString();
 return platformFetch<ConciergeUsagePage>(query?`concierge-usage?${query}`:"concierge-usage");
}

const conciergeUsagePath=(organizationId:string)=>`organizations/${encodeURIComponent(organizationId)}/concierge-usage`;
export async function getOrganizationConciergeUsage(organizationId:string){return platformFetch<ConciergeUsageDetail>(conciergeUsagePath(organizationId));}
export async function updateOrganizationConciergeLimit(organizationId:string,input:ConciergeLimitOverride){return platformFetch<ConciergeUsageDetail>(conciergeUsagePath(organizationId),{method:"PATCH",body:JSON.stringify(input)});}
export type SpecialTaxRegimesView={items:(SpecialTaxRegime&{enrolledProfiles:number;status:SpecialTaxStatus})[];today:string;warnDays:number};
export type SpecialTaxRateInput={validFrom:string;validTo:string|null;rate:string;legalBasis:string};
export async function listSpecialTaxRegimes(){return platformFetch<SpecialTaxRegimesView>("special-tax-regimes");}
export async function saveSpecialTaxRate(regime:string,input:SpecialTaxRateInput,id?:string){return platformFetch<void>(id?`special-tax-rates/${encodeURIComponent(id)}`:`special-tax-regimes/${encodeURIComponent(regime)}/rates`,{method:id?"PATCH":"POST",body:JSON.stringify(input)});}
export async function deleteSpecialTaxRate(id:string){return platformFetch<void>(`special-tax-rates/${encodeURIComponent(id)}`,{method:"DELETE"});}

export async function linkPlatformWhatsApp(countryCode:string,sessionRef?:string){return platformFetch<WhatsAppConnection>("whatsapp-channels/link",{method:"POST",cache:"no-store",body:JSON.stringify({countryCode,sessionRef})});}
export async function listWhatsAppChannelHistory(id:string){return platformFetch<{items:WhatsAppChannelHistory[]}>(`whatsapp-channels/${encodeURIComponent(id)}/history`,{cache:"no-store"});}
