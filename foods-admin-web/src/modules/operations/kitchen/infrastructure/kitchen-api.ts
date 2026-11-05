import {apiFetch} from "@/shared/api/client";
import type {KitchenResponse,KitchenStatus} from "../domain/types";

export function listKitchenTickets(channel="",destination="kitchen"){
  const params=new URLSearchParams();
  params.set("destination",destination);
  if(channel)params.set("channel",channel);
  const suffix=params.toString()?`?${params.toString()}`:"";
  return apiFetch<KitchenResponse>(`kitchen/tickets${suffix}`);
}

/** Marca un plato de la comanda como listo o lo devuelve a preparación. */
export function updateKitchenItemStatus(ticketId:string,itemId:string,status:Extract<KitchenStatus,"preparando"|"listo">){
  return apiFetch<void>(`kitchen/tickets/${encodeURIComponent(ticketId)}/items/${encodeURIComponent(itemId)}/status`,{
    method:"PATCH",
    body:JSON.stringify({status}),
  });
}

export function updateKitchenTicketStatus(id:string,status:Extract<KitchenStatus,"preparando"|"listo">){
  return apiFetch<void>(`kitchen/tickets/${id}/status`,{
    method:"PATCH",
    body:JSON.stringify({status}),
  });
}
