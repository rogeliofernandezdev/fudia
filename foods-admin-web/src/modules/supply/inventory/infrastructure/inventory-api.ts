import {apiFetch} from "@/shared/api/client";
import type {InventoryAdjustmentDraft,InventoryAdjustmentResult,InventoryItemDetail,InventoryList,InventoryProductOption,InventorySettingsDraft,InventoryStatusFilter,InventoryTransferDraft,InventoryTransferSummary,InventoryUnitChangePreview,InventoryUnitOption,LocationOption,StockMovementsResponse} from "../domain/types";

export function listInventory(input:{q:string;page:number;pageSize:number;status?:InventoryStatusFilter}){
  const params=new URLSearchParams({q:input.q,page:String(input.page),pageSize:String(input.pageSize),status:input.status??"active"});
  return apiFetch<InventoryList>(`inventory?${params.toString()}`);
}
// Lookups search on the server: the list is paginated and never truncates the catalog.
// Ajustes y transferencias solo operan artículos del local activo.
export function searchInventoryProducts(q=""){
  const params=new URLSearchParams({q:q.trim(),page:"1",pageSize:"20",scope:"location"});
  return apiFetch<{items:InventoryProductOption[];total:number}>(`inventory/products?${params.toString()}`);
}
export function createInventoryAdjustment(draft:InventoryAdjustmentDraft){
  return apiFetch<InventoryAdjustmentResult>("inventory/adjustments",{method:"POST",body:JSON.stringify({
    inventoryItemId:draft.inventoryItemId,movementType:draft.movementType,reason:draft.reason,quantity:Number(draft.quantity),observation:draft.observation.trim(),
    ...(draft.movementType==="entry"&&draft.unitCost.trim()!==""?{unitCost:Number(draft.unitCost)}:{}),
  })});
}
export function updateInventorySettings(draft:InventorySettingsDraft){
  return apiFetch<void>(`inventory/items/${draft.inventoryItemId}/settings`,{method:"PATCH",body:JSON.stringify({
    ...(draft.name!==undefined?{name:draft.name.trim()}:{}),
    ...(draft.unit?{unit:draft.unit}:{}),
    minimumStock:Number(draft.minimumStock||0),optimalStock:Number(draft.optimalStock||0),
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
export function getInventoryItem(id:string){return apiFetch<InventoryItemDetail>(`inventory/items/${id}`)}
export function previewInventoryUnitChange(id:string,unit:string){
  return apiFetch<InventoryUnitChangePreview>(`inventory/items/${id}/unit-change?${new URLSearchParams({unit}).toString()}`);
}
/** Activa o desactiva el artículo solo en el local activo. */
export function setInventoryItemActive(id:string,active:boolean){
  return apiFetch<void>(`inventory/items/${id}/status`,{method:"PATCH",body:JSON.stringify({active})});
}
/** Deja de ofrecer la presentación en compras nuevas; las anteriores la conservan. */
export function removeInventoryPresentation(id:string,presentationId:string){
  return apiFetch<void>(`inventory/items/${id}/presentations/${presentationId}`,{method:"DELETE"});
}
export function listInventoryUnitOptions(){return apiFetch<{items:InventoryUnitOption[]}>("inventory/units")}
