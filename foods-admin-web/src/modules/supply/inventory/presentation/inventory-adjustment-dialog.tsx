"use client";
import {useMemo} from "react";
import {useForm,useWatch} from "react-hook-form";
import {Button,Icon,Input,Select,Textarea} from "@/design-system";
import {useSession} from "@/providers/session-context";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import {inventoryAdjustmentResolver} from "../domain/inventory-adjustment-schema";
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
};

export function InventoryAdjustmentDialog({items,busy,close,save}:{items:InventoryProductOption[];busy:boolean;close:()=>void;save:(draft:InventoryAdjustmentDraft)=>void}){
  const{location}=useSession();
  const{
    control,register,handleSubmit,setValue,setError,
    formState:{errors,isSubmitted},
  }=useForm<InventoryAdjustmentDraft>({
    defaultValues:defaults,
    resolver:inventoryAdjustmentResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });

  const inventoryItemId=useWatch({control,name:"inventoryItemId"});
  const movementType=useWatch({control,name:"movementType"})??"entry";
  const quantityValue=useWatch({control,name:"quantity"});
  const selected=useMemo(()=>items.find(item=>item.id===inventoryItemId),[items,inventoryItemId]);
  const reasons=movementType==="entry"?entryReasons:exitReasons;
  const currentStock=Number(selected?.quantity??0);
  const quantity=Number(quantityValue||0);
  const projected=movementType==="entry"?currentStock+quantity:currentStock-quantity;
  const hasErrors=Object.keys(errors).length>0;

  function changeMovement(type:InventoryAdjustmentType){
    const nextReasons=type==="entry"?entryReasons:exitReasons;
    setValue("movementType",type,{shouldDirty:true,shouldValidate:isSubmitted});
    setValue("reason",nextReasons[0].value,{shouldDirty:true,shouldValidate:isSubmitted});
  }

  function submit(draft:InventoryAdjustmentDraft){
    const item=items.find(option=>option.id===draft.inventoryItemId);
    const amount=Number(draft.quantity);
    if(draft.movementType==="exit"&&item&&amount>Number(item.quantity)){
      setError("quantity",{type:"manual",message:"La salida no puede superar el stock actual."});
      return;
    }
    save(draft);
  }

  return <div className="modal-backdrop modal-overlay-in" role="presentation">
    <section className="crud-modal inventory-adjustment-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="inventory-adjustment-title" aria-busy={busy}>
      <div className="modal-accent"/>
      <header>
        <span className="modal-title-icon"><Icon name="stock" size={18}/></span>
        <div><small>INVENTARIO</small><h2 id="inventory-adjustment-title">Registrar ajuste</h2></div>
        <button type="button" aria-label="Cerrar" onClick={close} disabled={busy}><Icon name="close"/></button>
      </header>
      <form onSubmit={handleSubmit(submit)} noValidate>
        <div className="inventory-adjustment-body">
          <label className="inventory-adjustment-field">Artículo
            <Select autoFocus {...register("inventoryItemId")} aria-invalid={Boolean(errors.inventoryItemId)}>
              <option value="">{items.length?"Selecciona un artículo existente":"No hay artículos disponibles"}</option>
              {items.map(item=><option value={item.id} key={item.id}>{item.name}{item.kind==="ingredient"?" · Insumo":""}</option>)}
            </Select>
            {errors.inventoryItemId?.message&&<small className="wizard-field-error">{errors.inventoryItemId.message}</small>}
          </label>

          {selected&&<section className="inventory-adjustment-balance" aria-label="Saldo actual">
            <div><small>STOCK ACTUAL</small><strong>{formatRegionalNumber(currentStock,location?.country,{maximumFractionDigits:3})} {selected.unit}</strong></div>
            <div><small>UNIDAD BASE</small><strong>{selected.unit}</strong></div>
          </section>}

          <div className="inventory-adjustment-grid">
            <label>Tipo de movimiento
              <Select value={movementType} onChange={event=>changeMovement(event.target.value as InventoryAdjustmentType)}>
                <option value="entry">Entrada</option>
                <option value="exit">Salida</option>
              </Select>
            </label>
            <label>Motivo
              <Select {...register("reason")} aria-invalid={Boolean(errors.reason)}>
                {reasons.map(reason=><option value={reason.value} key={reason.value}>{reason.label}</option>)}
              </Select>
              {errors.reason?.message&&<small className="wizard-field-error">{errors.reason.message}</small>}
            </label>
            <label>Cantidad{selected?" ("+selected.unit+")":""}
              <Input type="number" min="0.001" step="0.001" inputMode="decimal" {...register("quantity")} placeholder="0" aria-invalid={Boolean(errors.quantity)}/>
              {errors.quantity?.message&&<small className="wizard-field-error">{errors.quantity.message}</small>}
            </label>
            <label className="span-2">Observación opcional
              <Textarea maxLength={240} rows={3} {...register("observation")} placeholder="Agrega una referencia si necesitas explicar el ajuste"/>
              {errors.observation?.message&&<small className="wizard-field-error">{errors.observation.message}</small>}
            </label>
          </div>

          {selected&&Number.isFinite(quantity)&&quantity>0&&<div className={"inventory-adjustment-preview "+movementType}>
            <Icon name={movementType==="entry"?"plus":"minus"} size={15}/>
            <span><b>Stock resultante: {formatRegionalNumber(Math.max(0,projected),location?.country,{maximumFractionDigits:3})} {selected.unit}</b><small>Actual {formatRegionalNumber(currentStock,location?.country,{maximumFractionDigits:3})} {selected.unit} · {movementType==="entry"?"+":"-"}{formatRegionalNumber(quantity,location?.country,{maximumFractionDigits:3})} {selected.unit}</small></span>
          </div>}

          {isSubmitted&&hasErrors&&<div className="inventory-adjustment-validation" role="alert"><Icon name="alert" size={15}/><span>Revisa los campos marcados antes de guardar.</span></div>}
        </div>
        <footer><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy||items.length===0}>{busy?"Guardando…":"Guardar"}</Button></footer>
      </form>
      {busy&&<div className="modal-busy" role="status"><i/><span>Guardando…</span></div>}
    </section>
  </div>;
}
