"use client";
import {Dialog} from "@/design-system/dialog";
import {useFieldArray,useForm,useWatch} from "react-hook-form";
import {Button,FormField,Icon,Input,RowActionButton,Select,Textarea} from "@/design-system";
import {useSession} from "@/providers/session-context";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import {inventoryTransferResolver} from "../domain/inventory-transfer-schema";
import type {InventoryProductOption,InventoryTransferDraft,LocationOption} from "../domain/types";

export function InventoryTransferDialog({items,locations,currentLocationId,busy,close,save}:{items:InventoryProductOption[];locations:LocationOption[];currentLocationId:string;busy:boolean;close:()=>void;save:(draft:InventoryTransferDraft)=>void}){
  const{location}=useSession();
  const destinations=locations.filter(option=>option.id!==currentLocationId);
  const{control,register,handleSubmit,setError,formState:{errors,isSubmitted}}=useForm<InventoryTransferDraft>({
    defaultValues:{idempotencyKey:crypto.randomUUID(),toLocationId:destinations.length===1?destinations[0].id:"",notes:"",items:items[0]?[{inventoryItemId:items[0].id,quantity:""}]:[]},
    resolver:inventoryTransferResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });
  const{fields,append,remove}=useFieldArray({control,name:"items"});
  const lines=useWatch({control,name:"items"})??[];
  const selectedIds=new Set(lines.map(line=>line?.inventoryItemId));
  const hasErrors=Object.keys(errors).length>0;
  const amount=(value:string)=>formatRegionalNumber(Number(value||0),location?.country,{maximumFractionDigits:3});

  function add(){
    const next=items.find(item=>!selectedIds.has(item.id));
    if(next)append({inventoryItemId:next.id,quantity:""});
  }

  function submit(draft:InventoryTransferDraft){
    let blocked=false;
    draft.items.forEach((line,index)=>{
      const item=items.find(option=>option.id===line.inventoryItemId);
      if(item&&Number(line.quantity)>Number(item.quantity)+0.000001){
        setError(`items.${index}.quantity`,{type:"manual",message:"Supera el stock disponible."});
        blocked=true;
      }
    });
    if(!blocked)save(draft);
  }

  return <div className="modal-backdrop modal-overlay-in" role="presentation">
    <Dialog className="crud-modal inventory-transfer-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="inventory-transfer-title" aria-busy={busy}>
      <div className="modal-accent"/>
      <header>
        <span className="modal-title-icon"><Icon name="truck" size={18}/></span>
        <div><small>INVENTARIO</small><h2 id="inventory-transfer-title">Transferir a otro local</h2></div>
        <button type="button" aria-label="Cerrar" onClick={close} disabled={busy}><Icon name="close"/></button>
      </header>
      <form onSubmit={handleSubmit(submit)} noValidate>
        <div className="inventory-transfer-body">
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

          <section className="inventory-transfer-lines" aria-labelledby="inventory-transfer-lines-title">
            <div className="inventory-transfer-lines-head">
              <b id="inventory-transfer-lines-title">Artículos a transferir</b>
              <Button type="button" kind="secondary" icon="plus" onClick={add} disabled={busy||fields.length>=items.length}>Agregar artículo</Button>
            </div>
            {fields.length>0&&<div className="table-wrap"><table className="inventory-transfer-table">
              <thead><tr><th>ARTÍCULO</th><th>DISPONIBLE</th><th>CANTIDAD</th><th><span className="sr-only">Acciones</span></th></tr></thead>
              <tbody>{fields.map((field,index)=>{
                const line=lines[index];
                const item=items.find(option=>option.id===line?.inventoryItemId);
                const lineErrors=errors.items?.[index];
                return <tr key={field.id} className={lineErrors?"has-error":""}>
                  <td><Select aria-label="Artículo" aria-invalid={Boolean(lineErrors?.inventoryItemId)} {...register(`items.${index}.inventoryItemId`)}>
                    {items.map(option=><option key={option.id} value={option.id} disabled={selectedIds.has(option.id)&&option.id!==line?.inventoryItemId}>{option.name}{option.kind==="ingredient"?" · Insumo":""}</option>)}
                  </Select>{lineErrors?.inventoryItemId?.message&&<small className="field-error">{lineErrors.inventoryItemId.message}</small>}</td>
                  <td className="inventory-transfer-available">{item?amount(item.quantity)+" "+item.unit:"—"}</td>
                  <td><div className="inventory-transfer-quantity"><Input type="number" min="0.001" step="0.001" inputMode="decimal" placeholder="0" aria-label={"Cantidad"+(item?" en "+item.unit:"")} aria-invalid={Boolean(lineErrors?.quantity)} {...register(`items.${index}.quantity`)}/><span>{item?.unit}</span></div>{item&&<small className="inventory-transfer-hint">Disponible {amount(item.quantity)} {item.unit}</small>}{lineErrors?.quantity?.message&&<small className="field-error">{lineErrors.quantity.message}</small>}</td>
                  <td><div className="table-actions"><RowActionButton action="remove" label="Quitar artículo" onClick={()=>remove(index)} disabled={busy||fields.length===1}/></div></td>
                </tr>;
              })}</tbody>
            </table></div>}
            {!fields.length&&<p className="inventory-transfer-empty">{items.length?"Agrega el primer artículo que quieres mover.":"Este local no tiene artículos con stock para transferir."}</p>}
            {errors.items?.message&&<small className="field-error" role="alert">{errors.items.message}</small>}
          </section>

          <FormField label="Nota" optional error={errors.notes?.message}>
            <Textarea rows={2} maxLength={500} placeholder="Motivo o referencia del traslado" {...register("notes")}/>
          </FormField>

          {isSubmitted&&hasErrors&&<div className="inventory-validation" role="alert"><Icon name="alert" size={15}/><span>Revisa los campos marcados antes de guardar.</span></div>}
        </div>
        <footer><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy||!destinations.length||!items.length}>{busy?"Guardando…":"Guardar"}</Button></footer>
      </form>
      {busy&&<div className="modal-busy" role="status"><i/><span>Guardando…</span></div>}
    </Dialog>
  </div>;
}
