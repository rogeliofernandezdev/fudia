export type Product={id:string;name:string;categoryName:string|null;price:string;active:boolean;defaultDailyQuota:number|null};
export type Option={productId:string;surcharge:string;quota:string};
export type Group={name:string;required:boolean;minSelections:number;maxSelections:number;options:Option[]};
export type Draft={name:string;description:string;price:string;groups:Group[];availableFrom:string;availableUntil:string;availableDays:number[]};
export type Combo={id:string;name:string;description:string;price:string;active:boolean;groupCount:number};
export type ComboDetailOption={productId:string;name:string;surcharge:string;quota:number|null};
export type ComboDetailGroup={id:string;name:string;required:boolean;minSelections:number;maxSelections:number;options:ComboDetailOption[]};
export type ComboDetail=Combo&{availableFrom:string|null;availableUntil:string|null;availableDays:number[]|null;groups:ComboDetailGroup[]};
