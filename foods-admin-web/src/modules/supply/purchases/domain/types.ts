export type PurchaseStatus="draft"|"pending_approval"|"approved"|"partially_received"|"received"|"cancelled";
export type PurchaseTab="orders"|"receipts"|"suppliers";
export type PresentationType="unit"|"package"|"box";

export type Supplier={
  id:string;
  taxId:string;
  name:string;
  email:string;
  phone:string;
  active:boolean;
};

export type SuppliersResponse={
  items:Supplier[];
  total:number;
  page:number;
  pageSize:number;
};

export type PurchasePresentation={
  id:string;
  presentationType:PresentationType;
  unitsPerPresentation:string;
};

export type PurchaseInventoryOption={
  id:string;
  productId:string|null;
  sku:string;
  name:string;
  kind:"product"|"ingredient";
  categoryName:string|null;
  quantityControl:"inventory"|null;
  unit:string;
  quantity?:string;
  minimumStock:string;
  presentations:PurchasePresentation[];
};

export type PurchaseItemCategory={
  id:string;
  name:string;
};

export type PurchaseOrderSummary={
  id:string;
  number:string;
  supplierId:string;
  supplierName:string;
  status:PurchaseStatus;
  total:string;
  notes:string;
  expectedAt:string|null;
  itemCount:number;
  createdAt:string;
  updatedAt:string;
  approvedAt:string|null;
  receivedAt:string|null;
  cancelledAt:string|null;
};

export type PurchaseOrderItem={
  id:string;
  inventoryItemId:string;
  itemName:string;
  sku:string;
  unit:string;
  presentationId:string;
  presentationType:PresentationType;
  unitsPerPresentation:string;
  quantity:string;
  receivedQuantity:string;
  pendingQuantity:string;
  stockQuantity:string;
  unitCost:string;
  lineTotal:string;
};

export type PurchaseOrder=PurchaseOrderSummary&{items:PurchaseOrderItem[]};

export type PurchaseOrdersResponse={
  items:PurchaseOrderSummary[];
  total:number;
  page:number;
  pageSize:number;
};

export type PurchaseLineDraft={
  inventoryItemId:string;
  presentationId:string;
  quantity:string;
  unitCost:string;
};

export type PurchaseOrderDraft={
  id:string;
  supplierId:string;
  expectedAt:string;
  notes:string;
  items:PurchaseLineDraft[];
};

export type SupplierDraft={
  id:string;
  taxId:string;
  name:string;
  email:string;
  phone:string;
};

export type PurchaseInventoryItemDraft={
  mode:"new_product"|"new_ingredient";
  categoryId:string;
  name:string;
  description:string;
  price:string;
  unit:string;
  presentationType:PresentationType;
  unitsPerPresentation:string;
  minimumStock:string;
};

export type PurchaseReceiptLineDraft={
  purchaseOrderItemId:string;
  quantity:string;
};

export type PurchaseReceiptDraft={
  purchaseOrderId:string;
  notes:string;
  items:PurchaseReceiptLineDraft[];
};

export type PurchaseReceiptResult={
  id:string;
  code:string;
  purchaseOrderId:string;
  number:string;
  status:"partially_received"|"received";
  receivedAt:string|null;
};
