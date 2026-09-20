export type InventoryStatus="ok"|"low"|"out";
export type InventoryPresentationType="unit"|"package"|"box";
export type InventoryKind="product"|"ingredient";

export type InventoryPresentation={
  id:string;
  presentationType:InventoryPresentationType;
  unitsPerPresentation:string;
};

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
  minimumStock:string;
  presentations:InventoryPresentation[];
};

export type InventoryEntryDraft={
  mode:"existing"|"new_product"|"new_ingredient";
  inventoryItemId:string;
  productId:string;
  sku:string;
  name:string;
  description:string;
  price:string;
  quantity:string;
  unit:string;
  presentationType:InventoryPresentationType;
  unitsPerPresentation:string;
  minimumStock:string;
  note:string;
};

export type StockMovement={
  id:string;
  inventoryItemId:string;
  productId:string|null;
  itemName:string;
  movementType:"entry"|"sale"|"sale_reversal"|"sale_adjustment";
  quantityDelta:string;
  balanceAfter:string;
  sourceType:"inventory_entry"|"order";
  sourceId:string;
  sourceReference:string;
  note:string;
  createdAt:string;
};
