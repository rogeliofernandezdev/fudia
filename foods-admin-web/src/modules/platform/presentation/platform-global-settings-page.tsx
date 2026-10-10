"use client";

import {Dialog} from "@/design-system/dialog";
import "./platform-global-settings.css";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import CountrySelect from "react-select";
import {createLookupSelectStyles} from "@/design-system/search-select-styles";
import {Button,ConfirmDialog,EmptyState,ErrorState,FormField,Icon,Input,PageHeader,RowActionButton,Status,TableSkeleton} from "@/design-system";
import {useFeedback} from "@/providers";
import type {PlatformWhatsAppChannel,PlatformWhatsAppChannelDraft} from "../domain/types";
import {getPlatformGlobalCatalogs,listPlatformWhatsAppChannels,savePlatformWhatsAppChannel} from "../infrastructure/platform-api";

type CountryLookupOption={label:string;value:string};
const countrySelectStyles=createLookupSelectStyles<CountryLookupOption>();
const blank:PlatformWhatsAppChannelDraft={countryCode:"PE",phoneNumber:"",displayName:"",active:false};
const columns=["País","Moneda","Nombre","Número","Estado","Acciones"];
const toDraft=(item:PlatformWhatsAppChannel):PlatformWhatsAppChannelDraft=>({countryCode:item.countryCode,phoneNumber:item.phoneNumber,displayName:item.displayName,active:item.active});

