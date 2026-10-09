export type AvailabilityStatus="available"|"low"|"sold_out"|"unavailable";
export type QuantityControl="none"|"portions"|"inventory";
export type AvailabilityItem={
  productId:string;
  name:string;
  categoryName:string|null;
  imageUrl:string|null;
  quantityControl:QuantityControl;
  status:AvailabilityStatus;
  source:string;
  manualStatus:"available"|"sold_out";
  portionQuantity:number|null;
  soldQuantity:number;
  remaining:number|null;
  inventoryUnit:string|null;
  note:string;
  businessDate:string;
};
export type AvailabilityResponse={
  items:AvailabilityItem[];
  businessDate:string;
  total:number;
  page:number;
  pageSize:number;
};
export type CategoryOption={id:string;name:string};
export type AvailabilityChange={item:AvailabilityItem;status:"available"|"sold_out";portionQuantity:number|null;kind:"quota"|"status"};
export type AvailabilityReasonDraft={reason:string};
export type AvailabilityHistoryItem={
 id:string;createdAt:string;businessDate:string|null;userId:string|null;userName:string;reason:string|null;
 previousPortionQuantity:number|null;portionQuantity:number|null;soldQuantity:number|null;
 previousManualStatus:"available"|"sold_out"|null;manualStatus:"available"|"sold_out"|null;
};
export type AvailabilityHistoryResponse={items:AvailabilityHistoryItem[];total:number;page:number;pageSize:number};
