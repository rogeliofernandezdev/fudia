import {apiFetch} from "@/shared/api/client";
import type {TipPolicy,TipSettings,TipPayout,TipPayoutDraft,TipWorkspace} from "../domain/types";
export const getTipSettings=(scope:"organization"|"location")=>apiFetch<TipPolicy>(`tip-settings?scope=${scope}`);
export const saveTipSettings=(scope:"organization"|"location",value:TipSettings&{inherit?:boolean})=>apiFetch<TipPolicy>(`tip-settings?scope=${scope}`,{method:"PUT",body:JSON.stringify(value)});
export const getTipWorkspace=(shiftId:string)=>apiFetch<TipWorkspace>(`cash-shifts/${shiftId}/tips`);
export const createTipPayout=(shiftId:string,draft:TipPayoutDraft)=>apiFetch<TipPayout>(`cash-shifts/${shiftId}/tip-payouts`,{method:"POST",body:JSON.stringify(draft)});
