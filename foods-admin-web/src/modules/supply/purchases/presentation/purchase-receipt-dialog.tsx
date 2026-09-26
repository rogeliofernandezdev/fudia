"use client";
import {useForm} from "react-hook-form";
import {Button,Icon,Input,Textarea} from "@/design-system";
import {useSession} from "@/providers/session-context";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import {purchaseReceiptResolver} from "../domain/purchase-schema";
import type {PurchaseOrder,PurchaseReceiptDraft} from "../domain/types";

export function PurchaseReceiptDialog({order,busy,close,save}:{order:PurchaseOrder;busy:boolean;close:()=>void;save:(draft:PurchaseReceiptDraft)=>void}){
  const{location}=useSession();
  const defaults:PurchaseReceiptDraft={
    purchaseOrderId:order.id,
    idempotencyKey:crypto.randomUUID(),
    notes:"",
    items:order.items.map(item=>({purchaseOrderItemId:item.id,quantity:item.pendingQuantity})),
  };
  const{register,handleSubmit,setError,formState:{errors}}=useForm<PurchaseReceiptDraft>({
    defaultValues:defaults,
    resolver:purchaseReceiptResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });

  function submit(draft:PurchaseReceiptDraft){
    for(let index=0;index<draft.items.length;index++){
      const line=draft.items[index];
      const orderLine=order.items.find(item=>item.id===line.purchaseOrderItemId);
      if(!orderLine)continue;
      const quantity=Number(line.quantity);
      const pending=Number(orderLine.pendingQuantity);
      if(quantity>pending+0.000001){
        setError(`items.${index}.quantity`,{type:"manual",message:"No puede superar la cantidad pendiente."});
        return;
      }
      if(quantity>0&&orderLine.presentationType!=="unit"&&!Number.isInteger(quantity)){
        setError(`items.${index}.quantity`,{type:"manual",message:"Paquetes y cajas se reciben en cantidades enteras."});
        return;
      }
    }
    save(draft);
  }

  return <div className="modal-backdrop modal-overlay-in" role="presentation">
    <section className="crud-modal purchase-receipt-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="purchase-receipt-title" aria-busy={busy}>
      <div className="modal-accent"/>
      <header>
        <span className="modal-title-icon"><Icon name="stock" size={18}/></span>
        <div><small>PASO 2 · RECEPCIÓN</small><h2 id="purchase-receipt-title">Recibir {order.number}</h2></div>
        <button type="button" aria-label="Cerrar" onClick={close} disabled={busy}><Icon name="close"/></button>
      </header>
      <form onSubmit={handleSubmit(submit)} noValidate>
        <div className="purchase-receipt-body">
          <div className="purchase-receipt-intro"><Icon name="truck" size={17}/><p><b>Registra únicamente lo que llegó.</b> Puedes recibir una parte ahora y completar lo pendiente en otra recepción.</p></div>
          <div className="purchase-receipt-lines">
            {order.items.map((item,index)=><article className="purchase-receipt-line" key={item.id}>
              <div className="purchase-receipt-item"><b>{item.itemName}</b><small>{item.presentationType==="unit"?"Unidad base":item.presentationType==="box"?"Caja":"Paquete"} · {item.unit}</small></div>
              <div><small>SOLICITADO</small><b>{formatRegionalNumber(Number(item.quantity),location?.country,{maximumFractionDigits:3})}</b></div>
              <div><small>RECIBIDO</small><b>{formatRegionalNumber(Number(item.receivedQuantity),location?.country,{maximumFractionDigits:3})}</b></div>
              <div><small>PENDIENTE</small><b>{formatRegionalNumber(Number(item.pendingQuantity),location?.country,{maximumFractionDigits:3})}</b></div>
              <label>Recibir ahora<Input type="number" min="0" max={item.pendingQuantity} step={item.presentationType==="unit"?"0.001":"1"} inputMode="decimal" {...register(`items.${index}.quantity`)} aria-invalid={Boolean(errors.items?.[index]?.quantity)}/>{errors.items?.[index]?.quantity?.message&&<small className="wizard-field-error">{errors.items[index]?.quantity?.message}</small>}</label>
              <input type="hidden" {...register(`items.${index}.purchaseOrderItemId`)}/>
            </article>)}
          </div>
          {typeof errors.items?.message==="string"&&<div className="purchase-validation" role="alert"><Icon name="alert" size={15}/>{errors.items.message}</div>}
          <label className="purchase-receipt-notes">Observación opcional<Textarea rows={3} maxLength={500} {...register("notes")} placeholder="Documento, lote, incidencia o referencia de esta recepción"/>{errors.notes?.message&&<small className="wizard-field-error">{errors.notes.message}</small>}</label>
        </div>
        <footer><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" kind="success" disabled={busy}>{busy?"Confirmando…":"Confirmar recepción"}</Button></footer>
      </form>
      {busy&&<div className="modal-busy" role="status"><i/><span>Confirmando recepción…</span></div>}
    </section>
  </div>;
}
