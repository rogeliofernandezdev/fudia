import {apiFetch,ApiClientError} from "@/shared/api/client";
import type {DeliveryFees,DeliveryFeesPayload} from "../domain/delivery-fees";
import type {DeliveryBoardOrder,DeliveryDispatch,DeliveryLink,LocationFailedDelivery} from "../domain/types";

const location=(id:string)=>`locations/${encodeURIComponent(id)}`;
const order=(id:string)=>`orders/${encodeURIComponent(id)}/delivery-dispatch`;

export function getDeliveryLink(locationId:string){return apiFetch<DeliveryLink>(`${location(locationId)}/delivery-link`);}
export function rotateDeliveryLink(locationId:string){return apiFetch<DeliveryLink>(`${location(locationId)}/delivery-link`,{method:"POST"});}
export async function getDeliveryDispatch(orderId:string){
  try{return await apiFetch<DeliveryDispatch>(order(orderId));}
  catch(error){
    if(error instanceof ApiClientError&&error.status===404&&error.code==="delivery_dispatch_not_found")return null;
    throw error;
  }
}
export function switchToOwnDelivery(orderId:string){return apiFetch<DeliveryDispatch>(`${order(orderId)}/own`,{method:"POST"});}

// Delivery del local: tablero, proveedores que lo cubren y entregas fallidas.
export function getDeliveryBoard(){return apiFetch<{items:DeliveryBoardOrder[]}>("delivery/orders");}
export function getLocationFailedDeliveries(){return apiFetch<{items:LocationFailedDelivery[]}>("delivery/failed-deliveries");}

// Tarifas de envío por distancia: el backend las expone bajo la configuración de Concierge, que las usa para cotizar.
export function getDeliveryFees(){return apiFetch<DeliveryFees>("concierge-settings/delivery-fees");}
export function saveDeliveryFees(payload:DeliveryFeesPayload){return apiFetch<DeliveryFees>("concierge-settings/delivery-fees",{method:"PUT",body:JSON.stringify(payload)});}
