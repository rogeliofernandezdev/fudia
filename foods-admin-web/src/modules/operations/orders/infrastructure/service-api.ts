import {apiFetch} from "@/shared/api/client";
import type {Draft} from "../../salon/domain/types";
import type {Order} from "../domain/types";

export function closeOrderAccount(id:string){return apiFetch<Order>(`orders/${id}/bill/close`,{method:"POST"})}
export function addOrderConsumption(id:string,draft:Draft,requestKey:string){
 return apiFetch<Order>(`orders/${id}/items`,{method:"POST",body:JSON.stringify({requestKey,items:draft.lines.map(line=>({productId:line.productId,qty:line.qty,note:line.note,selections:line.selections.map(selection=>({groupId:selection.groupId,productId:selection.productId}))}))})});
}
