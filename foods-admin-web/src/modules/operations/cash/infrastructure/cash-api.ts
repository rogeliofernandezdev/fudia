import {apiFetch} from "@/shared/api/client";
import type {CashMovement,CashMovementDraft,CashRegister,CashRegisterDraft,CashShift,CashShiftList,CloseCashShiftDraft,OpenCashShiftDraft} from "../domain/types";

export function listCashRegisters(q=""){
  const params=new URLSearchParams();
  if(q)params.set("q",q);
  const suffix=params.toString()?`?${params.toString()}`:"";
  return apiFetch<{items:CashRegister[]}>(`cash-registers${suffix}`);
}

export function createCashRegister(draft:CashRegisterDraft){
  return apiFetch<CashRegister>("cash-registers",{
    method:"POST",
    body:JSON.stringify({name:draft.name.trim()}),
  });
}

export function listCashShifts(input:{q:string;status:string;page:number;pageSize:number}){
  const params=new URLSearchParams({q:input.q,status:input.status,page:String(input.page),pageSize:String(input.pageSize)});
  return apiFetch<CashShiftList>(`cash-shifts?${params.toString()}`);
}

export function getCashShift(id:string){
  return apiFetch<CashShift>(`cash-shifts/${id}`);
}

export function openCashShift(cashRegisterId:string,draft:OpenCashShiftDraft){
  return apiFetch<CashShift>("cash-shifts",{
    method:"POST",
    body:JSON.stringify({cashRegisterId,openingAmount:Number(draft.openingAmount),note:draft.note.trim()}),
  });
}

export function createCashMovement(shiftId:string,draft:CashMovementDraft){
  return apiFetch<CashMovement>(`cash-shifts/${shiftId}/movements`,{
    method:"POST",
    body:JSON.stringify({
      movementType:draft.movementType,
      amount:Number(draft.amount),
      reason:draft.reason.trim(),
      note:draft.note.trim(),
    }),
  });
}

export function closeCashShift(shiftId:string,draft:CloseCashShiftDraft){
  return apiFetch<CashShift>(`cash-shifts/${shiftId}/close`,{
    method:"POST",
    body:JSON.stringify({countedAmount:Number(draft.countedAmount),note:draft.note.trim()}),
  });
}
