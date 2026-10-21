export type InventoryStatus="ok"|"low"|"out";
export type InventoryKind="product"|"ingredient";
export type InventoryAdjustmentType="entry"|"exit";
export type InventoryAdjustmentReason="surplus_adjustment"|"shortage_adjustment"|"waste"|"expiration"|"other_exit";

export type InventoryItem={
  inventoryItemId:string;
  productId:string|null;
  sku:string;
  name:string;
  kind:InventoryKind;
  categoryName:string|null;
  active:boolean;
  unit:string;
  quantity:string;
  minimumStock:string;
  reorderPoint:string;
  optimalStock:string;
  averageUnitCost:string;
  stockValue:string;
  status:InventoryStatus;
  updatedAt:string;
};

export type InventoryList={
  items:InventoryItem[];
  total:number;
  page:number;
  pageSize:number;
};

export type InventoryProductOption={
  id:string;
  productId:string|null;
  sku:string;
  name:string;
  kind:InventoryKind;
  categoryName:string|null;
  quantityControl:"inventory"|null;
  unit:string;
  quantity:string;
  minimumStock:string;
  reorderPoint:string;
  optimalStock:string;
  averageUnitCost:string;
};

export type InventoryAdjustmentDraft={
  inventoryItemId:string;
  movementType:InventoryAdjustmentType;
  reason:InventoryAdjustmentReason;
  quantity:string;
  observation:string;
};

export type InventoryAdjustmentResult={
  id:string;
  inventoryItemId:string;
  name:string;
  unit:string;
  movementType:InventoryAdjustmentType;
  reason:InventoryAdjustmentReason;
  quantity:number;
  stockBefore:number;
  stockAfter:number;
  observation:string;
  createdAt:string;
  createdByName:string;
};

export type StockMovement={
  id:string;
  inventoryItemId:string;
  productId:string|null;
  itemName:string;
  movementType:"entry"|"sale"|"sale_reversal"|"sale_adjustment"|"inventory_adjustment"|"supplier_return"|"receipt_correction"|"transfer_out"|"transfer_in"|"recipe_consumption"|"recipe_reversal";
  adjustmentType?:InventoryAdjustmentType|null;
  reason?:InventoryAdjustmentReason|null;
  quantityDelta:string;
  balanceBefore:string;
  balanceAfter:string;
  unitCost:string;
  valueDelta:string;
  balanceValueAfter:string;
  sourceType:"inventory_entry"|"order"|"inventory_adjustment"|"purchase_receipt"|"purchase_return"|"inventory_transfer";
  sourceId:string;
  sourceReference:string;
  note:string;
  createdByName:string;
  createdAt:string;
};


export type InventorySettingsDraft={inventoryItemId:string;minimumStock:string;reorderPoint:string;optimalStock:string};
export type InventoryTransferDraft={idempotencyKey:string;toLocationId:string;notes:string;items:Array<{inventoryItemId:string;quantity:string}>};
export type InventoryTransferSummary={id:string;code:string;fromLocationName:string;toLocationName:string;notes:string;createdByName:string;createdAt:string;itemCount:number};
export type LocationOption={id:string;name:string};
export type StockMovementsResponse={items:StockMovement[];total:number;page:number;pageSize:number};
