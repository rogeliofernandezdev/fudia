import {apiFetch} from "@/shared/api/client";
import type {PayDevice,PayDeviceStatusFilter,PayNotification,PayNotificationStatus,PayOrderRef,PayPairingDraft,PayPairingStatus,PayProvider} from "../domain/types";

const locationDevices=(locationId:string)=>`locations/${encodeURIComponent(locationId)}/pay-devices`;

/** Locales genera el código temporal del local que se muestra como QR. */
export function createPayPairing(locationId:string){
  return apiFetch<PayPairingDraft>(`${locationDevices(locationId)}/pairing`,{method:"POST"});
}

/** Estado del código mientras el QR está a la vista (como WhatsApp Web). */
export function getPayPairingStatus(locationId:string,id:string){
  return apiFetch<PayPairingStatus>(`${locationDevices(locationId)}/pairing/${encodeURIComponent(id)}`);
}

/** Una página de dispositivos del local; status vacío trae activos y revocados. */
export function listPayDevices(locationId:string,filter:{status:PayDeviceStatusFilter;page:number;size:number}){
  const params=new URLSearchParams({status:filter.status,page:String(filter.page),size:String(filter.size)});
  return apiFetch<{items:PayDevice[];total:number}>(`${locationDevices(locationId)}?${params.toString()}`);
}

/** Revoca la credencial: el dispositivo deja de autenticarse de inmediato. */
export function revokePayDevice(locationId:string,id:string){
  return apiFetch<PayDevice>(`${locationDevices(locationId)}/${encodeURIComponent(id)}/revoke`,{method:"POST"});
}

/** Pide al backend despertar (push silencioso) los teléfonos del local de la
 * sesión; `devices` dice cuántos pueden recibirlo. */
export function wakePayDevices(){
  return apiFetch<{devices:number;sent:number}>("pay-devices/wake",{method:"POST"});
}

export function listPayNotifications(filter:{status:string;provider:string}){
  const params=new URLSearchParams({status:filter.status,provider:filter.provider});
  return apiFetch<{items:PayNotification[];total:number}>(`pay-notifications?${params.toString()}`);
}

/** Una página de pagos detectados con el total de los filtros (la lista crece con el historial). */
/** `q` busca en el servidor por remitente o monto (las billeteras no envían número de operación). */
export function listPayNotificationPage(filter:{status:string;provider:string;q:string;page:number;size:number}){
  const params=new URLSearchParams({status:filter.status,provider:filter.provider,q:filter.q,page:String(filter.page),size:String(filter.size)});
  return apiFetch<{items:PayNotification[];total:number}>(`pay-notifications?${params.toString()}`);
}

/** Asociar: registra el cobro real con el flujo de pagos existente y liga la
 * notificación al pedido y al pago creado. */
export function associatePayNotification(id:string,orderId:string,method:string,amount?:number){
  return apiFetch<PayNotification>(`pay-notifications/${encodeURIComponent(id)}/associate`,{
    method:"POST",
    body:JSON.stringify({orderId,method,...(amount?{amount}:{})}),
  });
}

export function dismissPayNotification(id:string){
  return apiFetch<PayNotification>(`pay-notifications/${encodeURIComponent(id)}/dismiss`,{method:"POST"});
}

export function paySalesMethods(){
  return apiFetch<{items:Array<{code:string;name:string}>}>("payment-methods/operational?usage=sales");
}

/** Búsqueda de pedidos del local; paymentStatus acota a los que pueden
 * recibir cobro (pendiente o parcial). */
/** Solo pedidos por cobrar y listos para cobrar (`ready`, la misma regla del cobro):
 * ni los pagados ni los que el cobro rechazaría ocupan los 12 lugares de la búsqueda. */
export function searchPayOrders(q:string,paymentStatus="unpaid"){
  const params=new URLSearchParams({q,paymentStatus,ready:"true",page:"1",pageSize:"12"});
  return apiFetch<{items:PayOrderRef[]}>(`pos/orders?${params.toString()}`);
}

/** Detalle del pedido para «Ver detalle» antes de asociar. */
export function payOrderDetail(orderId:string){
  return apiFetch<{
    order:{code:string;channel:string;customerName?:string;status:string;total?:string};
    paidAmount:string;
    remainingAmount:string;
    paymentStatus:string;
    payments:Array<{id:string;method:string;methodName?:string;amount:string;createdAt:string}>;
  }>(`pos/orders/${encodeURIComponent(orderId)}`);
}

export const payStatusOptions:PayNotificationStatus[]=["detected","dismissed"];
export const payProviderOptions:PayProvider[]=["yape","plin"];
