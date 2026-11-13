"use client";
import {Dialog} from "@/design-system/dialog";
import {useState} from "react";
import {Controller,useForm,useWatch} from "react-hook-form";
import {Button,FormField,Icon,Input,MoneyInput,Select,Tabs,Textarea,ModalBody,ModalFooter,ModalForm,ModalHeader,type TabItem} from "@/design-system";
import {useSession} from "@/providers/session-context";
import {useCurrency} from "@/providers/use-money";
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
// Entrada y Salida cambian los campos del formulario: se eligen con las pestañas compartidas.
const movementTabs:TabItem<InventoryAdjustmentType>[]=[
  {value:"entry",label:"Entrada",icon:"plus"},
  {value:"exit",label:"Salida",icon:"minus"},
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
  const currency=useCurrency();
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
  const amount=(value:number)=>formatRegionalNumber(value,location?.country,{maximumFractionDigits:3});
  const money=(value:number)=>formatRegionalNumber(value,location?.country,{minimumFractionDigits:2,maximumFractionDigits:4});
  const hasErrors=Object.keys(errors).length>0;
  const costHelp=!selected?undefined:averageCost>0?"Vacío usa el promedio actual: "+money(averageCost):"Obligatorio: el artículo aún no tiene costo promedio.";

  function changeMovement(type:InventoryAdjustmentType){
    const nextReasons=type==="entry"?entryReasons:exitReasons;
    setValue("movementType",type,{shouldDirty:true,shouldValidate:isSubmitted});
    setValue("reason",nextReasons[0].value,{shouldDirty:true,shouldValidate:isSubmitted});
    if(type==="exit")setValue("unitCost","",{shouldValidate:isSubmitted});
  }

  function submit(draft:InventoryAdjustmentDraft){
    const total=Number(draft.quantity);
    if(draft.movementType==="exit"&&selected&&total>Number(selected.quantity)){
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

      <ModalHeader title="Registrar ajuste" titleId="inventory-adjustment-title" onClose={close} busy={busy} closeLabel="Cerrar"/>
      <ModalForm onSubmit={handleSubmit(submit)} noValidate>
        <ModalBody>
          <div className="inventory-adjustment-fields">
            <FormField as="div" className="span-2" label="Artículo" error={errors.inventoryItemId?.message} help={selected?`Stock actual: ${amount(currentStock)} ${selected.unit}`:undefined}>
              <Controller control={control} name="inventoryItemId" render={({field})=><InventoryItemLookup autoFocus ariaLabel="Artículo" placeholder="Buscar artículo existente..." value={selected} invalid={Boolean(errors.inventoryItemId)} disabled={busy} onChange={item=>{setSelected(item);field.onChange(item?.id??"")}}/>}/>
            </FormField>

            <Tabs className="span-2" label="Tipo de movimiento" value={movementType} onChange={changeMovement} items={movementTabs}/>

            <FormField label="Motivo" error={errors.reason?.message}>
              <Select {...register("reason")}>
                {reasons.map(reason=><option value={reason.value} key={reason.value}>{reason.label}</option>)}
              </Select>
            </FormField>
            <FormField label={selected?`Cantidad (${selected.unit})`:"Cantidad"} error={errors.quantity?.message}>
              <Input type="number" min="0.001" step="0.001" inputMode="decimal" {...register("quantity")} placeholder="0"/>
            </FormField>
            {movementType==="entry"&&<FormField label={selected?`Costo unitario por ${selected.unit}`:"Costo unitario"} error={errors.unitCost?.message} help={costHelp}>
              <MoneyInput symbol={currency.symbol} position={currency.position} {...register("unitCost")} placeholder={averageCost>0?money(averageCost):"0.00"}/>
            </FormField>}
            <FormField className="span-2" label="Observación" optional error={errors.observation?.message}>
              <Textarea maxLength={240} rows={3} {...register("observation")} placeholder="Agrega una referencia si necesitas explicar el ajuste"/>
            </FormField>

            {selected&&Number.isFinite(quantity)&&quantity>0&&<p className="inventory-adjustment-result span-2">
              <span>Stock resultante</span>
              <strong>{amount(Math.max(0,projected))} {selected.unit}</strong>
              <small>{amount(currentStock)} {movementType==="entry"?"+":"−"} {amount(quantity)}</small>
            </p>}

            {isSubmitted&&hasErrors&&<div className="inventory-validation span-2" role="alert"><Icon name="alert" size={16}/><span>Revisa los campos marcados antes de guardar.</span></div>}
          </div>
        </ModalBody>
        <ModalFooter><Button type="button" action="cancel" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy} loading={busy}>{busy?"Guardando…":"Guardar"}</Button></ModalFooter>
      </ModalForm>
    </Dialog>
  </div>;
}
