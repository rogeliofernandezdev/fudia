import type {CashShift} from "./types";

// Each new shift belongs to its opener; team changes do not transfer responsibility.
export function cashShiftAttribution(shift:Pick<CashShift,"status"|"openedByName"|"closedByName">){
  if(shift.status==="closed"){
    return {label:"Cerrado por",name:shift.closedByName||"Sin registro de cierre"};
  }
  return {label:"Cajero",name:shift.openedByName||"Sin responsable registrado"};
}

export function cashShiftTeamName(shift:Pick<CashShift,"activeUserNames">){
  return shift.activeUserNames?.join(", ")||(shift.activeUserNames?"Sin usuarios asignados":"Equipo no disponible");
}
