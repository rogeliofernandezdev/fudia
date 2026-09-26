import {apiFetch} from "@/shared/api/client";

export type RecipeIngredient={id?:string;inventoryItemId:string;name?:string;unit?:string;quantity:string;wastePercent:string};
export type Recipe={id:string;productId:string;productName:string;yieldQuantity:string;notes:string;active:boolean;updatedAt:string;items:RecipeIngredient[]};
export type RecipeSummary={id:string;productId:string;productName:string;yieldQuantity:string;notes:string;active:boolean;updatedAt:string;itemCount:number};
export type RecipeDraft={productId:string;yieldQuantity:string;notes:string;active:boolean;items:Array<{inventoryItemId:string;quantity:string;wastePercent:string}>};

export function listRecipes(q=""){return apiFetch<{items:RecipeSummary[];total:number}>(`recipes?q=${encodeURIComponent(q)}&page=1&pageSize=100`)}
export function getRecipe(productId:string){return apiFetch<Recipe>(`recipes/${productId}`)}
export function saveRecipe(draft:RecipeDraft){return apiFetch<{id:string;productId:string;productName:string}>(`recipes/${draft.productId}`,{method:"PUT",body:JSON.stringify({
 yieldQuantity:Number(draft.yieldQuantity),notes:draft.notes.trim(),active:draft.active,
 items:draft.items.map(i=>({inventoryItemId:i.inventoryItemId,quantity:Number(i.quantity),wastePercent:Number(i.wastePercent||0)})),
})})}
type RecipeProductLookup={items:Array<{id:string;name:string;quantityControl:string}>;total:number;page:number;pageSize:number};
export async function listRecipeProducts(q=""){
 const search=q.trim();
 if(!search)return apiFetch<RecipeProductLookup>("products?status=active&page=1&pageSize=10");
 const first=await apiFetch<RecipeProductLookup>(`products?status=active&q=${encodeURIComponent(search)}&page=1&pageSize=100`);
 if(first.items.length>=first.total)return first;
 const pages=Math.ceil(first.total/100);
 const rest=await Promise.all(Array.from({length:pages-1},(_,index)=>
  apiFetch<RecipeProductLookup>(`products?status=active&q=${encodeURIComponent(search)}&page=${index+2}&pageSize=100`)
 ));
 return {...first,items:[...first.items,...rest.flatMap(page=>page.items)]};
}
type RecipeInventoryLookup={items:Array<{id:string;name:string;kind:string;unit:string;quantity:string}>;total:number;page:number;pageSize:number};
export async function listRecipeInventory(q=""){
 const search=q.trim();
 if(!search)return apiFetch<RecipeInventoryLookup>("inventory/products?page=1&pageSize=10");
 const first=await apiFetch<RecipeInventoryLookup>(`inventory/products?q=${encodeURIComponent(search)}&page=1&pageSize=100`);
 if(first.items.length>=first.total)return first;
 const pages=Math.ceil(first.total/100);
 const rest=await Promise.all(Array.from({length:pages-1},(_,index)=>
  apiFetch<RecipeInventoryLookup>(`inventory/products?q=${encodeURIComponent(search)}&page=${index+2}&pageSize=100`)
 ));
 return {...first,items:[...first.items,...rest.flatMap(page=>page.items)]};
}
