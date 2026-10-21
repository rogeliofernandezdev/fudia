import {apiFetch} from "@/shared/api/client";
import type {KitchenResponse,KitchenStatus} from "../domain/types";

export function listKitchenTickets(channel="",destination="kitchen"){
  const params=new URLSearchParams();
  params.set("destination",destination);
  if(channel)params.set("channel",channel);
  const suffix=params.toString()?`?${params.toString()}`:"";
  return apiFetch<KitchenResponse>(`kitchen/tickets${suffix}`);
}

export function updateKitchenTicketStatus(id:string,status:Extract<KitchenStatus,"preparando"|"listo">){
  return apiFetch<void>(`kitchen/tickets/${id}/status`,{
    method:"PATCH",
    body:JSON.stringify({status}),
  });
}
