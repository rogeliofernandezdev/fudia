"use client";
import {useRef} from "react";
import {useForm} from "react-hook-form";
import {Button,FormField,Icon,Input,Select} from "@/design-system";
import {Dialog} from "@/design-system/dialog";
import type {CategoryDraft} from "../domain/types";
import {categoryResolver} from "../domain/product-schema";

export function CategoryDialog({draft,busy,close,save,retailOnly=false,error}:{draft:CategoryDraft;busy:boolean;close:()=>void;save:(draft:CategoryDraft)=>Promise<void>;retailOnly?:boolean;error?:string}){
 const submitting=useRef(false);
 const{register,handleSubmit,formState:{errors,isSubmitting}}=useForm<CategoryDraft>({defaultValues:draft,resolver:categoryResolver,mode:"onSubmit",reValidateMode:"onChange"});
 const locked=busy||isSubmitting;
 async function submit(){
  if(busy||submitting.current)return;
  submitting.current=true;
  try{await handleSubmit(values=>save({...values,sortOrder:Number(values.sortOrder)}))()}
  finally{submitting.current=false}
 }
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal compact modal-panel-in" aria-labelledby="category-title" aria-busy={locked}>
  <div className="modal-accent"/>
  <header><span className="modal-title-icon"><Icon name="menu" size={18}/></span><div><small>{draft.id?"EDITAR CATEGORÍA":"NUEVA CATEGORÍA"}</small><h2 id="category-title">Datos de la categoría</h2></div><button type="button" onClick={close} disabled={locked} aria-label="Cerrar"><Icon name="close"/></button></header>
  <form onSubmit={event=>{event.preventDefault();event.stopPropagation()}} noValidate><div className="form-grid" inert={busy}>
   <FormField className="span-2" label="Nombre" error={errors.name?.message}><Input data-dialog-initial-focus maxLength={120} {...register("name")} placeholder="Ej. Bebidas"/></FormField>
   <FormField className="span-2" label="Disponible para" error={errors.productScope?.message}><Select {...register("productScope")}>
    {!retailOnly&&<option value="prepared">Platos y preparados</option>}<option value="retail">Mercadería vendible</option><option value="both">Ambos</option>
   </Select></FormField>
   <FormField className="span-2" label="Orden de aparición" help="Menor número aparece primero en la carta." error={errors.sortOrder?.message}><Input type="number" min="0" inputMode="numeric" {...register("sortOrder")}/></FormField>
   {error&&<p className="field-error span-2" role="alert">{error}</p>}
  </div><footer><Button type="button" kind="ghost" onClick={close} disabled={locked}>Cancelar</Button><Button type="button" onClick={()=>void submit()} disabled={locked}>{locked?"Guardando…":"Guardar"}</Button></footer></form>
  {busy&&<div className="modal-busy" role="status"><i/><span>Guardando…</span></div>}
 </Dialog></div>;
}
