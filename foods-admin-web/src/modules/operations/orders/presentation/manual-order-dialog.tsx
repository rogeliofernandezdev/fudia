"use client";
import "./manual-order.css";
import {useEffect,useRef,useState,type FormEvent} from "react";
import {useForm,useWatch} from "react-hook-form";
import {Button,ConfirmDialog,Dialog,FormField,Icon,IconButton,Input,Select} from "@/design-system";
import {useFeedback} from "@/providers/feedback-provider";
import type {Draft} from "../../salon/domain/types";
import {ComandaView} from "./comanda-view";
import {emptyManualOrder,isManualOrderChannel,manualOrderResolver,type ManualOrderDraft,type ManualOrderFields} from "../domain/manual-order-schema";
import type {Option} from "../domain/types";

export function ManualOrderDialog({channels,currencySymbol,busy,onClose,onSave}:{channels:Option[];currencySymbol:string;busy:boolean;onClose:()=>void;onSave:(draft:ManualOrderDraft)=>Promise<void>}){
 const{register,control,handleSubmit,formState:{errors,isDirty,isSubmitting}}=useForm<ManualOrderFields>({defaultValues:emptyManualOrder,resolver:manualOrderResolver});
 const channel=useWatch({control,name:"channel"});
 const[phase,setPhase]=useState<"details"|"order">("details");
 const[draft,setDraft]=useState<Draft>({...emptyManualOrder,tableId:"",lines:[]});
 const[discard,setDiscard]=useState(false);
 const submission=useRef(false);
 const{notify}=useFeedback();
 const locked=busy||isSubmitting;
 const delivery=channel==="delivery";
 const channelOptions=channels.filter(option=>isManualOrderChannel(option.value));
 useEffect(()=>{
  if(!isDirty&&!draft.lines.length&&!draft.notes)return;
  const prevent=(event:BeforeUnloadEvent)=>{event.preventDefault();event.returnValue=""};
  window.addEventListener("beforeunload",prevent);return()=>window.removeEventListener("beforeunload",prevent);
 },[isDirty,draft.lines.length,draft.notes]);
 const close=(current=draft)=>{
  if(locked||submission.current)return;
  setDraft(current);
  if(isDirty||current.lines.length||current.notes)setDiscard(true);else onClose();
 };
 const continueOrder=(event:FormEvent<HTMLFormElement>)=>handleSubmit(fields=>{
  if(locked||submission.current||!channelOptions.some(option=>option.value===fields.channel))return;
  setDraft(previous=>({...previous,...fields,notes:previous.notes,tableId:"",address:fields.channel==="delivery"?fields.address:"",reference:fields.channel==="delivery"?fields.reference:"",deliveryFee:fields.channel==="delivery"?fields.deliveryFee:"0"}));
  setPhase("order");
 })(event);
 const save=async(current:Draft)=>{
  if(locked||submission.current)return;
  submission.current=true;
  setDraft(current);
  try{await onSave({...current,lines:current.lines})}finally{submission.current=false}
 };
 return <>
  {phase==="details"?<div className="modal-backdrop modal-overlay-in manual-order-backdrop">
   <Dialog className="manual-order-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="manual-order-title" aria-busy={locked}>
    <header className="manual-order-head"><span className="manual-order-icon"><Icon name="truck" size={18}/></span><h2 id="manual-order-title">Datos del pedido</h2><IconButton icon="close" label="Cerrar" disabled={locked} onClick={()=>close()}/></header>
    <form className="manual-order-form" onSubmit={continueOrder}>
     <div className="manual-order-body">
      <fieldset className="manual-order-fields" disabled={locked}>
       <FormField className="manual-order-wide" label="Tipo de atención" error={errors.channel?.message}><Select {...register("channel")} data-dialog-initial-focus>{channelOptions.map(option=><option value={option.value} key={option.value}>{option.label}</option>)}</Select></FormField>
       <FormField label="Cliente" optional={!delivery} error={errors.customerName?.message}><Input {...register("customerName")} autoComplete="name" maxLength={160}/></FormField>
       <FormField label="Teléfono" optional={!delivery} error={errors.customerPhone?.message}><Input {...register("customerPhone")} type="tel" autoComplete="tel" maxLength={30}/></FormField>
       {delivery&&<>
        <FormField className="manual-order-wide" label="Dirección de entrega" error={errors.address?.message}><Input {...register("address")} autoComplete="street-address" maxLength={240}/></FormField>
        <FormField label="Referencia de entrega" optional error={errors.reference?.message}><Input {...register("reference")} maxLength={240}/></FormField>
        <FormField label="Costo de envío" error={errors.deliveryFee?.message}><Input {...register("deliveryFee")} type="number" inputMode="decimal" min="0" step="0.01" aria-label={`Costo de envío en ${currencySymbol}`}/></FormField>
       </>}
      </fieldset>
     </div>
     <footer className="manual-order-actions"><Button kind="secondary" disabled={locked} onClick={()=>close()}>Cancelar</Button><Button type="submit" icon="chevron" disabled={locked||!channelOptions.length}>Continuar</Button></footer>
    </form>
   </Dialog>
  </div>:<ComandaView initial={draft} mode="create" allTables={[]} busy={locked} currencySymbol={currencySymbol} channelLabel={channelOptions.find(option=>option.value===draft.channel)?.label??""} notify={notify} close={close} save={save} onBack={current=>{if(locked||submission.current)return;setDraft(current);setPhase("details")}}/>}
  <ConfirmDialog open={discard} title="Descartar pedido" description="Los cambios del pedido no se han guardado." confirmLabel="Descartar" onCancel={()=>setDiscard(false)} onConfirm={()=>{if(!locked&&!submission.current)onClose()}}/>
 </>;
}
