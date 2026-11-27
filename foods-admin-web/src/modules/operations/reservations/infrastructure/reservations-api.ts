import {apiFetch} from "@/shared/api/client";
import {reservationStart} from "../domain/reservation-schema";
import type {Reservation,ReservationDepositDecision,ReservationDraft,ReservationList,ReservationStatus} from "../domain/types";

export function listReservations(input:{q:string;status:string;page:number;pageSize:number}){
  const params=new URLSearchParams({q:input.q,status:input.status,page:String(input.page),pageSize:String(input.pageSize)});
  return apiFetch<ReservationList>(`reservations?${params.toString()}`);
}
function payload(draft:ReservationDraft){
  return {
    customerName:draft.customerName.trim(),
    customerPhone:draft.customerPhone.trim(),
    startsAt:reservationStart(draft.date,draft.time).toISOString(),
    guests:Number(draft.guests),
    durationMinutes:Number(draft.durationMinutes),
    toleranceMinutes:Number(draft.toleranceMinutes),
    depositAmount:draft.depositEnabled?draft.depositAmount.trim():"0",
    tableId:draft.tableId||null,
    notes:draft.notes.trim(),
  };
}
export function saveReservation(draft:ReservationDraft){
  return apiFetch<Reservation>(draft.id?`reservations/${draft.id}`:"reservations",{method:draft.id?"PATCH":"POST",body:JSON.stringify(payload(draft))});
}
/** Cancelar con la garantía cobrada exige decidir si se devuelve o se retiene. */
/** `tableId` es la mesa donde se ubica al cliente al pasar a «En mesa»; `deposit`, qué hacer con la garantía al cancelar. */
export function setReservationStatus(id:string,status:ReservationStatus,options:{deposit?:ReservationDepositDecision;tableId?:string}={}){
  return apiFetch<void>(`reservations/${id}/status`,{method:"PATCH",body:JSON.stringify({status,...(options.deposit?{deposit:options.deposit}:{}),...(options.tableId?{tableId:options.tableId}:{})})});
}
export type DepositChargeDraft={method:string;reference:string;shiftId?:string};
/** Cobra la garantía en el turno de caja: entra al turno con su medio de pago. */
export function chargeReservationDeposit(id:string,draft:DepositChargeDraft){
  return apiFetch<{paymentId:string;amount:string;methodName:string}>(`reservations/${id}/deposit`,{method:"POST",body:JSON.stringify({method:draft.method,reference:draft.reference.trim(),...(draft.shiftId?{shiftId:draft.shiftId}:{})})});
}
/** Devuelve la garantía cobrada desde el turno de caja del usuario. */
export function refundReservationDeposit(id:string){
  return apiFetch<void>(`reservations/${id}/deposit/refund`,{method:"POST"});
}
