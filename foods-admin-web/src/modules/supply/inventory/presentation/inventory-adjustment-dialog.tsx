"use client";
import {Dialog} from "@/design-system/dialog";
import {useState} from "react";
import {Controller,useForm,useWatch} from "react-hook-form";
import {Button,Icon,Input,Select,Textarea,ModalHeader,ModalFooter,FormSection,ChoiceGroup} from "@/design-system";
import {useSession} from "@/providers/session-context";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import {inventoryAdjustmentResolver} from "../domain/inventory-adjustment-schema";
import {InventoryItemLookup} from "./inventory-item-lookup";
import type {InventoryAdjustmentDraft,InventoryAdjustmentReason,InventoryAdjustmentType,InventoryProductOption} from "../domain/types";

const entryReasons:Array<{value:InventoryAdjustmentReason;label:string}>=[
  {value:"surplus_adjustment",label:"Ajuste por sobrante"},
];
const exitReasons:Array<{value:InventoryAdjustmentReason;label:string}>=[
  {value:"shortage_adjustment",label:"Ajuste por faltante"},
  {value:"waste",label:"Merma"},
  {value:"expiration",label:"Vencimiento"},
  {value:"other_exit",label:"Otro motivo de salida"},
];

const defaults:InventoryAdjustmentDraft={
  inventoryItemId:"",
  movementType:"entry",
  reason:"surplus_adjustment",
  quantity:"",
  observation:"",
  unitCost:"",
};

