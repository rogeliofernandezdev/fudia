import type {Option,Product} from "./types";

/** A menu option is selected in whole units; null means no quantity control. */
export function reservationLimit(product:Product):number|null{
  return product.availableQuantity==null?null:Math.max(0,Math.floor(product.availableQuantity));
}

export function createComboOption(product:Product):Option{
  const limit=reservationLimit(product);
  return {productId:product.id,surcharge:"",quota:limit===null?"":String(limit)};
}

export function isReservationValid(quota:string,product:Product):boolean{
  if(!quota)return true;
  const value=Number(quota),limit=reservationLimit(product);
  return /^\d+$/.test(quota)&&Number.isSafeInteger(value)&&(limit===null||value<=limit);
}
