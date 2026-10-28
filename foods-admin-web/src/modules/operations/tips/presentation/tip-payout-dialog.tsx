"use client";
import {useState} from "react";
import {useQuery,useMutation,useQueryClient} from "@tanstack/react-query";
import LookupSelect from "react-select";
import {Dialog,ModalHeader,ModalForm,ModalBody,ModalFooter,Button,FormField,Input,Select,Textarea,ErrorState,EmptyState,Skeleton} from "@/design-system";
import {createLookupSelectStyles} from "@/design-system/search-select-styles";
import {useSession,useFeedback,useMoney} from "@/providers";
import {useSettings} from "@/providers/settings-context";
import {getTipWorkspace,createTipPayout} from "../infrastructure/tips-api";
import "./tips.css";

export function TipPayoutDialog({shift,close}:{shift:{id:string;cashRegisterName:string};close:()=>void}){
 const{organization,location,user}=useSession(),{notify}=useFeedback(),money=useMoney(),client=useQueryClient(),settings=useSettings();
 const query=useQuery({queryKey:["cash-tips",organization?.id,location?.id,user?.id,shift.id],queryFn:()=>getTipWorkspace(shift.id)});
 const[recipientId,setRecipient]=useState(""),[amount,setAmount]=useState(""),[method,setMethod]=useState<"cash"|"transfer">("cash"),[note,setNote]=useState(""),[requestKey]=useState(()=>crypto.randomUUID()),[validation,setValidation]=useState("");
 const mutation=useMutation({retry:false,mutationFn:()=>createTipPayout(shift.id,{recipientId,amount:Number(amount),method,note,requestKey}),onSuccess:()=>{for(const key of ["cash-tips","cash-shift","cash-shifts","cash-registers","cash-report"])void client.invalidateQueries({queryKey:[key]});close();notify({tone:"success",title:"Propinas entregadas",message:"La entrega y el saldo pendiente quedaron registrados."})},onError:error=>notify({tone:"danger",title:"No se pudo registrar la entrega",message:error.message})});
 const recipients=(query.data?.recipients??[]).map(person=>({value:person.id,label:person.name}));
 function change(){setValidation("")}
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={()=>{if(!mutation.isPending)close()}} className="crud-modal compact tip-payout-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="tip-payout-title">
 <ModalHeader title="Entrega de propinas" titleId="tip-payout-title" subtitle={shift.cashRegisterName} icon="cash" busy={mutation.isPending} onClose={close}/>
 <ModalForm onSubmit={event=>{event.preventDefault();if(mutation.isPending)return;if(!recipientId||!Number.isFinite(Number(amount))||Number(amount)<=0||Number(amount)>Number(query.data?.available??0)){setValidation("Selecciona un trabajador e ingresa un monto dentro del saldo disponible.");return}mutation.mutate()}} noValidate inert={mutation.isPending}>
 <ModalBody>{query.isLoading?<Skeleton className="tip-settings-skeleton"/>:query.isError?<ErrorState title="No pudimos cargar las propinas" text={query.error.message} onRetry={()=>void query.refetch()}/>:!query.data?<EmptyState title="Sin información disponible"/>:<div className="tip-settings-form">
 <div className="tip-available"><span>Propinas disponibles para entregar</span><strong>{money(Number(query.data.available))}</strong></div>
 <FormField as="div" label="Trabajador que recibe"><LookupSelect<{value:string;label:string}> inputId="tip-recipient" aria-label="Trabajador que recibe" options={recipients} value={recipients.find(option=>option.value===recipientId)??null} onChange={option=>{change();setRecipient(option?.value??"")}} isSearchable placeholder="Buscar trabajador…" noOptionsMessage={()=>"Sin trabajadores disponibles"} styles={createLookupSelectStyles<{value:string;label:string}>()}/></FormField>
 <div className="tip-settings-grid"><FormField label="Monto a entregar"><Input type="number" inputMode="decimal" min={10**-settings.currencyDecimals} step={10**-settings.currencyDecimals} max={query.data.available} value={amount} onChange={event=>{change();setAmount(event.target.value)}}/></FormField><FormField label="Medio de entrega"><Select value={method} onChange={event=>{change();setMethod(event.target.value as "cash"|"transfer")}}><option value="cash">Efectivo de este turno</option><option value="transfer">Transferencia</option></Select></FormField></div>
 <FormField label="Observación" optional><Textarea rows={2} maxLength={240} value={note} onChange={event=>{change();setNote(event.target.value)}}/></FormField>
 <p className="tip-help">{method==="cash"?"La entrega reducirá el efectivo esperado del turno y las propinas pendientes.":"La transferencia reducirá las propinas pendientes. No modifica el efectivo del cajón."}</p>
 {validation&&<p className="field-error" role="alert">{validation}</p>}
 </div>}</ModalBody>
 <ModalFooter><Button action="cancel" disabled={mutation.isPending} onClick={close}>Cancelar</Button><Button type="submit" disabled={mutation.isPending||!query.data||Number(query.data.available)<=0} loading={mutation.isPending}>{mutation.isPending?"Registrando…":"Registrar entrega"}</Button></ModalFooter>
 </ModalForm></Dialog></div>;
}
