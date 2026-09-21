import {apiFetch} from "@/shared/api/client";
import type {InventoryAdjustmentDraft,InventoryAdjustmentResult,InventoryList,InventoryProductOption,StockMovement} from "../domain/types";

export function listInventory(input:{q:string;page:number;pageSize:number}){
  const params=new URLSearchParams({q:input.q,page:String(input.page),pageSize:String(input.pageSize)});
  return apiFetch<InventoryList>(`inventory?${params.toString()}`);
}

export function listInventoryProducts(q=""){
  const params=new URLSearchParams({q});
  return apiFetch<{items:InventoryProductOption[]}>(`inventory/products?${params.toString()}`);
}

export function createInventoryAdjustment(draft:InventoryAdjustmentDraft){
  return apiFetch<InventoryAdjustmentResult>("inventory/adjustments",{
    method:"POST",
    body:JSON.stringify({
      inventoryItemId:draft.inventoryItemId,
      movementType:draft.movementType,
      reason:draft.reason,
      quantity:Number(draft.quantity),
      observation:draft.observation.trim(),
    }),
  });
}

export function listStockMovements(inventoryItemId=""){
  const params=new URLSearchParams();
  if(inventoryItemId)params.set("inventoryItemId",inventoryItemId);
  const suffix=params.toString()?`?${params.toString()}`:"";
  return apiFetch<{items:StockMovement[]}>(`inventory/movements${suffix}`);
}
