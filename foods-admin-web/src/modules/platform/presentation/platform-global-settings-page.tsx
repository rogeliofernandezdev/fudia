"use client";

import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,FormField,Input,PageHeader,Select} from "@/design-system";
import {useFeedback} from "@/providers";
import type {PlatformWhatsAppChannel,PlatformWhatsAppChannelDraft} from "../domain/types";
import {getPlatformGlobalCatalogs,listPlatformWhatsAppChannels,savePlatformWhatsAppChannel} from "../infrastructure/platform-api";

const blank:PlatformWhatsAppChannelDraft={countryCode:"PE",phoneNumber:"",phoneNumberId:"",displayName:"",secretRef:"",active:false};

export function PlatformGlobalSettingsPage(){
 const{notify}=useFeedback();
 const client=useQueryClient();
 const catalogs=useQuery({queryKey:["platform-global-catalogs"],queryFn:getPlatformGlobalCatalogs});
 const channels=useQuery({queryKey:["platform-whatsapp-channels"],queryFn:listPlatformWhatsAppChannels});
 const[draft,setDraft]=useState<PlatformWhatsAppChannelDraft|null>(null);
 const[editingId,setEditingId]=useState<string|undefined>();
 const save=useMutation({
  mutationFn:({value,id}:{value:PlatformWhatsAppChannelDraft;id?:string})=>savePlatformWhatsAppChannel(value,id),
  onSuccess:()=>{
   setDraft(null);setEditingId(undefined);
   void client.invalidateQueries({queryKey:["platform-whatsapp-channels"]});
   void client.invalidateQueries({queryKey:["platform-onboarding-context"]});
   notify({tone:"success",title:"Canal guardado",message:"La disponibilidad del país para onboarding se recalculará según el estado del canal."});
  },
  onError:e=>notify({tone:"danger",title:"No se pudo guardar",message:e.message}),
 });
 const edit=(item:PlatformWhatsAppChannel)=>{
  setEditingId(item.id);
  setDraft({countryCode:item.countryCode,phoneNumber:item.phoneNumber,phoneNumberId:item.phoneNumberId??"",displayName:item.displayName,secretRef:item.secretRef??"",active:item.active});
 };
 const start=()=>{setEditingId(undefined);setDraft({...blank,countryCode:catalogs.data?.countryOptions[0]?.code??"PE"});};
 const submit=()=>{
  if(!draft)return;
  if(!draft.countryCode||!draft.phoneNumber.trim()||!draft.displayName.trim()){
   notify({tone:"danger",title:"Faltan datos",message:"País, número y nombre visible son obligatorios."});return;
  }
  if(draft.active&&(!draft.phoneNumberId.trim()||!draft.secretRef.trim())){
   notify({tone:"danger",title:"Canal incompleto",message:"Para activarlo debes indicar Phone Number ID y referencia del secreto."});return;
  }
  save.mutate({value:draft,id:editingId});
 };

 return <>
  <PageHeader eyebrow="PLATAFORMA" title="Configuración Global" description="Administra los canales globales de WhatsApp por país. Solo los países con al menos un canal activo quedan disponibles para nuevas empresas." action={<Button icon="plus" onClick={start}>Nuevo canal</Button>}/>
  {(catalogs.isLoading||channels.isLoading)?<div className="panel catalog-state"><b>Cargando configuración global…</b></div>:
   (catalogs.isError||channels.isError)?<div className="panel catalog-state"><b>No pudimos cargar la configuración global.</b><Button kind="secondary" onClick={()=>{void catalogs.refetch();void channels.refetch()}}>Reintentar</Button></div>:
   <div className="panel">
    <div className="table-responsive"><table><thead><tr><th>País</th><th>Nombre</th><th>Número</th><th>Phone Number ID</th><th>Estado</th><th/></tr></thead><tbody>
     {(channels.data?.items??[]).map(item=><tr key={item.id}><td>{item.countryName} ({item.countryCode})</td><td>{item.displayName}</td><td>{item.phoneNumber}</td><td>{item.phoneNumberId??"Pendiente"}</td><td>{item.active?"Activo":"Borrador"}</td><td><Button kind="secondary" icon="edit" onClick={()=>edit(item)}>Editar</Button></td></tr>)}
     {!(channels.data?.items??[]).length&&<tr><td colSpan={6}>Todavía no hay canales configurados.</td></tr>}
    </tbody></table></div>
   </div>}

  {draft&&<div className="modal-backdrop"><section className="crud-modal" role="dialog" aria-modal="true" aria-labelledby="global-channel-title"><header><div><small>{editingId?"EDITAR CANAL":"NUEVO CANAL"}</small><h2 id="global-channel-title">Canal global de WhatsApp</h2></div></header>
   <div className="form-grid">
    <FormField label="País"><Select value={draft.countryCode} onChange={e=>setDraft({...draft,countryCode:e.target.value})}>{(catalogs.data?.countryOptions??[]).map(item=><option key={item.code} value={item.code}>{item.name} ({item.code}) · {item.defaultCurrency}</option>)}</Select></FormField>
    <FormField label="Nombre visible"><Input value={draft.displayName} onChange={e=>setDraft({...draft,displayName:e.target.value})} placeholder="FudIA Perú"/></FormField>
    <FormField label="Número WhatsApp" help="Formato internacional, por ejemplo +51914832364."><Input value={draft.phoneNumber} onChange={e=>setDraft({...draft,phoneNumber:e.target.value})}/></FormField>
    <FormField label="Phone Number ID" optional={!draft.active}><Input value={draft.phoneNumberId} onChange={e=>setDraft({...draft,phoneNumberId:e.target.value})}/></FormField>
    <FormField label="Referencia del secreto" help="Nombre o clave del secreto; nunca pegues aquí el token real." optional={!draft.active}><Input value={draft.secretRef} onChange={e=>setDraft({...draft,secretRef:e.target.value})} placeholder="META_WHATSAPP_PE_TOKEN"/></FormField>
    <label className="switch-row compact"><input type="checkbox" checked={draft.active} onChange={e=>setDraft({...draft,active:e.target.checked})}/><span/><b>Canal activo y país disponible para onboarding</b></label>
   </div>
   <footer><Button kind="ghost" disabled={save.isPending} onClick={()=>{setDraft(null);setEditingId(undefined)}}>Cancelar</Button><Button icon="check" disabled={save.isPending} onClick={submit}>{save.isPending?"Guardando…":"Guardar canal"}</Button></footer>
  </section></div>}
 </>;
}
