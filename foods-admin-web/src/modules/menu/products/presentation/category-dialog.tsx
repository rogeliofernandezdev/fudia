"use client";
import {useRef} from "react";
import {useForm} from "react-hook-form";
import {Button,FormField,Input,ModalHeader,ModalFooter} from "@/design-system";
import {Dialog} from "@/design-system/dialog";
import type {CategoryDraft} from "../domain/types";
import {categoryResolver} from "../domain/product-schema";

/** Categoría: solo nombre y orden. Su uso (Carta o Compras) llega en el borrador según el formulario de origen y el backend lo amplía a «Ambos» al recibir productos del otro tipo. */
export function CategoryDialog({draft,busy,close,save,error}:{draft:CategoryDraft;busy:boolean;close:()=>void;save:(draft:CategoryDraft)=>Promise<void>;error?:string}){
 const submitting=useRef(false);
 const{register,handleSubmit,formState:{errors,isSubmitting}}=useForm<CategoryDraft>({defaultValues:draft,resolver:categoryResolver,mode:"onSubmit",reValidateMode:"onChange"});
 const locked=busy||isSubmitting;
 async function submit(){
  if(busy||submitting.current)return;
  submitting.current=true;
  try{await handleSubmit(values=>save({...values,productScope:draft.productScope,sortOrder:Number(values.sortOrder)}))()}
  finally{submitting.current=false}
 }
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal compact modal-panel-in" aria-labelledby="category-title" aria-busy={locked}>

  <ModalHeader title="Datos de la categoría" titleId="category-title" icon="menu" subtitle={draft.id?"EDITAR CATEGORÍA":"NUEVA CATEGORÍA"} onClose={close} busy={locked} closeLabel="Cerrar"/>
  <form onSubmit={event=>{event.preventDefault();event.stopPropagation()}} noValidate><div className="form-grid" inert={busy}>
   <FormField className="span-2" label="Nombre" error={errors.name?.message}><Input data-dialog-initial-focus maxLength={120} {...register("name")} placeholder="Ej. Bebidas"/></FormField>
   <FormField className="span-2" label="Orden de aparición" help="Menor número aparece primero en la carta." error={errors.sortOrder?.message}><Input type="number" min="0" inputMode="numeric" {...register("sortOrder")}/></FormField>
   {error&&<p className="field-error span-2" role="alert">{error}</p>}
  </div><ModalFooter><Button type="button" kind="ghost" onClick={close} disabled={locked}>Cancelar</Button><Button type="button" onClick={()=>void submit()} disabled={locked} loading={locked}>{locked?"Guardando…":"Guardar"}</Button></ModalFooter></form>
 </Dialog></div>;
}
