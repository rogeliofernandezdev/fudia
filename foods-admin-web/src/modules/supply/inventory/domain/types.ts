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
  unitCountable:boolean;
  quantity:string;
  minimumStock:string;
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
  optimalStock:string;
  averageUnitCost:string;
};

export type InventoryAdjustmentDraft={
  inventoryItemId:string;
  movementType:InventoryAdjustmentType;
  reason:InventoryAdjustmentReason;
  quantity:string;
  observation:string;
  /** Costo por unidad base; solo aplica a entradas. Vacío usa el costo promedio vigente. */
  unitCost:string;
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
  unitCost:number;
  averageUnitCost:number;
  observation:string;
  createdAt:string;
  createdByName:string;
};

export type StockMovement={
  id:string;
  inventoryItemId:string;
  productId:string|null;
  itemName:string;
  unitName:string;
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


/** unit viaja solo cuando cambia: la ficha (nombre y unidad) es de la empresa. */
export type InventorySettingsDraft={inventoryItemId:string;name:string;unit?:string;minimumStock:string;optimalStock:string};
/** Estado del artículo en el local activo: el listado muestra los activos por defecto. */
export type InventoryStatusFilter="active"|"inactive"|"all";
export type InventoryItemPresentation={id:string;presentationType:string;unitsPerPresentation:string;name:string;isDefault:boolean;isBase:boolean;used:boolean};
export type InventoryItemLocation={locationId:string;name:string;active:boolean;quantity:string};
/** Ficha del artículo en el local activo, con las reglas de edición evaluadas por el backend. */
export type InventoryItemDetail={
  inventoryItemId:string;
  productId:string|null;
  productName:string|null;
  name:string;
  unit:string;
  unitName:string;
  unitCountable:boolean;
  active:boolean;
  quantity:string;
  minimumStock:string;
  optimalStock:string;
  hasHistory:boolean;
  recipes:Array<{productName:string;consumptionUnit:string}>;
  otherLocations:InventoryItemLocation[];
  presentations:InventoryItemPresentation[];
  canDelete:boolean;
  deleteBlocker?:string;
  canDeactivate:boolean;
  deactivateBlockers:string[];
};
/** free: sin historial; correction: las cantidades conservan sus números en la nueva unidad. */
export type InventoryUnitChangePreview={unit:string;unitName:string;mode:"same"|"free"|"correction";allowed:boolean;blockers:string[];quantity:string;fromUnit:string;toUnit:string};
export type InventoryUnitOption={id:string;code:string;name:string;countable:boolean};
export type InventoryTransferDraft={idempotencyKey:string;toLocationId:string;notes:string;items:Array<{inventoryItemId:string;quantity:string}>};
export type InventoryTransferSummary={id:string;code:string;fromLocationName:string;toLocationName:string;notes:string;createdByName:string;createdAt:string;itemCount:number};
export type LocationOption={id:string;name:string};
export type StockMovementsResponse={items:StockMovement[];total:number;page:number;pageSize:number};
