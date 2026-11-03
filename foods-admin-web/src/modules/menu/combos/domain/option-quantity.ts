import type {Option,Product} from "./types";

/** Whole units the location has today; null means the product has no quantity control. */
export function availableUnits(product:Product):number|null{
  return product.availableQuantity==null?null:Math.max(0,Math.floor(product.availableQuantity));
}

/** A new option starts without a daily cap: it sells until the shared stock runs out. */
export function createComboOption(product:Product):Option{
  return {productId:product.id,surcharge:"",quota:""};
}

/** The cap only limits this menu per day; it reserves nothing, so today's stock is not its upper bound. */
export function isCapValid(quota:string,product:Product):boolean{
  if(availableUnits(product)===0)return false;
  if(!quota)return true;
  const value=Number(quota);
  return /^\d+$/.test(quota)&&Number.isSafeInteger(value)&&value>0;
}
