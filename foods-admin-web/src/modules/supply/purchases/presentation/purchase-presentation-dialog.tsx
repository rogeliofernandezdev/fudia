"use client";
import {useRef,useState} from "react";
import {useMutation,useQueryClient} from "@tanstack/react-query";
import {Button,ModalHeader,ModalFooter} from "@/design-system";
import {Dialog} from "@/design-system/dialog";
import {useSession} from "@/providers/session-context";
import type {PurchaseInventoryOption,PurchasePresentation,PurchasePresentationDraft} from "../domain/types";
import {savePurchasePresentation} from "../infrastructure/purchases-api";
import {PurchasePresentationsField} from "./purchase-presentations-field";
import {presentationContentError,type PresentationUnitRule} from "../domain/presentation-content";

export function PurchasePresentationDialog({item,close,save}:{item:PurchaseInventoryOption;close:()=>void;save:(presentation:PurchasePresentation)=>void}){
 const{organization}=useSession(),qc=useQueryClient();
 const[rows,setRows]=useState<PurchasePresentationDraft[]>([]),[catalogBusy,setCatalogBusy]=useState(false),[validation,setValidation]=useState(""),[unitRule,setUnitRule]=useState<PresentationUnitRule>();
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
  const contentError=presentationContentError(unitRule,row.presentationType,row.unitsPerPresentation);
  if(contentError){setValidation(contentError);return}
  setValidation("");saving.current=true;mutation.mutate(row);
 }
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal supplier-modal modal-panel-in" aria-labelledby="purchase-presentation-title" aria-busy={mutation.isPending}>

  <ModalHeader title={item.name} titleId="purchase-presentation-title" icon="box" subtitle="Presentación de compra" onClose={close} busy={mutation.isPending||catalogBusy} closeLabel="Cerrar"/>
  <form onSubmit={event=>{event.preventDefault();event.stopPropagation()}} noValidate><div className="purchase-item-section">
   <PurchasePresentationsField unit={item.unit} value={rows} onChange={setRows} disabled={mutation.isPending} onBusyChange={setCatalogBusy} onUnitRuleChange={setUnitRule}/>
   <p className="field-help">Las compras anteriores y el stock no se modifican.</p>
   {(validation||mutation.isError)&&<p className="field-error" role="alert">{validation||mutation.error?.message}</p>}
  </div><ModalFooter><Button type="button" action="cancel" disabled={mutation.isPending||catalogBusy} onClick={close}>Cancelar</Button><Button type="button" disabled={mutation.isPending||catalogBusy} onClick={submit} loading={mutation.isPending}>{mutation.isPending?"Guardando…":"Guardar"}</Button></ModalFooter></form>
 </Dialog></div>;
}
