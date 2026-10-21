"use client";
import {useRef,useState} from "react";
import {useMutation,useQueryClient} from "@tanstack/react-query";
import {Button,Icon} from "@/design-system";
import {Dialog} from "@/design-system/dialog";
import {useSession} from "@/providers/session-context";
import type {PurchaseInventoryOption,PurchasePresentation,PurchasePresentationDraft} from "../domain/types";
import {savePurchasePresentation} from "../infrastructure/purchases-api";
import {PurchasePresentationsField} from "./purchase-presentations-field";

export function PurchasePresentationDialog({item,close,save}:{item:PurchaseInventoryOption;close:()=>void;save:(presentation:PurchasePresentation)=>void}){
 const{organization}=useSession(),qc=useQueryClient();
 const[rows,setRows]=useState<PurchasePresentationDraft[]>([]),[catalogBusy,setCatalogBusy]=useState(false),[validation,setValidation]=useState("");
 const saving=useRef(false);
 const mutation=useMutation({mutationFn:(row:PurchasePresentationDraft)=>savePurchasePresentation(item.id,row),retry:false,onSuccess:presentation=>{
  void qc.invalidateQueries({queryKey:["purchase-inventory"]});
  void qc.invalidateQueries({queryKey:["purchase-item-picker",organization?.id]});
  save(presentation);
 },onSettled:()=>{saving.current=false}});
 function submit(){
  if(saving.current||catalogBusy)return;
  const row=rows[0],factor=Number(row?.unitsPerPresentation);
  if(!row?.presentationType||!Number.isFinite(factor)||factor<=0||factor>=1e11||Math.abs(factor*1000-Math.round(factor*1000))>0.000001){setValidation("Ingresa una conversión mayor que cero, con hasta tres decimales.");return}
  setValidation("");saving.current=true;mutation.mutate(row);
 }
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal supplier-modal modal-panel-in" aria-labelledby="purchase-presentation-title" aria-busy={mutation.isPending}>
  <div className="modal-accent"/>
  <header><span className="modal-title-icon"><Icon name="box" size={18}/></span><div><small>PRESENTACIÓN DE COMPRA</small><h2 id="purchase-presentation-title">{item.name}</h2></div><button type="button" aria-label="Cerrar" disabled={mutation.isPending||catalogBusy} onClick={close}><Icon name="close"/></button></header>
  <form onSubmit={event=>{event.preventDefault();event.stopPropagation()}} noValidate><div className="purchase-item-section">
   <PurchasePresentationsField unit={item.unit} value={rows} onChange={setRows} disabled={mutation.isPending} onBusyChange={setCatalogBusy}/>
   <p className="muted">Las compras anteriores y el stock no se modifican.</p>
   {(validation||mutation.isError)&&<p className="field-error" role="alert">{validation||mutation.error?.message}</p>}
  </div><footer><Button type="button" kind="ghost" disabled={mutation.isPending||catalogBusy} onClick={close}>Cancelar</Button><Button type="button" disabled={mutation.isPending||catalogBusy} onClick={submit}>{mutation.isPending?"Guardando…":"Guardar"}</Button></footer></form>
 </Dialog></div>;
}
