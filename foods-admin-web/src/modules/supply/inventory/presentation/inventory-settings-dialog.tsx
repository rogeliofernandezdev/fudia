"use client";
import {useState} from "react";
import {Button,Icon,Input} from "@/design-system";
import type {InventoryItem,InventorySettingsDraft} from "../domain/types";

export function InventorySettingsDialog({item,busy,close,save}:{item:InventoryItem;busy:boolean;close:()=>void;save:(draft:InventorySettingsDraft)=>void}){
 const[value,setValue]=useState<InventorySettingsDraft>({inventoryItemId:item.inventoryItemId,minimumStock:item.minimumStock,reorderPoint:item.reorderPoint,optimalStock:item.optimalStock});
 function submit(){
  const min=Number(value.minimumStock),reorder=Number(value.reorderPoint),optimal=Number(value.optimalStock);
  if([min,reorder,optimal].some(v=>!Number.isFinite(v)||v<0))return;
  save({...value,reorderPoint:String(Math.max(min,reorder)),optimalStock:String(Math.max(min,reorder,optimal))});
 }
 return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal compact modal-panel-in" role="dialog" aria-modal="true">
  <header><span className="modal-title-icon"><Icon name="settings" size={18}/></span><div><small>INVENTARIO · LOCAL ACTIVO</small><h2>{item.name}</h2></div><button onClick={close} aria-label="Cerrar"><Icon name="close"/></button></header>
  <div className="form-grid">
   <label>Stock mínimo<Input type="number" min="0" step="0.001" value={value.minimumStock} onChange={e=>setValue({...value,minimumStock:e.target.value})}/></label>
   <label>Punto de reorden<Input type="number" min="0" step="0.001" value={value.reorderPoint} onChange={e=>setValue({...value,reorderPoint:e.target.value})}/></label>
   <label className="span-2">Stock óptimo<Input type="number" min="0" step="0.001" value={value.optimalStock} onChange={e=>setValue({...value,optimalStock:e.target.value})}/></label>
  </div>
  <footer><Button kind="ghost" onClick={close}>Cancelar</Button><Button disabled={busy} onClick={submit}>{busy?"Guardando…":"Guardar niveles"}</Button></footer>
 </section></div>
}
