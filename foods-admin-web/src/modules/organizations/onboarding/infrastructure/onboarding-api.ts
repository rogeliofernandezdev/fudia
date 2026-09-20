import {listProfiles} from "../../infrastructure/organizations-api";
import type {ConfigurationOnboardingDraft,OnboardingCatalogs} from "../domain/types";

export async function loadOnboardingCatalogs():Promise<OnboardingCatalogs>{
 const response=await listProfiles(1,1);
 return {countryOptions:response.countryOptions,currencyOptions:response.currencyOptions};
}
export async function createOrganizationFromOnboarding(value:ConfigurationOnboardingDraft){
 const response=await fetch("/api/platform/organizations",{
  method:"POST",
  headers:{"Content-Type":"application/json"},
  body:JSON.stringify(value),
 });
 const body=await response.json();
 if(!response.ok)throw new Error(body.message??"No pudimos registrar la empresa.");
 return body;
}