export function PlatformGlobalSettingsPage(){
 const{notify}=useFeedback();
 const client=useQueryClient();
 const catalogs=useQuery({queryKey:["platform-global-catalogs"],queryFn:getPlatformGlobalCatalogs});
 const channels=useQuery({queryKey:["platform-whatsapp-channels"],queryFn:listPlatformWhatsAppChannels});
 const[draft,setDraft]=useState<PlatformWhatsAppChannelDraft|null>(null);
 const[editingId,setEditingId]=useState<string|undefined>();
 const[validationError,setValidationError]=useState("");
 const[target,setTarget]=useState<PlatformWhatsAppChannel|null>(null);
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
 const toggle=useMutation({
  mutationFn:(item:PlatformWhatsAppChannel)=>savePlatformWhatsAppChannel({...toDraft(item),active:!item.active},item.id),
  onSuccess:(_,item)=>{
   setTarget(null);
   void client.invalidateQueries({queryKey:["platform-whatsapp-channels"]});
   void client.invalidateQueries({queryKey:["platform-onboarding-context"]});
   notify({tone:"success",title:item.active?"Canal desactivado":"Canal activado",message:item.active?"El país dejó de estar disponible para registrar empresas.":"El país quedó disponible para registrar empresas."});
  },
  onError:e=>notify({tone:"danger",title:"No se pudo actualizar",message:e.message}),
 });
 const edit=(item:PlatformWhatsAppChannel)=>{
  setValidationError("");
  setEditingId(item.id);
  setDraft(toDraft(item));
 };
 const start=()=>{
  setValidationError("");
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
   setValidationError("País, número y nombre visible son obligatorios.");return;
  }
  setValidationError("");
  save.mutate({value:draft,id:editingId});
 };
 const selectedCountry=catalogs.data?.countryOptions.find(country=>country.code===draft?.countryCode);
 const countryOptions=(catalogs.data?.countryOptions??[]).map(country=>({value:country.code,label:`${country.name} (${country.code})`}));
 const selectedCountryOption=countryOptions.find(country=>country.value===draft?.countryCode)??null;
 const items=channels.data?.items??[];
 const currencyOf=(code:string)=>catalogs.data?.countryOptions.find(country=>country.code===code)?.defaultCurrency??"—";
 const status=(item:PlatformWhatsAppChannel)=><Status active={item.active}>{item.active?"Activo":"Inactivo"}</Status>;
 const actions=(item:PlatformWhatsAppChannel)=><><RowActionButton action="edit" label={`Editar canal ${item.displayName}`} onClick={()=>edit(item)}/><RowActionButton action={item.active?"deactivate":"activate"} stateLabel={`Estado del canal ${item.displayName}`} onClick={()=>setTarget(item)}/></>;

 return <>
  <PageHeader title="Países y WhatsApp" description="Administra el número visible de WhatsApp de cada país y consulta su moneda predeterminada. Toda la configuración técnica de Meta se mantiene únicamente en variables de entorno." action={<Button icon="plus" onClick={start}>Nuevo canal</Button>}/>
  <section className="panel management standardized-management platform-channels-panel">
   {(catalogs.isLoading||channels.isLoading)?<TableSkeleton columns={columns} rows={4} label="Cargando canales de WhatsApp"/>:
    (catalogs.isError||channels.isError)?<ErrorState title="No pudimos cargar la configuración global" text={(catalogs.error??channels.error)?.message} retrying={catalogs.isFetching||channels.isFetching} onRetry={()=>{void catalogs.refetch();void channels.refetch()}}/>:
    !items.length?<EmptyState icon="chat" title="Aún no hay canales" text="Registra el número de WhatsApp de un país para habilitarlo en el registro de empresas." action={<Button icon="plus" onClick={start}>Nuevo canal</Button>}/>:
    <><div className="table-wrap hover-scroll"><table><thead><tr><th>País</th><th>Moneda</th><th>Nombre</th><th>Número</th><th>Estado</th><th>Acciones</th></tr></thead><tbody>
     {items.map((item,index)=><tr className={index%2?"alternate":""} key={item.id}><td><span className={`row-icon r${index%3}`}><Icon name="chat" size={18}/></span><b>{item.countryName}</b><small>{item.countryCode}</small></td><td>{currencyOf(item.countryCode)}</td><td>{item.displayName}</td><td>{item.phoneNumber}</td><td>{status(item)}</td><td><div className="table-actions">{actions(item)}</div></td></tr>)}
    </tbody></table></div>
    <div className="management-cards">{items.map(item=><article key={item.id}><header><span className="row-icon r0"><Icon name="chat"/></span><div><b>{item.countryName}</b><small>{item.countryCode}</small></div>{status(item)}</header><dl><div><dt>Nombre</dt><dd>{item.displayName}</dd></div><div><dt>Número</dt><dd>{item.phoneNumber}</dd></div><div><dt>Moneda</dt><dd>{currencyOf(item.countryCode)}</dd></div></dl><footer>{actions(item)}</footer></article>)}</div></>}
  </section>

  {draft&&<div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={()=>{setDraft(null);setEditingId(undefined)}} className="crud-modal platform-channel-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="global-channel-title" aria-busy={save.isPending}><div className="modal-accent"/><header><span className="modal-title-icon"><Icon name="chat" size={18}/></span><div><small>{editingId?"EDITAR CANAL":"NUEVO CANAL"}</small><h2 id="global-channel-title">Canal global de WhatsApp</h2></div><button type="button" aria-label="Cerrar" disabled={save.isPending} onClick={()=>{setDraft(null);setEditingId(undefined)}}><Icon name="close"/></button></header>
   <form onSubmit={event=>{event.preventDefault();submit()}}>
    <div className="form-grid platform-channel-fields">
     {validationError&&<p className="field-error span-2" role="alert">{validationError}</p>}
     <FormField as="div" label="País"><CountrySelect<CountryLookupOption,false> inputId="platform-channel-country" instanceId="platform-channel-country-autocomplete" aria-label="País" options={countryOptions} value={selectedCountryOption} onChange={option=>changeCountry(option?.value??"")} isSearchable placeholder="Buscar país..." noOptionsMessage={()=>"No hay coincidencias"} styles={countrySelectStyles} className="react-select-container platform-country-autocomplete" classNamePrefix="rs" menuPortalTarget={typeof document==="undefined"?undefined:document.body} menuPosition="fixed"/></FormField>
     <FormField label="Moneda asociada" help="Se asigna automáticamente según el país."><Input value={selectedCountry?.defaultCurrency??""} readOnly tabIndex={-1}/></FormField>
     <FormField className="span-2" label="Nombre visible"><Input value={draft.displayName} onChange={e=>setDraft({...draft,displayName:e.target.value})}/></FormField>
     <FormField className="span-2" label="Número WhatsApp"><Input value={draft.phoneNumber} onChange={e=>setDraft({...draft,phoneNumber:e.target.value})} placeholder="987654321"/></FormField>
    </div>
    <footer><Button type="button" kind="ghost" disabled={save.isPending} onClick={()=>{setDraft(null);setEditingId(undefined)}}>Cancelar</Button><Button type="submit" disabled={save.isPending}>{save.isPending?"Guardando…":"Guardar"}</Button></footer>
   </form>
  </Dialog></div>}
  <ConfirmDialog open={Boolean(target)} title={target?.active?"Desactivar canal":"Activar canal"} subject={target?{label:"Canal",name:`${target.displayName} · ${target.phoneNumber}`}:undefined} description={target?.active?"El país dejará de estar disponible para registrar empresas nuevas.":"El país quedará disponible para registrar empresas nuevas."} tone={target?.active?"danger":"success"} confirmLabel={target?.active?"Desactivar":"Activar"} pending={toggle.isPending} onCancel={()=>setTarget(null)} onConfirm={()=>target&&toggle.mutate(target)}/>
 </>;
}
