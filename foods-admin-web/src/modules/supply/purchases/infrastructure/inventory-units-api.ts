import {apiFetch} from "@/shared/api/client";
import type {InventoryUnit,InventoryUnitDraft} from "../domain/inventory-unit";

export function listInventoryUnits(){
 return apiFetch<{items:InventoryUnit[]}>("purchase-units");
}
export function createInventoryUnit(draft:InventoryUnitDraft){
 return apiFetch<InventoryUnit>("purchase-units",{method:"POST",body:JSON.stringify({code:draft.code.trim().toLowerCase(),name:draft.name.trim()})});
}