export function InventoryAdjustmentDialog({busy,close,save}:{busy:boolean;close:()=>void;save:(draft:InventoryAdjustmentDraft)=>void}){
  const{location}=useSession();
  const[selected,setSelected]=useState<InventoryProductOption|null>(null);
  const{
    control,register,handleSubmit,setValue,setError,
    formState:{errors,isSubmitted},
  }=useForm<InventoryAdjustmentDraft>({
    defaultValues:defaults,
    resolver:inventoryAdjustmentResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });

  const movementType=useWatch({control,name:"movementType"})??"entry";
  const quantityValue=useWatch({control,name:"quantity"});
  const reasons=movementType==="entry"?entryReasons:exitReasons;
  const currentStock=Number(selected?.quantity??0);
  const quantity=Number(quantityValue||0);
  const projected=movementType==="entry"?currentStock+quantity:currentStock-quantity;
  const averageCost=Number(selected?.averageUnitCost??0);
  const money=(value:number)=>formatRegionalNumber(value,location?.country,{minimumFractionDigits:2,maximumFractionDigits:4});
  const hasErrors=Object.keys(errors).length>0;

  function changeMovement(type:InventoryAdjustmentType){
    const nextReasons=type==="entry"?entryReasons:exitReasons;
    setValue("movementType",type,{shouldDirty:true,shouldValidate:isSubmitted});
    setValue("reason",nextReasons[0].value,{shouldDirty:true,shouldValidate:isSubmitted});
    if(type==="exit")setValue("unitCost","",{shouldValidate:isSubmitted});
  }

  function submit(draft:InventoryAdjustmentDraft){
    const amount=Number(draft.quantity);
    if(draft.movementType==="exit"&&selected&&amount>Number(selected.quantity)){
      setError("quantity",{type:"manual",message:"La salida no puede superar el stock actual."});
      return;
    }
    // Without an average, the entry would be worth zero: the cost is required.
    if(draft.movementType==="entry"&&selected&&averageCost<=0&&draft.unitCost.trim()===""){
      setError("unitCost",{type:"manual",message:"Ingresa el costo unitario: el artículo todavía no tiene costo promedio."});
      return;
    }
    save(draft);
  }

  return <div className="modal-backdrop modal-overlay-in" role="presentation">
    <Dialog onResponseClose={close} className="crud-modal inventory-adjustment-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="inventory-adjustment-title" aria-busy={busy}>

      <ModalHeader title="Registrar ajuste" titleId="inventory-adjustment-title" icon="stock" subtitle="Inventario" onClose={close} busy={busy} closeLabel="Cerrar"/>
      <form onSubmit={handleSubmit(submit)} noValidate>
        <div className="inventory-adjustment-body">
          <FormSection icon="stock" title="Artículo" bare>
          <div className="inventory-adjustment-field">Artículo
            <Controller control={control} name="inventoryItemId" render={({field})=><InventoryItemLookup autoFocus ariaLabel="Artículo" placeholder="Buscar artículo existente..." value={selected} invalid={Boolean(errors.inventoryItemId)} disabled={busy} onChange={item=>{setSelected(item);field.onChange(item?.id??"")}}/>}/>
            {errors.inventoryItemId?.message&&<small className="field-error">{errors.inventoryItemId.message}</small>}
          </div>

          {selected&&<section className="inventory-adjustment-balance" aria-label="Saldo actual">
            <div><small>Stock actual</small><strong>{formatRegionalNumber(currentStock,location?.country,{maximumFractionDigits:3})} {selected.unit}</strong></div>
            <div><small>Unidad de inventario</small><strong>{selected.unit}</strong></div>
          </section>}

          </FormSection>
          <FormSection icon="transfer" tone="digital" title="Movimiento" bare>
          <div className="inventory-adjustment-grid">
            <div className="span-2 inventory-adjustment-type"><span>Tipo de movimiento</span>
              <ChoiceGroup label="Tipo de movimiento" name="movementType" value={movementType} onChange={event=>changeMovement(event.target.value as InventoryAdjustmentType)} options={[{value:"entry",label:"Entrada",icon:"plus"},{value:"exit",label:"Salida",icon:"minus"}]}/>
            </div>
            <label>Motivo
              <Select icon="note" {...register("reason")} aria-invalid={Boolean(errors.reason)}>
                {reasons.map(reason=><option value={reason.value} key={reason.value}>{reason.label}</option>)}
              </Select>
              {errors.reason?.message&&<small className="field-error">{errors.reason.message}</small>}
            </label>
            <label>Cantidad{selected?" ("+selected.unit+")":""}
              <Input icon="gauge" type="number" min="0.001" step="0.001" inputMode="decimal" {...register("quantity")} placeholder="0" aria-invalid={Boolean(errors.quantity)}/>
              {errors.quantity?.message&&<small className="field-error">{errors.quantity.message}</small>}
            </label>
            {movementType==="entry"&&<label>Costo unitario{selected?" por "+selected.unit:""}
              <Input icon="cash" type="number" min="0" step="0.0001" inputMode="decimal" {...register("unitCost")} placeholder={averageCost>0?money(averageCost):"0.00"} aria-invalid={Boolean(errors.unitCost)}/>
              {errors.unitCost?.message?<small className="field-error">{errors.unitCost.message}</small>:selected&&<small className="inventory-adjustment-hint">{averageCost>0?"Vacío usa el promedio actual: "+money(averageCost):"Obligatorio: el artículo aún no tiene costo promedio."}</small>}
            </label>}
            <label className="span-2">Observación opcional
              <Textarea maxLength={240} rows={3} {...register("observation")} placeholder="Agrega una referencia si necesitas explicar el ajuste"/>
              {errors.observation?.message&&<small className="field-error">{errors.observation.message}</small>}
            </label>
          </div>
          </FormSection>

          {selected&&Number.isFinite(quantity)&&quantity>0&&<div className={"inventory-adjustment-preview "+movementType}>
            <Icon name={movementType==="entry"?"plus":"minus"} size={16}/>
            <span><b>Stock resultante: {formatRegionalNumber(Math.max(0,projected),location?.country,{maximumFractionDigits:3})} {selected.unit}</b><small>Actual {formatRegionalNumber(currentStock,location?.country,{maximumFractionDigits:3})} {selected.unit} · {movementType==="entry"?"+":"-"}{formatRegionalNumber(quantity,location?.country,{maximumFractionDigits:3})} {selected.unit}</small></span>
          </div>}

          {isSubmitted&&hasErrors&&<div className="inventory-validation" role="alert"><Icon name="alert" size={16}/><span>Revisa los campos marcados antes de guardar.</span></div>}
        </div>
        <ModalFooter><Button type="button" action="cancel" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy} loading={busy}>{busy?"Guardando…":"Guardar"}</Button></ModalFooter>
      </form>
    </Dialog>
  </div>;
}
