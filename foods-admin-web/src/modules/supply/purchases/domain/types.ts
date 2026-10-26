export type PurchaseStatus="draft"|"pending_approval"|"approved"|"partially_received"|"received"|"cancelled";
export type PurchaseTab="orders"|"receipts"|"suppliers";
export type PresentationType=string;
export type PurchaseCombination={code:string;name:string;unit:string;unitName:string;unitCountable:boolean;isDefault:boolean};
export type PurchasePresentationDraft={presentationType:string;unitsPerPresentation:string;isDefault:boolean};

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
  name:string;
  isDefault:boolean;
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
  presentationName:string;
  unitsPerPresentation:string;
  quantity:string;
  receivedQuantity:string;
  pendingQuantity:string;
  stockQuantity:string;
  unitCost:string;
  lineTotal:string;
};

/** Documento del flujo de recepción de una orden, del más antiguo al más reciente. */
export type PurchaseOrderEvent={id:string;kind:"receipt"|"receipt_correction"|"supplier_return";code:string;receiptId:string;reason:string;notes:string;createdByName:string;createdAt:string;itemCount:number};
export type PurchaseOrder=PurchaseOrderSummary&{items:PurchaseOrderItem[];history:PurchaseOrderEvent[]};

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

/** Producto de la carta que aún usa Disponibilidad (o ningún control) y puede vincularse al inventario. */
export type LinkableProduct={productId:string;sku:string;name:string;categoryName:string|null;quantityControl:"none"|"portions"};

export type PurchaseInventoryItemDraft={
  /** new_product se vende a clientes; new_ingredient es un insumo; link_product reutiliza un producto existente. */
  mode:"new_product"|"new_ingredient"|"link_product";
  productId:string;
  categoryId:string;
  name:string;
  price:string;
  unit:string;
  presentationType:PresentationType;
  unitsPerPresentation:string;
  minimumStock:string;
  presentations?:PurchasePresentationDraft[];
  /** Regla de la unidad recibida del API; no la elige el usuario. */
  unitCountable?:boolean;
  unitName?:string;
};

export type PurchaseReceiptLineDraft={
  purchaseOrderItemId:string;
  quantity:string;
};

export type PurchaseReceiptDraft={
  purchaseOrderId:string;
  idempotencyKey:string;
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


export type PurchaseReceiptSummary={id:string;code:string;purchaseOrderId:string;number:string;supplierName:string;notes:string;createdByName:string;createdAt:string;itemCount:number};
export type PurchaseReceiptItem={id:string;purchaseOrderItemId:string;inventoryItemId:string;itemName:string;quantity:string;presentationType:PresentationType;presentationName:string;unitsPerPresentation:string;stockQuantity:string;unitCost:string;returnedQuantity:string;returnableQuantity:string};
export type PurchaseReceiptDetail=PurchaseReceiptSummary&{items:PurchaseReceiptItem[]};
export type PurchaseReturnKind="supplier_return"|"receipt_correction";
export type PurchaseReturnDraft={idempotencyKey:string;purchaseReceiptId:string;kind:PurchaseReturnKind;reason:string;notes:string;items:Array<{purchaseReceiptItemId:string;quantity:string}>};
