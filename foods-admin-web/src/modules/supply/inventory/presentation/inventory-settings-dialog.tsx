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
 const{register,handleSubmit,formState:{errors,isSubmitting}}=useForm<InventorySettingsDraft>({
  defaultValues:{inventoryItemId:item.inventoryItemId,minimumStock:inputQuantity(item.minimumStock),optimalStock:inputQuantity(item.optimalStock)},
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
    await save({...value,optimalStock:String(Math.max(min,optimal))});
   })();
  }finally{submitting.current=false;}
 }
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal compact modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="inventory-settings-title" aria-busy={locked}>

  <ModalHeader title={item.name} titleId="inventory-settings-title" icon="settings" subtitle="Inventario · local activo" onClose={close} busy={locked} closeLabel="Cerrar"/>
  <form className="modal-form-compact" onSubmit={event=>event.preventDefault()} noValidate>
   <FormSection icon="gauge" tone="digital" title="Niveles de stock" className="inventory-settings-section" bare><div className="form-grid">
    <FormField label="Stock mínimo" icon="box" className="numeric-value" error={errors.minimumStock?.message}><Input type="number" min="0" {...quantityInput} disabled={locked} {...register("minimumStock")}/></FormField>
    <FormField label="Stock óptimo" icon="box" className="numeric-value" error={errors.optimalStock?.message}><Input type="number" min="0" {...quantityInput} disabled={locked} {...register("optimalStock")}/></FormField>
   </div></FormSection>
   <ModalFooter><Button action="cancel" onClick={close} disabled={locked}>Cancelar</Button><Button disabled={locked} onClick={submit} loading={locked}>{locked?"Guardando…":"Guardar"}</Button></ModalFooter>
  </form>
 </Dialog></div>
}
