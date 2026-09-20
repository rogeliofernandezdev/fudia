import {apiFetch} from "@/shared/api/client";
import type {Category,CategoryDraft,List,Product,ProductDraft} from "../domain/types";

export type ProductQuery={
  search:string;
  categoryId:string;
  status:string;
  page:number;
  pageSize:number;
};

export function listCategories(page:number,pageSize:number){
  const params=new URLSearchParams({includeInactive:"true",page:String(page),pageSize:String(pageSize)});
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
  if(file&&product.id){
    const form=new FormData();
    form.append("file",file);
    const response=await fetch(`/api/admin/products/${product.id}/image`,{method:"POST",body:form});
    if(!response.ok){
      const body=await response.json().catch(()=>({}));
      throw new Error(body.message??"No pudimos subir la imagen del producto.");
    }
  }
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
