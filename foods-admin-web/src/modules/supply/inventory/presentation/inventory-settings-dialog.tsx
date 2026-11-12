"use client";
import {Dialog} from "@/design-system/dialog";
import {useRef} from "react";
import {useForm} from "react-hook-form";
import {Button,FormField,Input,ModalHeader,ModalFooter,FormSection} from "@/design-system";
import {inventorySettingsResolver} from "../domain/inventory-settings-schema";
import type {InventoryItem,InventorySettingsDraft} from "../domain/types";

const inputQuantity=(value:string)=>value.trim()!==""&&Number.isFinite(Number(value))?String(Number(value)):value;

export function InventorySettingsDialog({item,busy,close,save}:{item:InventoryItem;busy:boolean;close:()=>void;save:(draft:InventorySettingsDraft)=>Promise<void>}){
 const submitting=useRef(false);
 const{register,handleSubmit,setError,formState:{errors,isSubmitting}}=useForm<InventorySettingsDraft>({
  defaultValues:{inventoryItemId:item.inventoryItemId,name:item.name,minimumStock:inputQuantity(item.minimumStock),optimalStock:inputQuantity(item.optimalStock)},
  resolver:inventorySettingsResolver(item.unitCountable),
  mode:"onSubmit",
  reValidateMode:"onChange",
 });
 const locked=busy||isSubmitting;
 const quantityInput={step:item.unitCountable?"1":"0.001",inputMode:item.unitCountable?"numeric" as const:"decimal" as const};
 async function submit(){
  if(busy||submitting.current)return;
  submitting.current=true;
  try{
   await handleSubmit(async value=>{
    const min=Number(value.minimumStock),optimal=Number(value.optimalStock);
    await save({...value,name:value.name.trim(),optimalStock:String(Math.max(min,optimal))});
   })();
  }catch(error){
   if(error&&typeof error==="object"&&"code" in error&&error.code==="inventory_item_name_taken")setError("name",{type:"server",message:"Ya existe un artículo o producto con ese nombre. Elige otro nombre."},{shouldFocus:true});
  }finally{submitting.current=false;}
 }
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal compact modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="inventory-settings-title" aria-busy={locked}>

  <ModalHeader title="Editar artículo" titleId="inventory-settings-title" icon="edit" onClose={close} busy={locked} closeLabel="Cerrar"/>
  <form className="modal-form-compact" onSubmit={event=>event.preventDefault()} noValidate>
   <FormSection icon="stock" title="Datos del artículo" className="inventory-settings-section" bare><div className="form-grid">
    <FormField label="Nombre" className="span-2" error={errors.name?.message} help={item.productId?"Este nombre también se usa en Carta y productos.":undefined}><Input maxLength={160} disabled={locked} aria-invalid={Boolean(errors.name)} {...register("name")}/></FormField>
    <FormField label="Stock mínimo" icon="box" className="numeric-value" error={errors.minimumStock?.message}><Input type="number" min="0" {...quantityInput} disabled={locked} {...register("minimumStock")}/></FormField>
    <FormField label="Stock óptimo" icon="box" className="numeric-value" error={errors.optimalStock?.message}><Input type="number" min="0" {...quantityInput} disabled={locked} {...register("optimalStock")}/></FormField>
   </div></FormSection>
   <ModalFooter><Button action="cancel" onClick={close} disabled={locked}>Cancelar</Button><Button disabled={locked} onClick={submit} loading={locked}>{locked?"Guardando…":"Guardar"}</Button></ModalFooter>
  </form>
 </Dialog></div>
}
