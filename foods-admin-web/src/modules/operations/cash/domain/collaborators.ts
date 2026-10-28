import type {CashShift,CashShiftUser,CashUserOption} from "./types";

/** La API excluye administradores, usuarios inactivos y ajenos al local. */
export function availableCashCollaborators(options:CashUserOption[],users:CashShiftUser[],currentUserId:string,status:CashShift["status"]){
  if(status!=="open")return [];
  const assigned=new Set(users.map(user=>user.userId));
  return options.filter(option=>option.id!==currentUserId&&!option.assignedShiftId&&!assigned.has(option.id));
}
