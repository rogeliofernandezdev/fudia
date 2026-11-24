"use client";
import {Dialog} from "@/design-system/dialog";
import {useRef} from "react";
import {useQuery} from "@tanstack/react-query";
import {useForm,useWatch} from "react-hook-form";
import {Button,FormField,IconButton,InlineNotice,Input,ModalHeader,ModalFooter,ModalForm,ModalBody,FormSection,Select,Tag} from "@/design-system";
import {inventorySettingsResolver} from "../domain/inventory-settings-schema";
import {previewInventoryUnitChange} from "../infrastructure/inventory-api";
import type {InventoryItemDetail,InventoryItemPresentation,InventorySettingsDraft,InventoryUnitOption} from "../domain/types";
import {pluralUnit,unitQuantityLabel} from "../domain/unit-label";

const inputQuantity=(value:string)=>value.trim()!==""&&Number.isFinite(Number(value))?String(Number(value)):value;

/**
 * Edita la ficha (nombre y unidad, compartidos por la empresa) y los niveles del local.
 * Las reglas de unidad y presentaciones las evalúa el backend. Nada se elimina:
 * el artículo se desactiva por local desde la lista y las presentaciones se ocultan.
 */
export function InventorySettingsDialog({item,units,busy,close,save,removePresentation}:{
  item:InventoryItemDetail;
  units:InventoryUnitOption[];
  busy:boolean;
  close:()=>void;
  save:(draft:InventorySettingsDraft)=>Promise<void>;
  removePresentation:(presentation:InventoryItemPresentation)=>void;
}){
 const submitting=useRef(false);
 const unitOptions=units.some(unit=>unit.code===item.unit)?units:[{id:item.unit,code:item.unit,name:item.unitName,countable:item.unitCountable},...units];
 const countable=(code:string)=>unitOptions.find(unit=>unit.code===code)?.countable??item.unitCountable;
 const{control,register,handleSubmit,setError,formState:{errors,isSubmitting}}=useForm<InventorySettingsDraft>({
  defaultValues:{inventoryItemId:item.inventoryItemId,name:item.name,unit:item.unit,minimumStock:inputQuantity(item.minimumStock),optimalStock:inputQuantity(item.optimalStock)},
  resolver:(values,context,options)=>inventorySettingsResolver(countable(values.unit??item.unit))(values,context,options),
  mode:"onSubmit",
  reValidateMode:"onChange",
 });
 const unit=useWatch({control,name:"unit"})??item.unit;
 const name=useWatch({control,name:"name"})??item.name;
 const unitChanged=unit!==item.unit;
 const preview=useQuery({
  queryKey:["inventory-unit-change",item.inventoryItemId,unit],
  queryFn:()=>previewInventoryUnitChange(item.inventoryItemId,unit),
  enabled:unitChanged,
  staleTime:0,
  retry:false,
 });
 const unitBlocked=unitChanged&&(preview.isLoading||preview.isError||preview.data?.allowed===false);
 const locked=busy||isSubmitting;
 const sharedChange=(unitChanged||name.trim()!==item.name)&&item.otherLocations.length>0;
 const removable=item.presentations.filter(presentation=>!presentation.isBase);
 const quantityInput={step:countable(unit)?"1":"0.001",inputMode:countable(unit)?"numeric" as const:"decimal" as const};

 async function submit(){
  if(busy||unitBlocked||submitting.current)return;
  submitting.current=true;
  try{
   await handleSubmit(async value=>{
    const min=Number(value.minimumStock),optimal=Number(value.optimalStock);
    await save({...value,name:value.name.trim(),unit:unitChanged?value.unit:undefined,optimalStock:String(Math.max(min,optimal))});
   })();
  }catch(error){
   if(error&&typeof error==="object"&&"code" in error&&error.code==="inventory_item_name_taken")setError("name",{type:"server",message:"Ya existe un artículo o producto con ese nombre. Elige otro nombre."},{shouldFocus:true});
  }finally{submitting.current=false;}
 }
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal compact modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="inventory-settings-title" aria-busy={locked}>

  <ModalHeader title="Editar artículo" titleId="inventory-settings-title" onClose={close} busy={locked} closeLabel="Cerrar"/>
  <ModalForm onSubmit={event=>event.preventDefault()} noValidate>
   <ModalBody>
    <FormSection icon="stock" title="Datos del artículo" bare><div className="form-grid">
     <FormField label="Nombre" className="span-2" error={errors.name?.message} help={item.productId?"Este nombre también se usa en Carta y productos.":undefined}><Input autoFocus maxLength={160} disabled={locked} aria-invalid={Boolean(errors.name)} {...register("name")}/></FormField>
     <FormField label="Unidad" className="span-2"><Select disabled={locked} {...register("unit")}>{unitOptions.map(option=><option key={option.code} value={option.code}>{option.name}</option>)}</Select></FormField>
     {unitChanged&&<div className="span-2 inventory-unit-change" aria-live="polite">
      {preview.isLoading?<InlineNotice tone="info">Revisando el cambio de unidad…</InlineNotice>
      :preview.isError?<InlineNotice tone="danger" title="No pudimos revisar el cambio">{preview.error.message}</InlineNotice>
      :preview.data&&!preview.data.allowed?<InlineNotice tone="danger" title="No se puede cambiar la unidad"><>{preview.data.blockers.map(blocker=><span className="inventory-blocker" key={blocker}>{blocker}</span>)}</></InlineNotice>
      :preview.data?.mode==="correction"?<InlineNotice tone="warning" title="Corrección de unidad">Las cantidades no cambian: {unitQuantityLabel(preview.data.quantity,preview.data.fromUnit)} pasan a {unitQuantityLabel(preview.data.quantity,preview.data.toUnit)}. Las compras y el kárdex anteriores también se leerán en {pluralUnit(preview.data.toUnit)}.</InlineNotice>
      :null}
     </div>}
     <FormField label="Stock mínimo" icon="box" className="numeric-value" error={errors.minimumStock?.message}><Input type="number" min="0" {...quantityInput} disabled={locked} {...register("minimumStock")}/></FormField>
     <FormField label="Stock óptimo" icon="box" className="numeric-value" error={errors.optimalStock?.message}><Input type="number" min="0" {...quantityInput} disabled={locked} {...register("optimalStock")}/></FormField>
     {sharedChange&&<InlineNotice className="span-2" tone="info">También lo usa {item.otherLocations.map(location=>location.name).join(", ")}: el nombre y la unidad cambian en todos sus locales.</InlineNotice>}
    </div></FormSection>
    {removable.length>0&&<section className="inventory-presentations" aria-label="Presentaciones de compra">
     <b className="modal-section-title">Presentaciones de compra</b>
     <ul>{item.presentations.map(presentation=><li key={presentation.id}>
      <span>{presentation.name} · {unitQuantityLabel(presentation.unitsPerPresentation,item.unitName)}</span>
      {presentation.isBase?<Tag>Base</Tag>:<IconButton icon="trash" className="danger" label={`Quitar ${presentation.name} · ${unitQuantityLabel(presentation.unitsPerPresentation,item.unitName)}`} disabled={locked} onClick={()=>removePresentation(presentation)}/>}
     </li>)}</ul>
    </section>}
   </ModalBody>
   <ModalFooter>
    <Button action="cancel" onClick={close} disabled={locked}>Cancelar</Button>
    <Button disabled={locked||unitBlocked} onClick={submit} loading={locked}>{locked?"Guardando…":"Guardar"}</Button>
   </ModalFooter>
  </ModalForm>
 </Dialog></div>
}
