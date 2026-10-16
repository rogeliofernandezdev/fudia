import {apiFetch} from "@/shared/api/client";
import type {ConciergeSettings} from "../domain/concierge-settings";
import type {DeliveryFees,DeliveryFeesPayload} from "../domain/delivery-fees";

export function getConciergeSettings(){
 return apiFetch<ConciergeSettings>("concierge-settings");
}

export function getDeliveryFees(){
 return apiFetch<DeliveryFees>("concierge-settings/delivery-fees");
}

export function saveDeliveryFees(payload:DeliveryFeesPayload){
 return apiFetch<DeliveryFees>("concierge-settings/delivery-fees",{method:"PUT",body:JSON.stringify(payload)});
}
