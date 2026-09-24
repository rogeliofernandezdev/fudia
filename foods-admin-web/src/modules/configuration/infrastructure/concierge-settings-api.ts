import {apiFetch} from "@/shared/api/client";
import type {ConciergeSettings,ConciergeSettingsDraft} from "../domain/concierge-settings";

export function getConciergeSettings(){
 return apiFetch<ConciergeSettings>("concierge-settings");
}

export function saveConciergeSettings(value:ConciergeSettingsDraft){
 return apiFetch<ConciergeSettings>("concierge-settings",{
  method:"PATCH",
  body:JSON.stringify(value),
 });
}
