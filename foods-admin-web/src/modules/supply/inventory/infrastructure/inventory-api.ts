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
    minimumStock:Number(draft.minimumStock||0),
    note:draft.note.trim(),
  };
  const payload=draft.mode==="existing"
    ?{...common,productId:draft.productId}
    :{...common,newProduct:{
      sku:draft.sku.trim(),
      name:draft.name.trim(),
      description:draft.description.trim(),
      price:draft.price.trim(),
    }};
  return apiFetch<{id:string;productId:string;sku:string;name:string;quantity:number;unit:string;balance:number;createdProduct:boolean}>("inventory/entries",{
    method:"POST",
    body:JSON.stringify(payload),
  });
}

export function listStockMovements(productId=""){
  const params=new URLSearchParams();
  if(productId)params.set("productId",productId);
  const suffix=params.toString()?`?${params.toString()}`:"";
  return apiFetch<{items:StockMovement[]}>(`inventory/movements${suffix}`);
}
