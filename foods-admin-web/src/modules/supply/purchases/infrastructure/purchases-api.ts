import {apiFetch} from "@/shared/api/client";
import {uploadProductImage} from "@/shared/api/product-image";
import type {PurchaseInventoryItemDraft,PurchaseInventoryOption,PurchaseItemCategory,PurchaseOrder,PurchaseOrderDraft,PurchaseOrdersResponse,PurchaseReceiptDetail,PurchaseReceiptDraft,PurchaseReceiptResult,PurchaseReceiptSummary,PurchaseReturnDraft,PurchaseStatus,Supplier,SupplierDraft,SuppliersResponse} from "../domain/types";

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

export function setPurchaseOrderStatus(id:string,status:"draft"|"pending_approval"|"cancelled"){
  return apiFetch<void>(`purchase-orders/${id}/status`,{method:"PATCH",body:JSON.stringify({status})});
}
export function approvePurchaseOrder(id:string){
  return apiFetch<void>(`purchase-orders/${id}/approve`,{method:"POST"});
}

export function receivePurchaseOrder(draft:PurchaseReceiptDraft){
  return apiFetch<PurchaseReceiptResult>(`purchase-orders/${draft.purchaseOrderId}/receive`,{
    method:"POST",
    body:JSON.stringify({
      idempotencyKey:draft.idempotencyKey,
      notes:draft.notes.trim(),
      items:draft.items.filter(item=>Number(item.quantity)>0).map(item=>({
        purchaseOrderItemId:item.purchaseOrderItemId,
        quantity:Number(item.quantity),
      })),
    }),
  });
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

export function listPurchaseInventory(q=""){
  const params=new URLSearchParams({q});
  return apiFetch<{items:PurchaseInventoryOption[]}>(`purchase-inventory-items?${params.toString()}`);
}

export async function listPurchaseItemCategories(){
  const pageSize=100;
  const items:PurchaseItemCategory[]=[];
  let page=1;
  let total=0;
  do{
    const params=new URLSearchParams({productType:"retail",page:String(page),pageSize:String(pageSize)});
    const response=await apiFetch<{items:PurchaseItemCategory[];total:number}>(`purchase-item-categories?${params.toString()}`);
    items.push(...response.items);
    total=response.total;
    if(response.items.length===0)break;
    page++;
  }while(items.length<total);
  return items;
}

export async function createPurchaseInventoryItem(draft:PurchaseInventoryItemDraft,file:File|null=null){
  const common={
    unit:draft.unit,
    presentationType:draft.presentationType,
    unitsPerPresentation:Number(draft.presentationType==="unit"?1:draft.unitsPerPresentation),
    minimumStock:Number(draft.minimumStock||0),
  };
  const payload=draft.mode==="new_ingredient"
    ?{...common,newIngredient:{name:draft.name.trim()}}
    :{...common,newProduct:{
      sku:"",
      name:draft.name.trim(),
      categoryId:draft.categoryId.trim(),
      description:draft.description.trim(),
      price:draft.price.trim(),
    }};
  const item=await apiFetch<PurchaseInventoryOption>("purchase-inventory-items",{method:"POST",body:JSON.stringify(payload)});
  if(file&&item.productId)await uploadProductImage(item.productId,file);
  return item;
}


export function listPurchaseReceipts(input:{q?:string;from?:string;to?:string;page?:number;pageSize?:number}={}){
 const params=new URLSearchParams({q:input.q??"",from:input.from??"",to:input.to??"",page:String(input.page??1),pageSize:String(input.pageSize??20)});
 return apiFetch<{items:PurchaseReceiptSummary[];total:number;page:number;pageSize:number}>(`purchase-receipts?${params.toString()}`);
}
export function getPurchaseReceipt(id:string){return apiFetch<PurchaseReceiptDetail>(`purchase-receipts/${id}`)}
export function createPurchaseReturn(draft:PurchaseReturnDraft){
 return apiFetch<{id:string;code:string;kind:string;purchaseOrderId:string;number:string}>(`purchase-receipts/${draft.purchaseReceiptId}/returns`,{method:"POST",body:JSON.stringify({
  kind:draft.kind,reason:draft.reason.trim(),notes:draft.notes.trim(),
  items:draft.items.filter(i=>Number(i.quantity)>0).map(i=>({purchaseReceiptItemId:i.purchaseReceiptItemId,quantity:Number(i.quantity)})),
 })})
}
