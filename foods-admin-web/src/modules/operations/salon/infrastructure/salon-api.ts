import {apiFetch} from "@/shared/api/client";
import type {Draft,FloorTable,Order} from "../domain/types";

function linesPayload(draft:Draft){
 return draft.lines.map(line=>({
  id:line.sourceItemId,
  productId:line.productId,
  name:line.name,
  qty:line.qty,
  unitPrice:line.unitPrice,
  note:line.note,
  reprice:Boolean(line.repriceCombo),
  selections:line.selections.map(selection=>({groupId:selection.groupId,productId:selection.productId})),
 }));
}
export function getSalonFloor(){return apiFetch<{items:FloorTable[]}>("orders/floor")}
export function getSalonOrder(id:string){return apiFetch<Order>(`orders/${id}`)}
export function updateSalonOrderStatus(id:string,status:string){
 return apiFetch<Order>(`orders/${id}/status`,{method:"PATCH",body:JSON.stringify({status})});
}
export function createSalonOrder(draft:Draft){
 return apiFetch<Order>("orders",{method:"POST",body:JSON.stringify({
  channel:draft.channel,customerName:draft.customerName,customerPhone:draft.customerPhone,address:draft.address,
  reference:draft.reference,tableId:draft.tableId,notes:draft.notes,deliveryFee:Number(draft.deliveryFee)||0,items:linesPayload(draft),
 })});
}
export function updateSalonOrder(id:string,draft:Draft){
 return apiFetch<Order>(`orders/${id}`,{method:"PATCH",body:JSON.stringify({
  customerName:draft.customerName,customerPhone:draft.customerPhone,address:draft.address,reference:draft.reference,
  notes:draft.notes,deliveryFee:Number(draft.deliveryFee)||0,items:linesPayload(draft),
 })});
}
