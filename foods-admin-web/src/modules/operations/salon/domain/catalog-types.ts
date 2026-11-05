export type CatalogProduct={id:string;name:string;description?:string;allergens?:string[];price:string;categoryId:string|null;imageUrl:string|null;availabilityStatus?:"available"|"low"|"sold_out"|"unavailable";remaining?:number|null;
 /** El plato pide opciones al tomarlo (término, acompañamiento, adicionales). */
 hasModifiers?:boolean};
/** Opción elegida de un plato simple; el recargo se suma al precio del plato. */
export type ModifierSelection={groupId:string;groupName:string;optionId:string;name:string;surcharge:number};
export type ProductModifierOption={id:string;name:string;surcharge:string};
export type ProductModifierGroup={id:string;name:string;required:boolean;minSelections:number;maxSelections:number;options:ProductModifierOption[]};
export type ProductModifierConfig={productId:string;groups:ProductModifierGroup[]};
export type ConfiguredProduct={productId:string;name:string;unitPrice:number;modifiers:ModifierSelection[];imageUrl?:string|null};
export type ComboSelection={groupId:string;groupName:string;productId:string;name:string;surcharge:number};
export type ConfiguredCombo={productId:string;name:string;unitPrice:number;selections:ComboSelection[];imageUrl?:string|null;availability?:ComboDetail};
export type ProductList={items:CatalogProduct[];total:number};
export type ComboSummary={id:string;name:string;description:string;price:string;imageUrl:string|null;groupCount:number;optionCount:number};
export type ComboList={items:ComboSummary[];total:number};
export type ComboOption={productId:string;name:string;surcharge:string;available:boolean;remaining?:number|null;quotaRemaining?:number|null};
export type ComboGroup={id:string;name:string;required:boolean;minSelections:number;maxSelections:number;options:ComboOption[]};
export type ComboDetail={id:string;name:string;description:string;price:string;imageUrl:string|null;groups:ComboGroup[]};
