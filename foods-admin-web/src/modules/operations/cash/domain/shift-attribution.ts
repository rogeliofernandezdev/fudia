import type {CashShift} from "./types";

// Opening is an audit event, never a fallback for the current team or closer.
export function cashShiftAttribution(shift:Pick<CashShift,"status"|"closedByName"|"activeUserNames">){
  if(shift.status==="closed"){
    return {label:"Cerrado por",name:shift.closedByName||"Sin registro de cierre"};
  }
  return {
    label:"Equipo actual",
    name:shift.activeUserNames?.join(", ")||(shift.activeUserNames?"Sin usuarios asignados":"Equipo no disponible"),
  };
}
