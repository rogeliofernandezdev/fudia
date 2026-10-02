import {apiFetch} from "@/shared/api/client";
import type {CashMovement,CashMovementDraft,CashOperation,CashOperationDraft,CashRegister,CashRegisterDraft,CashShift,CashShiftList,CashShiftUser,CashUserOption,CloseCashShiftDraft,OpenCashShiftDraft} from "../domain/types";

export function getCurrentCashShift(){
  return apiFetch<{shift:CashShift|null}>("cash-shifts/current");
}

export function listCashRegisters(q=""){
  const params=new URLSearchParams();
  if(q)params.set("q",q);
  const suffix=params.toString()?`?${params.toString()}`:"";
  return apiFetch<{items:CashRegister[]}>(`cash-registers${suffix}`);
}

export function createCashRegister(draft:CashRegisterDraft){
  return apiFetch<CashRegister>("cash-registers",{
    method:"POST",
    body:JSON.stringify({name:draft.name.trim(),blindClose:draft.blindClose}),
  });
}

export function updateCashRegister(id:string,draft:CashRegisterDraft){
  return apiFetch<CashRegister>(`cash-registers/${id}`,{
    method:"PATCH",
    body:JSON.stringify({name:draft.name.trim(),blindClose:draft.blindClose}),
  });
}

export function setCashRegisterActive(id:string,active:boolean){
  return apiFetch<void>(`cash-registers/${id}/status`,{
    method:"PATCH",
    body:JSON.stringify({active}),
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

export function listCashUserOptions(){
  return apiFetch<{items:CashUserOption[]}>("cash-users/options");
}

export function listCashShiftUsers(shiftId:string){
  return apiFetch<{items:CashShiftUser[]}>(`cash-shifts/${shiftId}/users`);
}

export function assignCashShiftUser(shiftId:string,userId:string){
  return apiFetch<CashShiftUser>(`cash-shifts/${shiftId}/users`,{
    method:"POST",
    body:JSON.stringify({userId}),
  });
}

export function unassignCashShiftUser(shiftId:string,userId:string){
  return apiFetch<void>(`cash-shifts/${shiftId}/users/${userId}`,{method:"DELETE"});
}

export function createCashOperation(shiftId:string,draft:CashOperationDraft){
  return apiFetch<CashOperation>(`cash-shifts/${shiftId}/operations`,{
    method:"POST",
    body:JSON.stringify({
      operationType:draft.operationType,
      targetShiftId:draft.targetShiftId,
      amount:Number(draft.amount),
      reason:draft.reason.trim(),
      note:draft.note.trim(),
    }),
  });
}

export function closeCashShift(shiftId:string,draft:CloseCashShiftDraft){
  return apiFetch<CashShift>(`cash-shifts/${shiftId}/close`,{
    method:"POST",
    body:JSON.stringify({
      countedAmount:draft.counts.some(line=>Number(line.quantity)>0)?undefined:Number(draft.countedAmount),
      counts:draft.counts.filter(line=>Number(line.denomination)>0&&Number(line.quantity)>0).map(line=>({
        denomination:Number(line.denomination),
        quantity:Number(line.quantity),
      })),
      note:draft.note.trim(),
    }),
  });
}
