import type {CourierAction,CourierAssignment,CourierPosition,CourierView,PanelAction,PanelActionResult,PartnerPanel} from "../domain/types";

// El panel y el enlace del motorizado no tienen sesión: el enlace privado viaja
// en la ruta del BFF público del mismo origen, que lo reenvía a foods-backend.
const panelBase=(token:string)=>`/api/public/delivery-panel/${encodeURIComponent(token)}`;
const courierBase=(token:string)=>`/api/public/delivery-courier/${encodeURIComponent(token)}`;

async function call<T>(url:string,init?:RequestInit):Promise<T>{
  const response=await fetch(url,{cache:"no-store",...init});
  if(response.status===204)return undefined as T;
  const data=await response.json().catch(()=>null) as {message?:string}|null;
  if(!response.ok)throw new Error(data?.message??"No pudimos conectar con el servicio de reparto.");
  return data as T;
}
const post=(body:unknown):RequestInit=>({method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(body)});

export const loadPartnerPanel=(token:string)=>call<PartnerPanel>(panelBase(token));
export const sendPanelAction=(token:string,action:PanelAction,offer:string)=>call<PanelActionResult>(`${panelBase(token)}/actions`,post({action,token:offer}));
/** Registra o cambia al motorizado: cambiar el nombre invalida el enlace anterior. */
export const assignCourier=(token:string,offer:string,courierName:string)=>call<CourierAssignment>(`${panelBase(token)}/courier`,post({token:offer,courierName}));

export const loadCourierView=(courierToken:string)=>call<CourierView>(courierBase(courierToken));
export const sendCourierAction=(courierToken:string,action:CourierAction)=>call<PanelActionResult>(`${courierBase(courierToken)}/actions`,post({action}));
export const sendCourierLocation=(courierToken:string,position:CourierPosition)=>call<void>(`${courierBase(courierToken)}/location`,post(position));
