import type {Option,Product} from "./types";

/** Whole units the location has today; null means the product has no quantity control. */
export function availableUnits(product:Product):number|null{
  return product.availableQuantity==null?null:Math.max(0,Math.floor(product.availableQuantity));
}

/** A new option starts capped at today's available units; without quantity control it stays empty. */
export function createComboOption(product:Product):Option{
  const limit=availableUnits(product);
  return {productId:product.id,surcharge:"",quota:limit===null?"":String(limit)};
}

/** The cap never exceeds the units available today; empty sells everything available. */
export function isCapValid(quota:string,product:Product):boolean{
  const limit=availableUnits(product);
  if(limit===0)return false;
  if(!quota)return true;
  const value=Number(quota);
  return /^\d+$/.test(quota)&&Number.isSafeInteger(value)&&value>0&&(limit===null||value<=limit);
}
