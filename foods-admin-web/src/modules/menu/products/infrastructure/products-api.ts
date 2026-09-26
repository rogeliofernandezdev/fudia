import {apiFetch} from "@/shared/api/client";
import {uploadProductImage} from "@/shared/api/product-image";
import type {Category,CategoryDraft,List,Product,ProductDraft,ProductType} from "../domain/types";

export type ProductQuery={
  search:string;
  categoryId:string;
  status:string;
  page:number;
  pageSize:number;
};

export function listCategories(page:number,pageSize:number,productType?:ProductType){
  const params=new URLSearchParams({includeInactive:"true",page:String(page),pageSize:String(pageSize)});
  if(productType)params.set("productType",productType);
  return apiFetch<List<Category>>(`categories?${params.toString()}`);
}

export function listProducts(query:ProductQuery){
  const params=new URLSearchParams({
    q:query.search,
    categoryId:query.categoryId,
    status:query.status,
    page:String(query.page),
    pageSize:String(query.pageSize),
  });
  return apiFetch<List<Product>>(`products?${params.toString()}`);
}

export async function saveProduct(draft:ProductDraft,file:File|null){
  const product=await apiFetch<Product>(draft.id?`products/${draft.id}`:"products",{
    method:draft.id?"PATCH":"POST",
    body:JSON.stringify({
      ...draft,
      categoryId:draft.categoryId||null,
      prepMinutes:draft.prepMinutes?Number(draft.prepMinutes):null,
      costPrice:draft.costPrice||null,
    }),
  });
  if(file&&product.id)await uploadProductImage(product.id,file);
  return product;
}

export function saveCategory(draft:CategoryDraft){
  return apiFetch<Category>(draft.id?`categories/${draft.id}`:"categories",{
    method:draft.id?"PATCH":"POST",
    body:JSON.stringify(draft),
  });
}

export function deactivateCatalogItem(kind:"products"|"categories",id:string){
  return apiFetch<void>(`${kind}/${id}`,{method:"DELETE"});
}

export async function listAllergens(q:string){
  const params=new URLSearchParams({q});
  const response=await apiFetch<{items:string[]}>(`allergens?${params.toString()}`);
  return response.items;
}
