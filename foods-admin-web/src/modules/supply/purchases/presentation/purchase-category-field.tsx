"use client";
import {useState} from "react";
import {createPortal} from "react-dom";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,FormField,Select} from "@/design-system";
import {CategoryDialog,saveCategory,type Category,type CategoryDraft} from "@/modules/menu";
import {useSession} from "@/providers/session-context";
import {listPurchaseItemCategories} from "../infrastructure/purchases-api";
import type {PurchaseItemCategory} from "../domain/types";

export function PurchaseCategoryField({value,onChange,error,disabled,onBusyChange}:{value:string;onChange:(value:string)=>void;error?:string;disabled:boolean;onBusyChange:(busy:boolean)=>void}){
 const{organization,can}=useSession();
 const client=useQueryClient();
 const queryKey=["purchase-item-categories",organization?.id];
 const categories=useQuery({queryKey,queryFn:listPurchaseItemCategories,staleTime:30000});
 const[creating,setCreating]=useState(false);
 const[selected,setSelected]=useState<PurchaseItemCategory|null>(null);
 function close(){setCreating(false);onBusyChange(false)}
 const mutation=useMutation({mutationFn:saveCategory,retry:false,onSuccess:(category:Category)=>{
  setSelected(category);onChange(category.id);
  client.setQueryData<PurchaseItemCategory[]>(queryKey,current=>[...(current??[]).filter(item=>item.id!==category.id),category]);
  void client.invalidateQueries({queryKey});void client.invalidateQueries({queryKey:["categories"]});
  close();
 }});
 const items=categories.data??[];
 const options=selected&&!items.some(item=>item.id===selected.id)?[...items,selected]:items;
 const draft:CategoryDraft={name:"",sortOrder:0,active:true,productScope:"retail"};
 return <>
  <FormField label="Categoría" as="div" error={categories.isError?categories.error.message:error} help={!categories.isLoading&&!categories.isError&&!options.length?(can("menu.manage")?"Crea una categoría para mercadería vendible con +.":"Solicita una categoría para mercadería vendible al administrador."):undefined}>
   {categories.isLoading?<div className="purchase-reference-loading" role="status" aria-label="Cargando categorías"><i/></div>:<div className="purchase-reference-controls">
    <Select aria-label="Categoría" value={value} disabled={disabled||creating||categories.isError||!options.length} aria-invalid={Boolean(error)||categories.isError} onChange={event=>onChange(event.target.value)}>
     <option value="">{categories.isError?"No pudimos cargar las categorías":options.length?"Selecciona una categoría":"No hay categorías para mercadería vendible"}</option>
     {options.map(category=><option value={category.id} key={category.id}>{category.name}</option>)}
    </Select>
    {categories.isError?<Button type="button" kind="secondary" icon="refresh" disabled={disabled||categories.isFetching} onClick={()=>void categories.refetch()}>Reintentar</Button>:can("menu.manage")&&<Button type="button" kind="secondary" icon="plus" disabled={disabled||creating} onClick={()=>{mutation.reset();setCreating(true);onBusyChange(true)}} aria-label="Nueva categoría" title="Nueva categoría"/>}
   </div>}
  </FormField>
  {creating&&createPortal(<CategoryDialog draft={draft} retailOnly busy={mutation.isPending} error={mutation.error?.message} close={close} save={async value=>{try{await mutation.mutateAsync(value)}catch{/* Keep the category draft open and display the API error. */}}}/>,document.body)}
 </>;
}
