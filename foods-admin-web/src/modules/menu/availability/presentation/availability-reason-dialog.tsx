"use client";
import {useRef} from "react";
import {useForm} from "react-hook-form";
import {Button,Dialog,FormField,Icon,Input,Textarea} from "@/design-system";
import {availabilityReasonResolver} from "../domain/availability-reason-schema";
import type {AvailabilityChange,AvailabilityReasonDraft} from "../domain/types";

export function AvailabilityReasonDialog({change,busy,close,save}:{change:AvailabilityChange;busy:boolean;close:()=>void;save:(reason:string)=>Promise<void>}){
 const submitting=useRef(false);
 const{register,handleSubmit,formState:{errors,isSubmitting}}=useForm<AvailabilityReasonDraft>({defaultValues:{reason:""},resolver:availabilityReasonResolver,mode:"onSubmit",reValidateMode:"onChange"});
 const locked=busy||isSubmitting;
 async function submit(){
  if(busy||submitting.current)return;
  submitting.current=true;
  try{await handleSubmit(async value=>{await save(value.reason.trim());})();}finally{submitting.current=false;}
 }
 const title=change.kind==="quota"?"AJUSTAR CUPO":change.status==="sold_out"?"AGOTAR HOY":"REACTIVAR";
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal compact modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="availability-reason-title" aria-busy={locked}>
  <div className="modal-accent"/>
  <header><span className="modal-title-icon"><Icon name="availability"/></span><div><small>{title}</small><h2 id="availability-reason-title">{change.item.name}</h2></div><button type="button" aria-label="Cerrar" onClick={close} disabled={locked}><Icon name="close"/></button></header>
  <form onSubmit={event=>event.preventDefault()} noValidate>
   <div className="form-grid">
    {change.kind==="quota"&&<><FormField label="Cupo actual"><Input value={change.item.portionQuantity??"Sin registrar"} readOnly disabled/></FormField><FormField label="Nuevo cupo"><Input value={change.portionQuantity??""} readOnly disabled/></FormField></>}
    <FormField className="span-2" label="Motivo del cambio" error={errors.reason?.message}><Textarea autoFocus rows={3} maxLength={240} disabled={locked} {...register("reason")} placeholder="Explica por qué realizas el cambio"/></FormField>
   </div>
   <footer><Button kind="ghost" onClick={close} disabled={locked}>Cancelar</Button><Button onClick={submit} disabled={locked}>{locked?"Guardando…":"Guardar"}</Button></footer>
  </form>
 </Dialog></div>;
}
