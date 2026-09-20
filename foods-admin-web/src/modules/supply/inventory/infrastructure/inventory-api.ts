import {apiFetch} from "@/shared/api/client";
import type {InventoryEntryDraft,InventoryList,InventoryProductOption,StockMovement} from "../domain/types";

export function listInventory(input:{q:string;page:number;pageSize:number}){
  const params=new URLSearchParams({q:input.q,page:String(input.page),pageSize:String(input.pageSize)});
  return apiFetch<InventoryList>(`inventory?${params.toString()}`);
}

export function listInventoryProducts(q=""){
  const params=new URLSearchParams({q});
  return apiFetch<{items:InventoryProductOption[]}>(`inventory/products?${params.toString()}`);
}

export function createInventoryEntry(draft:InventoryEntryDraft){
  const common={
    quantity:Number(draft.quantity),
    unit:draft.unit.trim()||"und",
    presentationType:draft.presentationType,
    unitsPerPresentation:Number(draft.unitsPerPresentation||1),
    minimumStock:Number(draft.minimumStock||0),
    note:draft.note.trim(),
  };
  const payload=draft.mode==="existing"
    ?{...common,inventoryItemId:draft.inventoryItemId}
    :draft.mode==="new_ingredient"
      ?{...common,newIngredient:{name:draft.name.trim()}}
      :{...common,newProduct:{
        sku:draft.sku.trim(),
        name:draft.name.trim(),
        description:draft.description.trim(),
        price:draft.price.trim(),
      }};
  return apiFetch<{
    id:string;
    inventoryItemId:string;
    productId:string|null;
    sku:string;
    name:string;
    kind:"product"|"ingredient";
    quantity:number;
    presentationId:string;
    presentationType:InventoryEntryDraft["presentationType"];
    unitsPerPresentation:number;
    stockQuantity:number;
    unit:string;
    balance:number;
    createdProduct:boolean;
    createdInventoryItem:boolean;
  }>("inventory/entries",{method:"POST",body:JSON.stringify(payload)});
}

export function listStockMovements(inventoryItemId=""){
  const params=new URLSearchParams();
  if(inventoryItemId)params.set("inventoryItemId",inventoryItemId);
  const suffix=params.toString()?`?${params.toString()}`:"";
  return apiFetch<{items:StockMovement[]}>(`inventory/movements${suffix}`);
}
