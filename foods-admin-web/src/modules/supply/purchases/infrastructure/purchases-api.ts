import {apiFetch} from "@/shared/api/client";
import type {PurchaseInventoryOption,PurchaseOrder,PurchaseOrderDraft,PurchaseOrdersResponse,PurchaseStatus,Supplier,SupplierDraft,SuppliersResponse} from "../domain/types";

export function listPurchaseOrders(input:{q:string;status:string;page:number;pageSize:number}){
  const params=new URLSearchParams({q:input.q,status:input.status,page:String(input.page),pageSize:String(input.pageSize)});
  return apiFetch<PurchaseOrdersResponse>(`purchase-orders?${params.toString()}`);
}

export function getPurchaseOrder(id:string){
  return apiFetch<PurchaseOrder>(`purchase-orders/${id}`);
}

export function savePurchaseOrder(draft:PurchaseOrderDraft){
  const payload={
    supplierId:draft.supplierId,
    expectedAt:draft.expectedAt,
    notes:draft.notes.trim(),
    items:draft.items.map(item=>({
      inventoryItemId:item.inventoryItemId,
      presentationId:item.presentationId,
      quantity:Number(item.quantity),
      unitCost:item.unitCost.trim(),
    })),
  };
  return apiFetch<{id:string;number:string;status:PurchaseStatus}>(draft.id?`purchase-orders/${draft.id}`:"purchase-orders",{
    method:draft.id?"PATCH":"POST",
    body:JSON.stringify(payload),
  });
}

export function setPurchaseOrderStatus(id:string,status:Exclude<PurchaseStatus,"received">){
  return apiFetch<void>(`purchase-orders/${id}/status`,{method:"PATCH",body:JSON.stringify({status})});
}

export function receivePurchaseOrder(id:string){
  return apiFetch<{id:string;number:string;status:"received";receivedAt:string}>(`purchase-orders/${id}/receive`,{method:"POST"});
}

export function listSuppliers(input:{q?:string;status?:string;page?:number;pageSize?:number}={}){
  const params=new URLSearchParams({
    q:input.q??"",
    status:input.status??"",
    page:String(input.page??1),
    pageSize:String(input.pageSize??20),
  });
  return apiFetch<SuppliersResponse>(`suppliers?${params.toString()}`);
}

export function saveSupplier(draft:SupplierDraft){
  const payload={taxId:draft.taxId.trim(),name:draft.name.trim(),email:draft.email.trim(),phone:draft.phone.trim()};
  return apiFetch<Supplier>(draft.id?`suppliers/${draft.id}`:"suppliers",{
    method:draft.id?"PATCH":"POST",
    body:JSON.stringify(payload),
  });
}

export function setSupplierActive(id:string,active:boolean){
  return apiFetch<void>(`suppliers/${id}/status`,{method:"PATCH",body:JSON.stringify({active})});
}

export function listPurchaseInventory(){
  return apiFetch<{items:PurchaseInventoryOption[]}>("inventory/products");
}
