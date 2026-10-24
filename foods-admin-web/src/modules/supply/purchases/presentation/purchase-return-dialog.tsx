"use client";
import {Dialog} from "@/design-system/dialog";
import {DataTable} from "@/design-system/data-table";
import {useState,type FormEvent} from "react";
import {Button,FormField,Icon,Input,Select,Textarea,ModalHeader,ModalFooter} from "@/design-system";
import {useSession} from "@/providers/session-context";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import {requiresWholeQuantity} from "../domain/purchase-totals";
import type {PurchaseReceiptDetail,PurchaseReturnDraft,PurchaseReturnKind} from "../domain/types";

type ReturnErrors={reason?:string;items?:string;lines:Record<string,string>};
const noErrors:ReturnErrors={lines:{}};

export function PurchaseReturnDialog({receipt,busy,close,save}:{receipt:PurchaseReceiptDetail;busy:boolean;close:()=>void;save:(draft:PurchaseReturnDraft)=>void}){
 const{location}=useSession();
 const[idempotencyKey]=useState(()=>crypto.randomUUID());
 const[kind,setKind]=useState<PurchaseReturnKind>("receipt_correction");
 const[reason,setReason]=useState("");const[notes,setNotes]=useState("");
 const[quantities,setQuantities]=useState<Record<string,string>>(Object.fromEntries(receipt.items.map(i=>[i.id,""])));
 const[errors,setErrors]=useState<ReturnErrors>(noErrors);
 const quantity=(value:string|number)=>formatRegionalNumber(Number(value),location?.country,{maximumFractionDigits:3});

 function submit(event:FormEvent){
  event.preventDefault();
  if(busy)return;
  const items=receipt.items.map(i=>({purchaseReceiptItemId:i.id,quantity:quantities[i.id]??""})).filter(i=>Number(i.quantity)>0);
  const next:ReturnErrors={lines:{}};
  if(!reason.trim())next.reason="Indica el motivo de la corrección o devolución.";
  if(!items.length)next.items="Ingresa una cantidad mayor que cero en al menos un artículo.";
  for(const line of items){
   const item=receipt.items.find(i=>i.id===line.purchaseReceiptItemId);
   if(item&&Number(line.quantity)>Number(item.returnableQuantity)+0.000001)next.lines[line.purchaseReceiptItemId]=`Máximo ${quantity(item.returnableQuantity)}.`;
   else if(item&&requiresWholeQuantity(item.presentationType)&&!Number.isInteger(Number(line.quantity)))next.lines[line.purchaseReceiptItemId]="Esta presentación se devuelve en cantidades enteras.";
  }
  setErrors(next);
  if(next.reason||next.items||Object.keys(next.lines).length)return;
  save({idempotencyKey,purchaseReceiptId:receipt.id,kind,reason,notes,items});
 }

 return <div className="modal-backdrop modal-overlay-in" role="presentation"><Dialog onResponseClose={close} className="crud-modal purchase-return-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="purchase-return-title" aria-busy={busy}>

  <ModalHeader title="Corregir o devolver recepción" titleId="purchase-return-title" icon="undo" subtitle={receipt.supplierName} onClose={close} busy={busy} closeLabel="Cerrar"/>
  <form onSubmit={submit} noValidate>
   <div className="purchase-return-body" inert={busy}>
    <div className="form-grid">
     <FormField className="span-2" label="Tipo"><Select value={kind} onChange={e=>setKind(e.target.value as PurchaseReturnKind)}><option value="receipt_correction">Corrección de recepción · reabre pendiente en la OC</option><option value="supplier_return">Devolución a proveedor · mantiene la OC recibida</option></Select></FormField>
     <FormField className="span-2" label="Motivo" error={errors.reason}><Input autoFocus maxLength={160} value={reason} onChange={e=>setReason(e.target.value)} placeholder="Ej. cantidad registrada por error / mercadería defectuosa"/></FormField>
    </div>
    <DataTable label="Artículos a corregir o devolver" containerClassName="hover-scroll purchase-return-table"><thead><tr><th>Artículo</th><th>Recibido</th><th>Ya devuelto</th><th>Disponible</th><th>Cantidad</th></tr></thead><tbody>{receipt.items.map((i,index)=><tr className={index%2?"alternate":""} key={i.id}>
     <td><b>{i.itemName}</b><small>{i.presentationName||i.presentationType}</small></td>
     <td>{quantity(i.quantity)}</td>
     <td>{quantity(i.returnedQuantity)}</td>
     <td><b>{quantity(i.returnableQuantity)}</b></td>
     <td><Input type="number" min="0" max={i.returnableQuantity} step={requiresWholeQuantity(i.presentationType)?"1":"0.001"} inputMode="decimal" aria-label={`Cantidad a corregir de ${i.itemName}`} aria-invalid={Boolean(errors.lines[i.id])} value={quantities[i.id]??""} onChange={e=>setQuantities({...quantities,[i.id]:e.target.value})}/>{errors.lines[i.id]&&<small className="field-error" role="alert">{errors.lines[i.id]}</small>}</td>
    </tr>)}</tbody></DataTable>
    {errors.items&&<div className="purchase-validation" role="alert"><Icon name="alert" size={16}/>{errors.items}</div>}
    <FormField label="Notas" optional><Textarea rows={2} maxLength={500} value={notes} onChange={e=>setNotes(e.target.value)}/></FormField>
   </div>
   <ModalFooter><Button type="button" action="cancel" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy} loading={busy}>{busy?"Procesando…":kind==="receipt_correction"?"Registrar corrección":"Registrar devolución"}</Button></ModalFooter>
  </form>
 </Dialog></div>;
}
