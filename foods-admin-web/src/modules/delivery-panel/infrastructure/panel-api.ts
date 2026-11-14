import type {CourierPosition,PanelAction,PanelActionResult,PartnerPanel} from "../domain/types";

// El panel no tiene sesión: el enlace privado viaja en la ruta del BFF público
// del mismo origen, que lo reenvía a foods-backend.
const base=(token:string)=>`/api/public/delivery-panel/${encodeURIComponent(token)}`;

async function call<T>(url:string,init?:RequestInit):Promise<T>{
  const response=await fetch(url,{cache:"no-store",...init});
  if(response.status===204)return undefined as T;
  const data=await response.json().catch(()=>null) as {message?:string}|null;
  if(!response.ok)throw new Error(data?.message??"No pudimos conectar con el panel de reparto.");
  return data as T;
}
const post=(body:unknown):RequestInit=>({method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(body)});

export const loadPartnerPanel=(token:string)=>call<PartnerPanel>(base(token));
export const sendPanelAction=(token:string,action:PanelAction,offer:string)=>call<PanelActionResult>(`${base(token)}/actions`,post({action,token:offer}));
export const sendCourierLocation=(token:string,position:CourierPosition)=>call<void>(`${base(token)}/location`,post(position));
