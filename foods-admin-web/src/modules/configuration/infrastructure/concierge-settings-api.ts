import {apiFetch} from "@/shared/api/client";
import type {ConciergeSettings} from "../domain/concierge-settings";

export function getConciergeSettings(){
 return apiFetch<ConciergeSettings>("concierge-settings");
}
