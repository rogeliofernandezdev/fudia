"use client";
import {useState} from "react";
import {Button,Icon,Input,Select,Textarea} from "@/design-system";
import type {PurchaseReceiptDetail,PurchaseReturnDraft,PurchaseReturnKind} from "../domain/types";

export function PurchaseReturnDialog({receipt,busy,close,save}:{receipt:PurchaseReceiptDetail;busy:boolean;close:()=>void;save:(draft:PurchaseReturnDraft)=>void}){
 const[idempotencyKey]=useState(()=>crypto.randomUUID());
 const[kind,setKind]=useState<PurchaseReturnKind>("receipt_correction");
 const[reason,setReason]=useState("");const[notes,setNotes]=useState("");
 const[quantities,setQuantities]=useState<Record<string,string>>(Object.fromEntries(receipt.items.map(i=>[i.id,""])));
 function submit(){
  const items=receipt.items.map(i=>({purchaseReceiptItemId:i.id,quantity:quantities[i.id]??""})).filter(i=>Number(i.quantity)>0);
  if(!reason.trim()||!items.length)return;
  if(items.some(x=>{const item=receipt.items.find(i=>i.id===x.purchaseReceiptItemId);return item&&Number(x.quantity)>Number(item.returnableQuantity)+0.000001}))return;
  save({idempotencyKey,purchaseReceiptId:receipt.id,kind,reason,notes,items});
 }
 return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal modal-panel-in" role="dialog" aria-modal="true">
  <header><span className="modal-title-icon"><Icon name="truck" size={18}/></span><div><small>{receipt.code} · {receipt.number}</small><h2>Corregir o devolver recepción</h2></div><button onClick={close} aria-label="Cerrar"><Icon name="close"/></button></header>
  <div className="form-grid">
   <label className="span-2">Tipo<Select value={kind} onChange={e=>setKind(e.target.value as PurchaseReturnKind)}><option value="receipt_correction">Corrección de recepción · reabre pendiente en la OC</option><option value="supplier_return">Devolución a proveedor · mantiene la OC recibida</option></Select></label>
   <label className="span-2">Motivo<Input maxLength={160} value={reason} onChange={e=>setReason(e.target.value)} placeholder="Ej. cantidad registrada por error / mercadería defectuosa"/></label>
  </div>
  <div className="table-wrap"><table><thead><tr><th>ARTÍCULO</th><th>RECIBIDO</th><th>YA DEVUELTO</th><th>DISPONIBLE</th><th>CANTIDAD</th></tr></thead><tbody>{receipt.items.map(i=><tr key={i.id}><td><b>{i.itemName}</b><small>{i.presentationType}</small></td><td>{i.quantity}</td><td>{i.returnedQuantity}</td><td><b>{i.returnableQuantity}</b></td><td><Input type="number" min="0" max={i.returnableQuantity} step={i.presentationType==="unit"?"0.001":"1"} value={quantities[i.id]??""} onChange={e=>setQuantities({...quantities,[i.id]:e.target.value})}/></td></tr>)}</tbody></table></div>
  <label>Notas<Textarea rows={2} maxLength={500} value={notes} onChange={e=>setNotes(e.target.value)}/></label>
  <footer><Button kind="ghost" onClick={close}>Cancelar</Button><Button disabled={busy||!reason.trim()} onClick={submit}>{busy?"Procesando…":kind==="receipt_correction"?"Registrar corrección":"Registrar devolución"}</Button></footer>
 </section></div>
}
