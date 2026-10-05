"use client";

import "./platform-global-settings.css";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,FormField,Icon,Input,PageHeader,RowActionButton,Select,Status} from "@/design-system";
import {useFeedback} from "@/providers";
import type {PlatformWhatsAppChannel,PlatformWhatsAppChannelDraft} from "../domain/types";
import {getPlatformGlobalCatalogs,listPlatformWhatsAppChannels,savePlatformWhatsAppChannel} from "../infrastructure/platform-api";

const blank:PlatformWhatsAppChannelDraft={countryCode:"PE",phoneNumber:"",phoneNumberId:"",displayName:"",active:false};

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
  setDraft({countryCode:item.countryCode,phoneNumber:item.phoneNumber,phoneNumberId:item.phoneNumberId??"",displayName:item.displayName,active:item.active});
 };
 const start=()=>{setEditingId(undefined);setDraft({...blank,countryCode:catalogs.data?.countryOptions[0]?.code??"PE"});};
 const submit=()=>{
  if(!draft)return;
  if(!draft.countryCode||!draft.phoneNumber.trim()||!draft.displayName.trim()){
   notify({tone:"danger",title:"Faltan datos",message:"País, número y nombre visible son obligatorios."});return;
  }
  if(draft.active&&!draft.phoneNumberId.trim()){
   notify({tone:"danger",title:"Canal incompleto",message:"Para activarlo debes indicar el Phone Number ID."});return;
  }
  save.mutate({value:draft,id:editingId});
 };
 const selectedCountry=catalogs.data?.countryOptions.find(country=>country.code===draft?.countryCode);

 return <>
  <PageHeader eyebrow="PLATAFORMA" title="Países y WhatsApp" description="Administra el número de WhatsApp de cada país y consulta su moneda predeterminada. Las credenciales de Meta se mantienen únicamente en las variables de entorno." action={<Button icon="plus" onClick={start}>Nuevo canal</Button>}/>
  {(catalogs.isLoading||channels.isLoading)?<div className="panel catalog-state"><b>Cargando configuración global…</b></div>:
   (catalogs.isError||channels.isError)?<div className="panel catalog-state"><b>No pudimos cargar la configuración global.</b><Button kind="secondary" onClick={()=>{void catalogs.refetch();void channels.refetch()}}>Reintentar</Button></div>:
   <div className="panel">
    <div className="table-responsive"><table><thead><tr><th>País</th><th>Moneda</th><th>Nombre</th><th>Número</th><th>Phone Number ID</th><th>Estado</th><th>Acciones</th></tr></thead><tbody>
     {(channels.data?.items??[]).map(item=><tr key={item.id}><td>{item.countryName} ({item.countryCode})</td><td>{catalogs.data?.countryOptions.find(country=>country.code===item.countryCode)?.defaultCurrency??"—"}</td><td>{item.displayName}</td><td>{item.phoneNumber}</td><td>{item.phoneNumberId??"Pendiente"}</td><td><Status tone={item.active?"green":"gray"}>{item.active?"Activo":"Borrador"}</Status></td><td><div className="table-actions"><RowActionButton action="edit" label={`Editar canal ${item.displayName}`} onClick={()=>edit(item)}/></div></td></tr>)}
     {!(channels.data?.items??[]).length&&<tr><td colSpan={7}>Todavía no hay canales configurados.</td></tr>}
    </tbody></table></div>
   </div>}

  {draft&&<div className="modal-backdrop modal-overlay-in"><section className="crud-modal platform-channel-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="global-channel-title" aria-busy={save.isPending}><div className="modal-accent"/><header><span className="modal-title-icon"><Icon name="chat" size={18}/></span><div><small>{editingId?"EDITAR CANAL":"NUEVO CANAL"}</small><h2 id="global-channel-title">Canal global de WhatsApp</h2></div><button type="button" aria-label="Cerrar" disabled={save.isPending} onClick={()=>{setDraft(null);setEditingId(undefined)}}><Icon name="close"/></button></header>
   <form onSubmit={event=>{event.preventDefault();submit()}}>
    <div className="form-grid platform-channel-fields">
     <FormField label="País"><Select value={draft.countryCode} onChange={e=>setDraft({...draft,countryCode:e.target.value})}>{(catalogs.data?.countryOptions??[]).map(item=><option key={item.code} value={item.code}>{item.name} ({item.code})</option>)}</Select></FormField>
     <FormField label="Moneda asociada" help="Se asigna automáticamente según el país."><Input value={selectedCountry?.defaultCurrency??""} readOnly tabIndex={-1}/></FormField>
     <FormField className="span-2" label="Nombre visible"><Input value={draft.displayName} onChange={e=>setDraft({...draft,displayName:e.target.value})} placeholder={selectedCountry?`FudIA ${selectedCountry.name}`:"Nombre del canal"}/></FormField>
     <FormField label="Número WhatsApp" help="Formato internacional, por ejemplo +51914832364."><Input value={draft.phoneNumber} onChange={e=>setDraft({...draft,phoneNumber:e.target.value})}/></FormField>
     <FormField label="Phone Number ID" help="Identificador no secreto del número en Meta." optional={!draft.active}><Input value={draft.phoneNumberId} onChange={e=>setDraft({...draft,phoneNumberId:e.target.value})}/></FormField>
     <label className="switch-row compact span-2"><input type="checkbox" checked={draft.active} onChange={e=>setDraft({...draft,active:e.target.checked})}/><span/><b>Canal activo y país disponible para onboarding</b></label>
    </div>
    <footer><Button type="button" kind="ghost" disabled={save.isPending} onClick={()=>{setDraft(null);setEditingId(undefined)}}>Cancelar</Button><Button type="submit" icon="check" disabled={save.isPending}>{save.isPending?"Guardando…":"Guardar"}</Button></footer>
   </form>
  </section></div>}
 </>;
}
