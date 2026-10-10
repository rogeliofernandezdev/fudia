"use client";
import {Dialog} from "@/design-system/dialog";
import {useState,type FormEvent} from "react";
import {Button,FormField,Icon,Input,Select,Textarea} from "@/design-system";
import {useSession} from "@/providers/session-context";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
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
  }
  setErrors(next);
  if(next.reason||next.items||Object.keys(next.lines).length)return;
  save({idempotencyKey,purchaseReceiptId:receipt.id,kind,reason,notes,items});
 }

 return <div className="modal-backdrop modal-overlay-in" role="presentation"><Dialog onResponseClose={close} className="crud-modal purchase-return-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="purchase-return-title" aria-busy={busy}>
  <div className="modal-accent"/>
  <header><span className="modal-title-icon"><Icon name="undo" size={18}/></span><div><small>{receipt.code} · {receipt.number}</small><h2 id="purchase-return-title">Corregir o devolver recepción</h2></div><button type="button" aria-label="Cerrar" onClick={close} disabled={busy}><Icon name="close"/></button></header>
  <form onSubmit={submit} noValidate>
   <div className="purchase-return-body" inert={busy}>
    <div className="form-grid">
     <FormField className="span-2" label="Tipo"><Select value={kind} onChange={e=>setKind(e.target.value as PurchaseReturnKind)}><option value="receipt_correction">Corrección de recepción · reabre pendiente en la OC</option><option value="supplier_return">Devolución a proveedor · mantiene la OC recibida</option></Select></FormField>
     <FormField className="span-2" label="Motivo" error={errors.reason}><Input autoFocus maxLength={160} value={reason} onChange={e=>setReason(e.target.value)} placeholder="Ej. cantidad registrada por error / mercadería defectuosa"/></FormField>
    </div>
    <div className="table-wrap hover-scroll purchase-return-table" tabIndex={0}><table><thead><tr><th>ARTÍCULO</th><th>RECIBIDO</th><th>YA DEVUELTO</th><th>DISPONIBLE</th><th>CANTIDAD</th></tr></thead><tbody>{receipt.items.map((i,index)=><tr className={index%2?"alternate":""} key={i.id}>
     <td><b>{i.itemName}</b><small>{i.presentationName||i.presentationType}</small></td>
     <td>{quantity(i.quantity)}</td>
     <td>{quantity(i.returnedQuantity)}</td>
     <td><b>{quantity(i.returnableQuantity)}</b></td>
     <td><Input type="number" min="0" max={i.returnableQuantity} step={i.presentationType==="unit"?"0.001":"1"} inputMode="decimal" aria-label={`Cantidad a corregir de ${i.itemName}`} aria-invalid={Boolean(errors.lines[i.id])} value={quantities[i.id]??""} onChange={e=>setQuantities({...quantities,[i.id]:e.target.value})}/>{errors.lines[i.id]&&<small className="field-error" role="alert">{errors.lines[i.id]}</small>}</td>
    </tr>)}</tbody></table></div>
    {errors.items&&<div className="purchase-validation" role="alert"><Icon name="alert" size={15}/>{errors.items}</div>}
    <FormField label="Notas" optional><Textarea rows={2} maxLength={500} value={notes} onChange={e=>setNotes(e.target.value)}/></FormField>
   </div>
   <footer><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy}>{busy?"Procesando…":kind==="receipt_correction"?"Registrar corrección":"Registrar devolución"}</Button></footer>
  </form>
  {busy&&<div className="modal-busy" role="status"><i/><span>Procesando…</span></div>}
 </Dialog></div>;
}
