"use client";
import {Dialog} from "@/design-system/dialog";
import {useRef} from "react";
import {useForm} from "react-hook-form";
import {Button,FormField,Icon,Input} from "@/design-system";
import {inventorySettingsResolver} from "../domain/inventory-settings-schema";
import type {InventoryItem,InventorySettingsDraft} from "../domain/types";

export function InventorySettingsDialog({item,busy,close,save}:{item:InventoryItem;busy:boolean;close:()=>void;save:(draft:InventorySettingsDraft)=>Promise<void>}){
 const submitting=useRef(false);
 const{register,handleSubmit,formState:{errors,isSubmitting}}=useForm<InventorySettingsDraft>({
  defaultValues:{inventoryItemId:item.inventoryItemId,minimumStock:item.minimumStock,reorderPoint:item.reorderPoint,optimalStock:item.optimalStock},
  resolver:inventorySettingsResolver,
  mode:"onSubmit",
  reValidateMode:"onChange",
 });
 const locked=busy||isSubmitting;
 async function submit(){
  if(busy||submitting.current)return;
  submitting.current=true;
  try{
   await handleSubmit(async value=>{
    const min=Number(value.minimumStock),reorder=Number(value.reorderPoint),optimal=Number(value.optimalStock);
    await save({...value,reorderPoint:String(Math.max(min,reorder)),optimalStock:String(Math.max(min,reorder,optimal))});
   })();
  }finally{submitting.current=false;}
 }
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal compact modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="inventory-settings-title" aria-busy={locked}>
  <div className="modal-accent"/>
  <header><span className="modal-title-icon"><Icon name="settings" size={18}/></span><div><small>INVENTARIO · LOCAL ACTIVO</small><h2 id="inventory-settings-title">{item.name}</h2></div><button type="button" onClick={close} disabled={locked} aria-label="Cerrar"><Icon name="close"/></button></header>
  <form onSubmit={event=>event.preventDefault()} noValidate>
   <div className="form-grid">
    <FormField label="Stock mínimo" error={errors.minimumStock?.message}><Input type="number" min="0" step="0.001" inputMode="decimal" disabled={locked} {...register("minimumStock")}/></FormField>
    <FormField label="Punto de reorden" error={errors.reorderPoint?.message}><Input type="number" min="0" step="0.001" inputMode="decimal" disabled={locked} {...register("reorderPoint")}/></FormField>
    <FormField className="span-2" label="Stock óptimo" error={errors.optimalStock?.message}><Input type="number" min="0" step="0.001" inputMode="decimal" disabled={locked} {...register("optimalStock")}/></FormField>
   </div>
   <footer><Button kind="ghost" onClick={close} disabled={locked}>Cancelar</Button><Button disabled={locked} onClick={submit}>{locked?"Guardando…":"Guardar"}</Button></footer>
  </form>
 </Dialog></div>
}
