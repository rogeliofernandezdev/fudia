export type InventoryStatus="ok"|"low"|"out";

export type InventoryItem={
  productId:string;
  sku:string;
  name:string;
  categoryName:string|null;
  active:boolean;
  unit:string;
  quantity:string;
  minimumStock:string;
  status:InventoryStatus;
  updatedAt:string;
  quantityControl:"inventory";
};

export type InventoryList={
  items:InventoryItem[];
  total:number;
  page:number;
  pageSize:number;
};

export type InventoryProductOption={
  id:string;
  sku:string;
  name:string;
  categoryName:string|null;
  quantityControl:"inventory";
  unit:string|null;
  minimumStock:string|null;
};

export type InventoryEntryDraft={
  mode:"existing"|"new";
  productId:string;
  sku:string;
  name:string;
  description:string;
  price:string;
  quantity:string;
  unit:string;
  minimumStock:string;
  note:string;
};

export type StockMovement={
  id:string;
  productId:string;
  productName:string;
  movementType:"entry"|"sale"|"sale_reversal"|"sale_adjustment";
  quantityDelta:string;
  balanceAfter:string;
  sourceType:"inventory_entry"|"order";
  sourceId:string;
  note:string;
  createdAt:string;
};
