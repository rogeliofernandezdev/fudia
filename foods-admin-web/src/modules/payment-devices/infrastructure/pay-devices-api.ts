import {apiFetch} from "@/shared/api/client";
import type {PayDevice,PayNotification,PayNotificationStatus,PayOrderRef,PayPairingDraft,PayPairingStatus,PayProvider} from "../domain/types";

/** Caja genera el código temporal que se muestra como QR. */
export function createPayPairing(){
  return apiFetch<PayPairingDraft>("pay-devices/pairing",{method:"POST"});
}

/** Estado del código mientras el QR está a la vista (como WhatsApp Web). */
export function getPayPairingStatus(id:string){
  return apiFetch<PayPairingStatus>(`pay-devices/pairing/${encodeURIComponent(id)}`);
}

export function listPayDevices(){
  return apiFetch<{items:PayDevice[]}>("pay-devices");
}

export function renamePayDevice(id:string,name:string){
  return apiFetch<PayDevice>(`pay-devices/${encodeURIComponent(id)}`,{method:"PATCH",body:JSON.stringify({name})});
}

/** Revoca la credencial: el dispositivo deja de autenticarse de inmediato. */
export function revokePayDevice(id:string){
  return apiFetch<PayDevice>(`pay-devices/${encodeURIComponent(id)}/revoke`,{method:"POST"});
}

export function listPayNotifications(filter:{status:string;provider:string}){
  const params=new URLSearchParams({status:filter.status,provider:filter.provider});
  return apiFetch<{items:PayNotification[];total:number}>(`pay-notifications?${params.toString()}`);
}

/** Una página de pagos detectados con el total de los filtros (la lista crece con el historial). */
export function listPayNotificationPage(filter:{status:string;provider:string;page:number;size:number}){
  const params=new URLSearchParams({status:filter.status,provider:filter.provider,page:String(filter.page),size:String(filter.size)});
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
export function searchPayOrders(q:string,paymentStatus=""){
  const params=new URLSearchParams({q,paymentStatus,page:"1",pageSize:"12"});
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
