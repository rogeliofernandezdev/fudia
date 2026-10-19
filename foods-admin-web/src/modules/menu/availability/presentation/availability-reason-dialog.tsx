"use client";
import {useRef} from "react";
import {useForm,useWatch} from "react-hook-form";
import {Button,Dialog,FormField,Input,Textarea,ModalHeader,ModalFooter} from "@/design-system";
import {availabilityReasonResolver} from "../domain/availability-reason-schema";
import type {AvailabilityChange,AvailabilityReasonDraft,AvailabilityReasonOption} from "../domain/types";

/**
 * Motivo de un toque para agotar o ajustar el cupo: opciones del API con la primera preseleccionada
 * («Se acabó»), así en hora punta el cambio queda en dos toques. Solo «Otro» abre un detalle obligatorio.
 */
export function AvailabilityReasonDialog({change,options,busy,close,save}:{change:AvailabilityChange;options:AvailabilityReasonOption[];busy:boolean;close:()=>void;save:(value:AvailabilityReasonDraft)=>Promise<void>}){
 const submitting=useRef(false);
 const{register,handleSubmit,control,formState:{errors,isSubmitting}}=useForm<AvailabilityReasonDraft>({defaultValues:{reasonCode:options[0]?.value??"",reason:""},resolver:availabilityReasonResolver,mode:"onSubmit",reValidateMode:"onChange"});
 const locked=busy||isSubmitting;
 const other=useWatch({control,name:"reasonCode"})==="other";
 async function submit(){
  if(busy||submitting.current)return;
  submitting.current=true;
  try{await handleSubmit(async value=>{await save({reasonCode:value.reasonCode,reason:value.reasonCode==="other"?value.reason.trim():""});})();}finally{submitting.current=false;}
 }
 const title=change.kind==="quota"?"AJUSTAR CUPO":"AGOTAR HOY";
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal compact modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="availability-reason-title" aria-busy={locked}>

  <ModalHeader title={change.item.name} titleId="availability-reason-title" icon="availability" subtitle={title} onClose={close} busy={locked} closeLabel="Cerrar"/>
  <form onSubmit={event=>event.preventDefault()} noValidate>
   <div className="form-grid">
    {change.kind==="quota"&&<><FormField label="Cupo actual"><Input value={change.item.portionQuantity??"Sin registrar"} readOnly disabled/></FormField><FormField label="Nuevo cupo"><Input value={change.portionQuantity??""} readOnly disabled/></FormField></>}
    <FormField className="span-2" as="div" label="Motivo" error={errors.reasonCode?.message}>
     <div className="availability-reasons" role="radiogroup" aria-label="Motivo del cambio">
      {options.map(option=><label key={option.value} className="availability-reason"><input type="radio" value={option.value} disabled={locked} {...register("reasonCode")}/><i aria-hidden="true"/>{option.label}</label>)}
     </div>
    </FormField>
    {other&&<FormField className="span-2" label="Detalle" error={errors.reason?.message}><Textarea autoFocus rows={2} maxLength={240} disabled={locked} {...register("reason")} placeholder="Cuéntanos qué pasó"/></FormField>}
   </div>
   <ModalFooter><Button kind="ghost" onClick={close} disabled={locked}>Cancelar</Button><Button onClick={submit} disabled={locked} loading={locked}>{locked?"Guardando…":"Guardar"}</Button></ModalFooter>
  </form>
 </Dialog></div>;
}
