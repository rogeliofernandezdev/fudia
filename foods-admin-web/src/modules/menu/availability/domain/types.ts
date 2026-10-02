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
