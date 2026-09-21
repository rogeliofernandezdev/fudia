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
  movementType:"entry"|"sale"|"sale_reversal"|"sale_adjustment"|"inventory_adjustment";
  adjustmentType?:InventoryAdjustmentType|null;
  reason?:InventoryAdjustmentReason|null;
  quantityDelta:string;
  balanceBefore:string;
  balanceAfter:string;
  sourceType:"inventory_entry"|"order"|"inventory_adjustment";
  sourceId:string;
  sourceReference:string;
  note:string;
  createdByName:string;
  createdAt:string;
};
