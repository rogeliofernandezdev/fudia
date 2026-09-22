import {apiFetch} from "@/shared/api/client";
import type {Reservation,ReservationDraft,ReservationList,ReservationStatus} from "../domain/types";

export function listReservations(input:{q:string;status:string;page:number;pageSize:number}){
  const params=new URLSearchParams({q:input.q,status:input.status,page:String(input.page),pageSize:String(input.pageSize)});
  return apiFetch<ReservationList>(`reservations?${params.toString()}`);
}
function payload(draft:ReservationDraft){
  return {
    customerName:draft.customerName.trim(),
    customerPhone:draft.customerPhone.trim(),
    startsAt:new Date(draft.startsAt).toISOString(),
    guests:Number(draft.guests),
    tableId:draft.tableId||null,
    notes:draft.notes.trim(),
  };
}
export function saveReservation(draft:ReservationDraft){
  return apiFetch<Reservation>(draft.id?`reservations/${draft.id}`:"reservations",{method:draft.id?"PATCH":"POST",body:JSON.stringify(payload(draft))});
}
export function setReservationStatus(id:string,status:ReservationStatus){
  return apiFetch<void>(`reservations/${id}/status`,{method:"PATCH",body:JSON.stringify({status})});
}
