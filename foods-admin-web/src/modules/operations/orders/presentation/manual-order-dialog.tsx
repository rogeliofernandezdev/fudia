"use client";
import "./manual-order.css";
import {useEffect,useRef,useState,type FormEvent} from "react";
import {useForm,useWatch} from "react-hook-form";
import {Button,ConfirmDialog,Dialog,FormField,Input,ModalHeader,ModalFooter,MoneyInput,Tabs} from "@/design-system";
import {LocationMap,type Coordinates} from "@/design-system/location-map";
import {channelIcons} from "./order-meta";
import {useFeedback} from "@/providers/feedback-provider";
import type {Draft} from "../../salon/domain/types";
import {ComandaView} from "./comanda-view";
import {emptyManualOrder,isManualOrderChannel,manualOrderResolver,type ManualOrderDraft,type ManualOrderFields} from "../domain/manual-order-schema";
import type {DeliveryQuote,Option} from "../domain/types";

type QuoteState={status:"idle"|"loading"|"unavailable"|"error"}|{status:"ready";quote:DeliveryQuote};

export function ManualOrderDialog({channels,currencySymbol,busy,onClose,onSave,quoteDelivery}:{channels:Option[];currencySymbol:string;busy:boolean;onClose:()=>void;onSave:(draft:ManualOrderDraft)=>Promise<void>;/** Sugiere el envío con las tarifas del local; rechaza con code delivery_unavailable si no hay tarifas. */quoteDelivery:(point:{latitude:number;longitude:number})=>Promise<DeliveryQuote>}){
 const{register,control,handleSubmit,setValue,formState:{errors,isDirty,isSubmitting}}=useForm<ManualOrderFields>({defaultValues:emptyManualOrder,resolver:manualOrderResolver});
 const channel=useWatch({control,name:"channel"});
 const address=useWatch({control,name:"address"});
 const latitude=useWatch({control,name:"deliveryLatitude"});
 const longitude=useWatch({control,name:"deliveryLongitude"});
 const[phase,setPhase]=useState<"details"|"order">("details");
 const[draft,setDraft]=useState<Draft>({...emptyManualOrder,tableId:"",lines:[]});
 const[discard,setDiscard]=useState(false);
 const[quote,setQuote]=useState<QuoteState>({status:"idle"});
 const submission=useRef(false);
 const quoteSequence=useRef(0);
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
  try{await onSave({...current,deliveryLatitude:latitude,deliveryLongitude:longitude,lines:current.lines})}finally{submission.current=false}
 };
 // Marcar el punto sugiere el envío con las tarifas del local; el cajero puede corregirlo.
 const placePoint=async(point:Coordinates)=>{
  setValue("deliveryLatitude",point.lat.toFixed(6),{shouldDirty:true});
  setValue("deliveryLongitude",point.lng.toFixed(6),{shouldDirty:true});
  const sequence=++quoteSequence.current;
  setQuote({status:"loading"});
  try{
   const result=await quoteDelivery({latitude:point.lat,longitude:point.lng});
   if(sequence!==quoteSequence.current)return;
   setQuote({status:"ready",quote:result});
   if(result.covered)setValue("deliveryFee",Number(result.fee).toFixed(2),{shouldDirty:true,shouldValidate:true});
  }catch(error){
   if(sequence!==quoteSequence.current)return;
   setQuote({status:(error as {code?:string})?.code==="delivery_unavailable"?"unavailable":"error"});
  }
 };
 const feeHelp=quote.status==="loading"?"Calculando envío…"
  :quote.status==="error"?"No se pudo calcular; ingrésalo a mano."
  :quote.status==="ready"?quote.quote.covered?`Sugerido para ${quote.quote.distanceKm} km`:`Fuera de cobertura (hasta ${quote.quote.maxDistanceKm} km)`
  :undefined;
 // Sin títulos de sección: la proximidad agrupa. En delivery (escritorio) el mapa ocupa su propio panel.
 return <>
  {phase==="details"?<div className="modal-backdrop modal-overlay-in">
   <Dialog onResponseClose={onClose} className={`crud-modal manual-order-modal modal-panel-in${delivery?" delivery":""}`} role="dialog" aria-modal="true" aria-labelledby="manual-order-title" aria-busy={locked}>
    <ModalHeader title="Datos del pedido" titleId="manual-order-title" icon="truck" onClose={()=>close()} busy={locked}/>
    <form className="modal-form manual-order-form" onSubmit={continueOrder}>
     <div className="manual-order-body">
      <fieldset className="manual-order-fields" disabled={locked}>
       <div className="manual-order-type">
        <Tabs className="manual-order-channel" label="Tipo de atención" value={channel} onChange={value=>setValue("channel",value,{shouldDirty:true,shouldValidate:Boolean(errors.channel)})} items={channelOptions.map(option=>({value:option.value,label:option.label,icon:channelIcons[option.value]??"receipt"}))}/>
        {errors.channel?.message&&<small className="field-error" role="alert">{errors.channel.message}</small>}
       </div>
       <FormField className="manual-order-name" label="Nombre del cliente" optional={!delivery} error={errors.customerName?.message}><Input icon="user" {...register("customerName")} autoComplete="name" maxLength={160}/></FormField>
       <FormField className="manual-order-phone" label="Teléfono" optional={!delivery} error={errors.customerPhone?.message}><Input icon="phone" {...register("customerPhone")} type="tel" autoComplete="tel" maxLength={30}/></FormField>
       {delivery&&<>
        <div className="manual-order-map">
         <LocationMap hint={false} addressError={errors.address?.message} addressLabel="Dirección de entrega" mapLabel="Mapa para marcar el punto de entrega" address={address} latitude={latitude?Number(latitude):null} longitude={longitude?Number(longitude):null} onAddressChange={value=>setValue("address",value,{shouldDirty:true,shouldValidate:Boolean(errors.address)})} onChange={point=>void placePoint(point)}/>
        </div>
        <FormField className="manual-order-reference" label="Referencia" optional error={errors.reference?.message}><Input icon="note" {...register("reference")} maxLength={240}/></FormField>
        <FormField className="manual-order-fee" label="Costo de envío" help={feeHelp} error={errors.deliveryFee?.message}><MoneyInput symbol={currencySymbol} {...register("deliveryFee")} aria-label={`Costo de envío en ${currencySymbol}`}/></FormField>
       </>}
      </fieldset>
     </div>
     <ModalFooter className="manual-order-actions"><Button action="cancel" disabled={locked} onClick={()=>close()}>Cancelar</Button><Button type="submit" action="continue" disabled={locked||!channelOptions.length}>Continuar</Button></ModalFooter>
    </form>
   </Dialog>
  </div>:<ComandaView initial={draft} mode="create" allTables={[]} busy={locked} channelLabel={channelOptions.find(option=>option.value===draft.channel)?.label??""} notify={notify} close={close} onResponseClose={onClose} save={save} onBack={current=>{if(locked||submission.current)return;setDraft(current);setPhase("details")}}/>}
  <ConfirmDialog open={discard} title="Descartar pedido" description="Los cambios del pedido no se han guardado." confirmLabel="Descartar" onCancel={()=>setDiscard(false)} onConfirm={()=>{if(!locked&&!submission.current)onClose()}}/>
 </>;
}
