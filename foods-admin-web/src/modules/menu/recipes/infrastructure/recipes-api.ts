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
export function listRecipeProducts(q=""){return apiFetch<{items:Array<{id:string;name:string;sku?:string;quantityControl:string}>}>(`products?status=active&q=${encodeURIComponent(q)}&page=1&pageSize=10`)}
export function listRecipeInventory(){return apiFetch<{items:Array<{id:string;name:string;kind:string;unit:string;quantity:string}>}>("inventory/products")}
