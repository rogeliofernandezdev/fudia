"use client";

import "./platform-global-settings.css";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import CountrySelect from "react-select";
import {createLookupSelectStyles} from "@/design-system/search-select-styles";
import {Button,FormField,Icon,Input,PageHeader,RowActionButton,Status} from "@/design-system";
import {useFeedback} from "@/providers";
import type {PlatformWhatsAppChannel,PlatformWhatsAppChannelDraft} from "../domain/types";
import {getPlatformGlobalCatalogs,listPlatformWhatsAppChannels,savePlatformWhatsAppChannel} from "../infrastructure/platform-api";

type CountryLookupOption={label:string;value:string};
const countrySelectStyles=createLookupSelectStyles<CountryLookupOption>();
const blank:PlatformWhatsAppChannelDraft={countryCode:"PE",phoneNumber:"",displayName:"",active:false};

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
  setDraft({countryCode:item.countryCode,phoneNumber:item.phoneNumber,displayName:item.displayName,active:item.active});
 };
 const start=()=>{
  const country=catalogs.data?.countryOptions.find(item=>item.code==="PE")??catalogs.data?.countryOptions[0];
  setEditingId(undefined);
  setDraft(country?{...blank,countryCode:country.code,displayName:country.name,phoneNumber:country.callingCode}:{...blank});
 };
 const changeCountry=(countryCode:string)=>{
  const nextCountry=catalogs.data?.countryOptions.find(country=>country.code===countryCode);
  setDraft(current=>{
   if(!current)return current;
   const currentCountry=catalogs.data?.countryOptions.find(country=>country.code===current.countryCode);
   let localNumber=current.phoneNumber;
   if(currentCountry?.callingCode&&localNumber.startsWith(currentCountry.callingCode)){
    localNumber=localNumber.slice(currentCountry.callingCode.length);
   }else if(localNumber.startsWith("+")){
    localNumber="";
   }
   return {
    ...current,
    countryCode,
    displayName:nextCountry?.name??"",
    phoneNumber:nextCountry?.callingCode?`${nextCountry.callingCode}${localNumber}`:localNumber,
   };
  });
 };
 const submit=()=>{
  if(!draft)return;
  if(!draft.countryCode||!draft.phoneNumber.trim()||!draft.displayName.trim()){
   notify({tone:"danger",title:"Faltan datos",message:"País, número y nombre visible son obligatorios."});return;
  }
  save.mutate({value:draft,id:editingId});
 };
 const selectedCountry=catalogs.data?.countryOptions.find(country=>country.code===draft?.countryCode);
 const countryOptions=(catalogs.data?.countryOptions??[]).map(country=>({value:country.code,label:`${country.name} (${country.code})`}));
 const selectedCountryOption=countryOptions.find(country=>country.value===draft?.countryCode)??null;

 return <>
  <PageHeader eyebrow="PLATAFORMA" title="Países y WhatsApp" description="Administra el número visible de WhatsApp de cada país y consulta su moneda predeterminada. Toda la configuración técnica de Meta se mantiene únicamente en variables de entorno." action={<Button icon="plus" onClick={start}>Nuevo canal</Button>}/>
  {(catalogs.isLoading||channels.isLoading)?<div className="panel catalog-state"><b>Cargando configuración global…</b></div>:
   (catalogs.isError||channels.isError)?<div className="panel catalog-state"><b>No pudimos cargar la configuración global.</b><Button kind="secondary" onClick={()=>{void catalogs.refetch();void channels.refetch()}}>Reintentar</Button></div>:
   <div className="panel">
    <div className="table-responsive"><table><thead><tr><th>País</th><th>Moneda</th><th>Nombre</th><th>Número</th><th>Estado</th><th>Acciones</th></tr></thead><tbody>
     {(channels.data?.items??[]).map(item=><tr key={item.id}><td>{item.countryName} ({item.countryCode})</td><td>{catalogs.data?.countryOptions.find(country=>country.code===item.countryCode)?.defaultCurrency??"—"}</td><td>{item.displayName}</td><td>{item.phoneNumber}</td><td><Status active={item.active}>{item.active?"Activo":"Borrador"}</Status></td><td><div className="table-actions"><RowActionButton action="edit" label={`Editar canal ${item.displayName}`} onClick={()=>edit(item)}/></div></td></tr>)}
     {!(channels.data?.items??[]).length&&<tr><td colSpan={6}>Todavía no hay canales configurados.</td></tr>}
    </tbody></table></div>
   </div>}

  {draft&&<div className="modal-backdrop modal-overlay-in"><section className="crud-modal platform-channel-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="global-channel-title" aria-busy={save.isPending}><div className="modal-accent"/><header><span className="modal-title-icon"><Icon name="chat" size={18}/></span><div><small>{editingId?"EDITAR CANAL":"NUEVO CANAL"}</small><h2 id="global-channel-title">Canal global de WhatsApp</h2></div><button type="button" aria-label="Cerrar" disabled={save.isPending} onClick={()=>{setDraft(null);setEditingId(undefined)}}><Icon name="close"/></button></header>
   <form onSubmit={event=>{event.preventDefault();submit()}}>
    <div className="form-grid platform-channel-fields">
     <FormField as="div" label="País"><CountrySelect<CountryLookupOption,false> inputId="platform-channel-country" instanceId="platform-channel-country-autocomplete" aria-label="País" options={countryOptions} value={selectedCountryOption} onChange={option=>changeCountry(option?.value??"")} isSearchable placeholder="Buscar país..." noOptionsMessage={()=>"No hay coincidencias"} styles={countrySelectStyles} className="react-select-container platform-country-autocomplete" classNamePrefix="rs" menuPortalTarget={typeof document==="undefined"?undefined:document.body} menuPosition="fixed"/></FormField>
     <FormField label="Moneda asociada" help="Se asigna automáticamente según el país."><Input value={selectedCountry?.defaultCurrency??""} readOnly tabIndex={-1}/></FormField>
     <FormField className="span-2" label="Nombre visible"><Input value={draft.displayName} onChange={e=>setDraft({...draft,displayName:e.target.value})}/></FormField>
     <FormField className="span-2" label="Número WhatsApp"><Input value={draft.phoneNumber} onChange={e=>setDraft({...draft,phoneNumber:e.target.value})} placeholder="987654321"/></FormField>
     <label className="switch-row compact span-2"><input type="checkbox" checked={draft.active} onChange={e=>setDraft({...draft,active:e.target.checked})}/><span/><b>Canal activo y país disponible para onboarding</b></label>
    </div>
    <footer><Button type="button" kind="ghost" disabled={save.isPending} onClick={()=>{setDraft(null);setEditingId(undefined)}}>Cancelar</Button><Button type="submit" disabled={save.isPending}>{save.isPending?"Guardando…":"Guardar"}</Button></footer>
   </form>
  </section></div>}
 </>;
}