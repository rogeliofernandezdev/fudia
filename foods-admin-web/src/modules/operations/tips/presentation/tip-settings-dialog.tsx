"use client";
import {useState} from "react";
import {useQuery,useMutation,useQueryClient} from "@tanstack/react-query";
import {Button,Dialog,ErrorState,Skeleton,ModalHeader,ModalForm,ModalBody,ModalFooter} from "@/design-system";
import {useSession,useFeedback} from "@/providers";
import {getTipSettings,saveTipSettings} from "../infrastructure/tips-api";
import type {TipPolicy,TipSettings} from "../domain/types";
import {TipSettingsFields} from "./tip-settings-fields";
import "./tips.css";

export function TipSettingsDialog({location,close}:{location:{id:string;name:string};close:()=>void}){
 const{organization,user,can}=useSession(),{notify}=useFeedback(),client=useQueryClient();
 const query=useQuery({queryKey:["tip-settings",organization?.id,location.id,user?.id,"location"],queryFn:()=>getTipSettings("location",location.id),refetchOnWindowFocus:false});
 const mutation=useMutation({mutationFn:(draft:TipSettings)=>saveTipSettings("location",{...draft,inherit:false},location.id),onSuccess:()=>{
  void client.invalidateQueries({queryKey:["tip-settings"]});void client.invalidateQueries({queryKey:["pos-order"]});
  close();notify({tone:"success",title:"Propinas configuradas",message:`La configuración se guardó para ${location.name}.`});
 },onError:error=>notify({tone:"danger",title:"No se pudo guardar",message:error.message})});
 const onClose=()=>{if(!mutation.isPending)close()};
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={onClose} className="crud-modal compact tip-settings-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="tip-settings-title">
  <ModalHeader title="Propinas" titleId="tip-settings-title" icon="percent" busy={mutation.isPending} onClose={onClose}/>
  {query.isLoading?<ModalBody><Skeleton className="tip-settings-skeleton"/></ModalBody>:query.isError?<ModalBody><ErrorState title="No pudimos cargar las propinas" text={query.error.message} onRetry={()=>void query.refetch()}/></ModalBody>:query.data?<TipSettingsForm key={`${location.id}-${query.dataUpdatedAt}`} locationName={location.name} policy={query.data} canManage={can("organizations.manage")} busy={mutation.isPending} close={onClose} save={draft=>mutation.mutate(draft)}/>:<ModalBody><p>No hay configuración disponible para este local.</p></ModalBody>}
 </Dialog></div>;
}

function TipSettingsForm({locationName,policy,canManage,busy,close,save}:{locationName:string;policy:TipPolicy;canManage:boolean;busy:boolean;close:()=>void;save:(draft:TipSettings)=>void}){
 const[draft,setDraft]=useState<TipSettings>({mode:policy.mode,percentage:policy.percentage,showSuggestion:policy.mode==="suggest"||policy.showSuggestion,allowFixed:policy.allowFixed,channels:policy.channels});
 const disabled=!canManage||busy;
 return <ModalForm onSubmit={event=>{event.preventDefault();if(canManage&&!busy)save(draft)}} inert={busy}>
  <ModalBody><div className="tip-settings-form">
   <p className="tip-help">Local: <strong>{locationName}</strong></p>
   <TipSettingsFields value={draft} onChange={setDraft} disabled={disabled} currency={policy.currency} channelOptions={policy.channelOptions}/>
  </div></ModalBody>
  <ModalFooter><Button action="cancel" onClick={close} disabled={busy}>Cancelar</Button>{canManage&&<Button type="submit" disabled={busy} loading={busy}>{busy?"Guardando…":"Guardar"}</Button>}</ModalFooter>
 </ModalForm>;
}
