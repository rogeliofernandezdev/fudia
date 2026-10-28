"use client";
import {useState} from "react";
import {useQuery,useMutation,useQueryClient} from "@tanstack/react-query";
import {Button,FormField,Input,Select,Tabs,ErrorState,Skeleton,FormSection} from "@/design-system";
import {useSession,useFeedback} from "@/providers";
import {getTipSettings,saveTipSettings} from "../infrastructure/tips-api";
import type {TipPolicy,TipSettings} from "../domain/types";
import "./tips.css";

export function TipSettingsPanel(){
 const{organization,location,user,can}=useSession();
 const[scope,setScope]=useState<"organization"|"location">("location");
 const query=useQuery({queryKey:["tip-settings",organization?.id,location?.id,user?.id,scope],queryFn:()=>getTipSettings(scope),enabled:can("organizations.read"),refetchOnWindowFocus:false});
 if(!can("organizations.read"))return null;
 return <section className="panel tip-settings-panel" aria-label="Configuración de propinas">
  <FormSection icon="percent" title="Propinas" bare>
   <Tabs label="Alcance de la configuración de propinas" value={scope} onChange={setScope} items={[{value:"organization",label:"Restaurante",icon:"building"},{value:"location",label:"Local activo",icon:"store"}]}/>
   {query.isLoading?<div aria-label="Cargando configuración de propinas"><Skeleton className="tip-settings-skeleton"/></div>:query.isError?<ErrorState title="No pudimos cargar las propinas" text={query.error.message} onRetry={()=>void query.refetch()}/>:query.data?<TipSettingsForm key={`${scope}-${location?.id}-${query.dataUpdatedAt}`} scope={scope} policy={query.data} canManage={can("organizations.manage")}/>:<p>No hay configuración disponible.</p>}
  </FormSection>
 </section>;
}
function TipSettingsForm({scope,policy,canManage}:{scope:"organization"|"location";policy:TipPolicy;canManage:boolean}){
 const[draft,setDraft]=useState<TipSettings>({mode:policy.mode,percentage:policy.percentage,showSuggestion:policy.showSuggestion,allowFixed:policy.allowFixed,channels:policy.channels}),[inherit,setInherit]=useState(policy.inherited);
 const{notify}=useFeedback(),client=useQueryClient();
 const mutation=useMutation({mutationFn:()=>saveTipSettings(scope,{...draft,inherit:scope==="location"&&inherit}),onSuccess:()=>{void client.invalidateQueries({queryKey:["tip-settings"]});void client.invalidateQueries({queryKey:["pos-order"]});notify({tone:"success",title:"Propinas configuradas",message:"La configuración se aplicará a las próximas propinas."})},onError:error=>notify({tone:"danger",title:"No se pudo guardar",message:error.message})});
 const disabled=!canManage||mutation.isPending||(scope==="location"&&inherit);
 return <form onSubmit={event=>{event.preventDefault();if(canManage&&!mutation.isPending)mutation.mutate()}} className="tip-settings-form">
  {scope==="location"&&<label className="tip-checkbox"><input type="checkbox" checked={inherit} disabled={!canManage||mutation.isPending} onChange={event=>setInherit(event.target.checked)}/><span>Usar la configuración del restaurante</span></label>}
  <div className="tip-settings-grid"><FormField label="Funcionamiento"><Select disabled={disabled} value={draft.mode} onChange={event=>setDraft({...draft,mode:event.target.value as TipSettings["mode"]})}><option value="disabled">Desactivado</option><option value="suggest">Solo sugerencia</option><option value="collect">Registrar y gestionar propinas</option></Select></FormField>
  <FormField label="Porcentaje sugerido (%)"><Input type="number" step="0.01" min="0" max="100" required disabled={disabled} value={draft.percentage} onChange={event=>setDraft({...draft,percentage:Number(event.target.value)})}/></FormField></div>
  <label className="tip-checkbox"><input type="checkbox" checked={draft.showSuggestion} disabled={disabled} onChange={event=>setDraft({...draft,showSuggestion:event.target.checked})}/><span>Mostrar sugerencia en la precuenta</span></label>
  <label className="tip-checkbox"><input type="checkbox" checked={draft.allowFixed} disabled={disabled} onChange={event=>setDraft({...draft,allowFixed:event.target.checked})}/><span>Permitir monto libre elegido por el cliente</span></label>
  <fieldset className="tip-channels"><legend>Tipos de atención</legend>{policy.channelOptions.map(channel=><label className="tip-checkbox" key={channel.value}><input type="checkbox" checked={draft.channels.includes(channel.value)} disabled={disabled} onChange={event=>setDraft({...draft,channels:event.target.checked?[...draft.channels,channel.value]:draft.channels.filter(value=>value!==channel.value)})}/><span>{channel.label}</span></label>)}</fieldset>
  <p className="tip-help">La sugerencia no se cobra automáticamente. El cliente debe aceptar el importe. El monto libre usa {policy.currency}.</p>
  {canManage&&<div className="tip-settings-actions"><Button type="submit" disabled={mutation.isPending} loading={mutation.isPending}>{mutation.isPending?"Guardando…":"Guardar propinas"}</Button></div>}
 </form>;
}
