import {apiFetch} from "@/shared/api/client";
import type {DeliveryDispatch,DeliveryLink} from "../domain/types";

const location=(id:string)=>`locations/${encodeURIComponent(id)}`;
const order=(id:string)=>`orders/${encodeURIComponent(id)}/delivery-dispatch`;

export function getDeliveryLink(locationId:string){return apiFetch<DeliveryLink>(`${location(locationId)}/delivery-link`);}
export function rotateDeliveryLink(locationId:string){return apiFetch<DeliveryLink>(`${location(locationId)}/delivery-link`,{method:"POST"});}
export function retryDeliveryDispatch(orderId:string){return apiFetch<DeliveryDispatch>(`${order(orderId)}/retry`,{method:"POST"});}
export function switchToOwnDelivery(orderId:string){return apiFetch<DeliveryDispatch>(`${order(orderId)}/own`,{method:"POST"});}
