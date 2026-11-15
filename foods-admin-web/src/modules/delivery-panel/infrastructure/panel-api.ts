import type {CourierAction,CourierAssignment,CourierPosition,CourierView,PanelAction,PanelActionResult,PartnerPanel,PartnerSession} from "../domain/types";

// El panel de la central exige la sesión del operador: viaja en una cookie
// HttpOnly que el BFF del mismo origen reenvía a foods-backend. El enlace del
// motorizado no tiene sesión: su enlace privado va en la ruta del BFF público.
const partnerBase="/api/delivery-partner";
const courierBase=(token:string)=>`/api/public/delivery-courier/${encodeURIComponent(token)}`;

/** Error con el estado y el código del backend: 401 pide iniciar sesión y `password_change_required`, cambiar la contraseña. */
export class PanelRequestError extends Error{
  constructor(message:string,readonly status:number,readonly code:string){super(message)}
}

async function call<T>(url:string,init?:RequestInit):Promise<T>{
  const response=await fetch(url,{cache:"no-store",...init});
  if(response.status===204)return undefined as T;
  const data=await response.json().catch(()=>null) as {message?:string;code?:string}|null;
  if(!response.ok)throw new PanelRequestError(data?.message??"No pudimos conectar con el servicio de reparto.",response.status,data?.code??"");
  return data as T;
}
const post=(body:unknown):RequestInit=>({method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(body)});

/** La sesión venció o no existe: hay que volver a iniciar sesión. */
export const needsLogin=(error:unknown)=>error instanceof PanelRequestError&&error.status===401;
/** El operador aún usa su contraseña temporal. */
export const needsPasswordChange=(error:unknown)=>error instanceof PanelRequestError&&error.code==="password_change_required";

export const loginPartner=(email:string,password:string)=>call<PartnerSession>(`${partnerBase}/auth/login`,post({email,password}));
export const logoutPartner=()=>call<void>(`${partnerBase}/auth/logout`,{method:"POST"});
export const loadPartnerSession=()=>call<PartnerSession>(`${partnerBase}/session`);
export const changePartnerPassword=(currentPassword:string,newPassword:string)=>call<PartnerSession>(`${partnerBase}/password`,post({currentPassword,newPassword}));

export const loadPartnerPanel=()=>call<PartnerPanel>(`${partnerBase}/panel`);
export const sendPanelAction=(action:PanelAction,offer:string)=>call<PanelActionResult>(`${partnerBase}/panel/actions`,post({action,token:offer}));
/** Registra o cambia al motorizado: cambiar el nombre invalida el enlace anterior. */
export const assignCourier=(offer:string,courierName:string)=>call<CourierAssignment>(`${partnerBase}/panel/courier`,post({token:offer,courierName}));

export const loadCourierView=(courierToken:string)=>call<CourierView>(courierBase(courierToken));
export const sendCourierAction=(courierToken:string,action:CourierAction)=>call<PanelActionResult>(`${courierBase(courierToken)}/actions`,post({action}));
export const sendCourierLocation=(courierToken:string,position:CourierPosition)=>call<void>(`${courierBase(courierToken)}/location`,post(position));
