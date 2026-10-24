"use client";
import {Dialog} from "@/design-system/dialog";
import {useState} from "react";
import {Controller,useFieldArray,useForm,useWatch} from "react-hook-form";
import {Button,FormField,Icon,Input,RowActionButton,Select,Textarea,ModalHeader,ModalFooter,FormSection,IconTile} from "@/design-system";
import {useSession} from "@/providers/session-context";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import {inventoryTransferResolver} from "../domain/inventory-transfer-schema";
import type {InventoryProductOption,InventoryTransferDraft,LocationOption} from "../domain/types";
import {InventoryItemLookup} from "./inventory-item-lookup";

export function InventoryTransferDialog({locations,currentLocationId,busy,close,save}:{locations:LocationOption[];currentLocationId:string;busy:boolean;close:()=>void;save:(draft:InventoryTransferDraft)=>void}){
  const{location}=useSession();
  // Items picked in the lookups, kept to show stock and unit per line.
  const[picked,setPicked]=useState<Record<string,InventoryProductOption>>({});
  const destinations=locations.filter(option=>option.id!==currentLocationId);
  const{control,register,handleSubmit,setError,formState:{errors,isSubmitted}}=useForm<InventoryTransferDraft>({
    defaultValues:{idempotencyKey:crypto.randomUUID(),toLocationId:destinations.length===1?destinations[0].id:"",notes:"",items:[{inventoryItemId:"",quantity:""}]},
    resolver:inventoryTransferResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });
  const{fields,append,remove}=useFieldArray({control,name:"items"});
  const lines=useWatch({control,name:"items"})??[];
  const selectedIds=lines.map(line=>line?.inventoryItemId).filter(Boolean);
  const hasErrors=Object.keys(errors).length>0;
  const amount=(value:string)=>formatRegionalNumber(Number(value||0),location?.country,{maximumFractionDigits:3});

  function add(){
    append({inventoryItemId:"",quantity:""});
  }

  function submit(draft:InventoryTransferDraft){
    let blocked=false;
    draft.items.forEach((line,index)=>{
      const item=picked[line.inventoryItemId];
      if(item&&Number(line.quantity)>Number(item.quantity)+0.000001){
        setError(`items.${index}.quantity`,{type:"manual",message:"Supera el stock disponible."});
        blocked=true;
      }
    });
    if(!blocked)save(draft);
  }

  return <div className="modal-backdrop modal-overlay-in" role="presentation">
    <Dialog onResponseClose={close} className="crud-modal inventory-transfer-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="inventory-transfer-title" aria-busy={busy}>

      <ModalHeader title="Transferir a otro local" titleId="inventory-transfer-title" icon="truck" subtitle="Inventario" onClose={close} busy={busy} closeLabel="Cerrar"/>
      <form onSubmit={handleSubmit(submit)} noValidate>
        <div className="inventory-transfer-body">
          <FormSection icon="route" title="Ruta del traslado" bare>
          <div className="inventory-transfer-route">
            <div className="inventory-transfer-origin"><small>Origen</small><strong>{location?.name??"Local activo"}</strong></div>
            <span className="inventory-transfer-arrow" aria-hidden="true"><Icon name="chevron" size={16}/></span>
            <FormField label="Local destino" error={errors.toLocationId?.message}>
              <Select autoFocus {...register("toLocationId")}>
                <option value="">{destinations.length?"Selecciona destino":"No hay otros locales activos"}</option>
                {destinations.map(option=><option key={option.id} value={option.id}>{option.name}</option>)}
              </Select>
            </FormField>
          </div>
          </FormSection>

          <section className="inventory-transfer-lines" aria-labelledby="inventory-transfer-lines-title">
            <div className="inventory-transfer-lines-head">
              <h3 id="inventory-transfer-lines-title" className="form-section-title"><IconTile name="stock" tone="brand" size="sm"/>Artículos a transferir</h3>
              <Button type="button" kind="secondary" icon="plus" onClick={add} disabled={busy||lines.some(line=>!line?.inventoryItemId)}>Agregar artículo</Button>
            </div>
            {fields.length>0&&<div className="table-wrap"><table className="inventory-transfer-table">
              <thead><tr><th>Artículo</th><th>Disponible</th><th>Cantidad</th><th><span className="sr-only">Acciones</span></th></tr></thead>
              <tbody>{fields.map((field,index)=>{
                const line=lines[index];
                const item=line?.inventoryItemId?picked[line.inventoryItemId]:undefined;
                const lineErrors=errors.items?.[index];
                return <tr key={field.id} className={lineErrors?"has-error":""}>
                  <td><Controller control={control} name={`items.${index}.inventoryItemId`} render={({field:lookup})=><InventoryItemLookup ariaLabel="Artículo" value={item??null} invalid={Boolean(lineErrors?.inventoryItemId)} disabled={busy} excludeIds={selectedIds.filter(id=>id!==lookup.value)} onChange={option=>{if(option)setPicked(current=>({...current,[option.id]:option}));lookup.onChange(option?.id??"")}}/>}/>{lineErrors?.inventoryItemId?.message&&<small className="field-error">{lineErrors.inventoryItemId.message}</small>}</td>
                  <td className="inventory-transfer-available">{item?amount(item.quantity)+" "+item.unit:"—"}</td>
                  <td><div className="inventory-transfer-quantity"><Input type="number" min="0.001" step="0.001" inputMode="decimal" placeholder="0" aria-label={"Cantidad"+(item?" en "+item.unit:"")} aria-invalid={Boolean(lineErrors?.quantity)} {...register(`items.${index}.quantity`)}/><span>{item?.unit}</span></div>{item&&<small className="inventory-transfer-hint">Disponible {amount(item.quantity)} {item.unit}</small>}{lineErrors?.quantity?.message&&<small className="field-error">{lineErrors.quantity.message}</small>}</td>
                  <td><div className="table-actions"><RowActionButton action="remove" label="Quitar artículo" onClick={()=>remove(index)} disabled={busy||fields.length===1}/></div></td>
                </tr>;
              })}</tbody>
            </table></div>}
            {!fields.length&&<p className="inventory-transfer-empty">Agrega el primer artículo que quieres mover.</p>}
            {errors.items?.message&&<small className="field-error" role="alert">{errors.items.message}</small>}
          </section>

          <FormSection icon="note" tone="neutral" title="Nota" bare>
          <FormField label="Nota" optional error={errors.notes?.message}>
            <Textarea rows={2} maxLength={500} placeholder="Motivo o referencia del traslado" {...register("notes")}/>
          </FormField>
          </FormSection>

          {isSubmitted&&hasErrors&&<div className="inventory-validation" role="alert"><Icon name="alert" size={16}/><span>Revisa los campos marcados antes de guardar.</span></div>}
        </div>
        <ModalFooter><Button type="button" action="cancel" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy||!destinations.length} loading={busy}>{busy?"Guardando…":"Guardar"}</Button></ModalFooter>
      </form>
    </Dialog>
  </div>;
}
