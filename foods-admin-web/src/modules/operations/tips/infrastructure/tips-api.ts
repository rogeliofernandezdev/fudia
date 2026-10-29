import {apiFetch} from "@/shared/api/client";
import type {TipPolicy,TipSettings,TipPayout,TipPayoutDraft,TipWorkspace} from "../domain/types";
const settingsPath=(scope:"organization"|"location",locationId?:string)=>`tip-settings?scope=${scope}${locationId?`&locationId=${encodeURIComponent(locationId)}`:""}`;
export const getTipSettings=(scope:"organization"|"location",locationId?:string)=>apiFetch<TipPolicy>(settingsPath(scope,locationId));
export const saveTipSettings=(scope:"organization"|"location",value:TipSettings&{inherit?:boolean},locationId?:string)=>apiFetch<TipPolicy>(settingsPath(scope,locationId),{method:"PUT",body:JSON.stringify(value)});
export const getTipWorkspace=(shiftId:string)=>apiFetch<TipWorkspace>(`cash-shifts/${shiftId}/tips`);
export const createTipPayout=(shiftId:string,draft:TipPayoutDraft)=>apiFetch<TipPayout>(`cash-shifts/${shiftId}/tip-payouts`,{method:"POST",body:JSON.stringify(draft)});
