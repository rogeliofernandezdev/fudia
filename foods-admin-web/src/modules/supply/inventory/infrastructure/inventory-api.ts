import {apiFetch} from "@/shared/api/client";
import type {InventoryAdjustmentDraft,InventoryAdjustmentResult,InventoryList,InventoryProductOption,InventorySettingsDraft,InventoryTransferDraft,InventoryTransferSummary,LocationOption,StockMovementsResponse} from "../domain/types";

export function listInventory(input:{q:string;page:number;pageSize:number}){
  const params=new URLSearchParams({q:input.q,page:String(input.page),pageSize:String(input.pageSize)});
  return apiFetch<InventoryList>(`inventory?${params.toString()}`);
}
export function listInventoryProducts(q=""){
  const params=new URLSearchParams({q});
  return apiFetch<{items:InventoryProductOption[]}>(`inventory/products?${params.toString()}`);
}
export function createInventoryAdjustment(draft:InventoryAdjustmentDraft){
  return apiFetch<InventoryAdjustmentResult>("inventory/adjustments",{method:"POST",body:JSON.stringify({
    inventoryItemId:draft.inventoryItemId,movementType:draft.movementType,reason:draft.reason,quantity:Number(draft.quantity),observation:draft.observation.trim(),
  })});
}
export function updateInventorySettings(draft:InventorySettingsDraft){
  return apiFetch<void>(`inventory/items/${draft.inventoryItemId}/settings`,{method:"PATCH",body:JSON.stringify({
    minimumStock:Number(draft.minimumStock||0),reorderPoint:Number(draft.reorderPoint||0),optimalStock:Number(draft.optimalStock||0),
  })});
}
export function listStockMovements(input:{inventoryItemId?:string;movementType?:string;sourceType?:string;from?:string;to?:string;page?:number;pageSize?:number}={}){
  const params=new URLSearchParams({page:String(input.page??1),pageSize:String(input.pageSize??25)});
  if(input.inventoryItemId)params.set("inventoryItemId",input.inventoryItemId);
  if(input.movementType)params.set("movementType",input.movementType);
  if(input.sourceType)params.set("sourceType",input.sourceType);
  if(input.from)params.set("from",input.from);
  if(input.to)params.set("to",input.to);
  return apiFetch<StockMovementsResponse>(`inventory/movements?${params.toString()}`);
}
export function listInventoryTransfers(page=1,pageSize=20){
  return apiFetch<{items:InventoryTransferSummary[];total:number;page:number;pageSize:number}>(`inventory/transfers?page=${page}&pageSize=${pageSize}`);
}
export function createInventoryTransfer(draft:InventoryTransferDraft){
  return apiFetch<{id:string;code:string;toLocationId:string;toLocationName:string}>("inventory/transfers",{method:"POST",body:JSON.stringify({
    idempotencyKey:draft.idempotencyKey,toLocationId:draft.toLocationId,notes:draft.notes.trim(),items:draft.items.filter(i=>Number(i.quantity)>0).map(i=>({inventoryItemId:i.inventoryItemId,quantity:Number(i.quantity)})),
  })});
}
export function listTransferLocations(){return apiFetch<{items:LocationOption[]}>("locations/available")}
