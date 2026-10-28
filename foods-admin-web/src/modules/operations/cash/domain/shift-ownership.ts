import type {CashShift} from "./types";

/** Cierre personal del turno; ni otro cajero ni un administrador lo sustituyen. */
export function canCloseCashShift(shift:Pick<CashShift,"status"|"openedById">,userId:string|undefined,canManage:boolean){
  return canManage&&shift.status==="open"&&Boolean(userId)&&shift.openedById===userId;
}
