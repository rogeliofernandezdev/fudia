"use client";

import {useEffect, useRef, useState} from "react";
import {useQuery} from "@tanstack/react-query";
import CountrySelect from "react-select";
import {Button, ConfirmDialog, ErrorState, FormField, Input, ModalBody, ModalForm, ModalFooter, ModalHeader, Select, Skeleton} from "@/design-system";
import {Dialog} from "@/design-system/dialog";
import {useSession} from "@/providers";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {createLookupSelectStyles} from "@/design-system/search-select-styles";
import type {Country, PlatformWhatsAppChannel, PlatformWhatsAppChannelDraft, WhatsAppProviderOption} from "../domain/types";
import {linkPlatformWhatsApp, listWhatsAppChannelHistory} from "../infrastructure/platform-api";

type CountryOption={label:string;value:string};
const countryStyles=createLookupSelectStyles<CountryOption>();
export const connectionLabel=(state:string)=>({connected:"Conectado",pairing:"Esperando vinculación",pending:"Esperando vinculación",connecting:"Conectando",disconnected:"Desconectado",unlinked:"Sin vincular",unconfigured:"Sin configurar",unavailable:"No disponible",number_mismatch:"El número no coincide"}[state]??"Sin verificar");

type Props={initial:PlatformWhatsAppChannelDraft;existing?:PlatformWhatsAppChannel;countries:Country[];usedCountries:string[];providers:WhatsAppProviderOption[];pending:boolean;onClose:()=>void;onSave:(draft:PlatformWhatsAppChannelDraft)=>void};

export function WhatsAppChannelEditor({initial,existing,countries,usedCountries,providers,pending,onClose,onSave}:Props){
 const [draft,setDraft]=useState(initial);
 const {location}=useSession();
 const [error,setError]=useState("");
 const [confirm,setConfirm]=useState<PlatformWhatsAppChannelDraft|null>(null);
 const [linking,setLinking]=useState(false);
 const sessionRef=useRef("");
 const [qrError,setQrError]=useState("");
 const canvas=useRef<HTMLCanvasElement>(null);
 const link=useQuery({queryKey:["whatsapp-device-link",draft.countryCode,linking],queryFn:async()=>{const data=await linkPlatformWhatsApp(draft.countryCode,sessionRef.current||undefined);sessionRef.current=data.sessionRef??sessionRef.current;return data},enabled:linking&&draft.provider==="WHATSMEOW",refetchInterval:query=>query.state.data?.state==="connected"?false:3000,retry:false,gcTime:0});
 const history=useQuery({queryKey:["whatsapp-channel-history",existing?.id],queryFn:()=>listWhatsAppChannelHistory(existing!.id),enabled:Boolean(existing),gcTime:0});
 const code=link.data?.qrCode;
 const value=linking&&draft.provider==="WHATSMEOW"&&link.data?.state==="connected"?{...draft,sessionRef:link.data.sessionRef,phoneNumber:link.data.phoneNumber}:draft;
 useEffect(()=>{
  if(!code||!canvas.current)return;
  let cancelled=false;
  void import("qrcode").then(module=>{
   if(cancelled||!canvas.current)return;
   return module.default.toCanvas(canvas.current,code,{width:240,margin:4});
  }).catch(()=>{if(!cancelled)setQrError("No pudimos mostrar el QR. Vuelve a consultar la vinculación.")});
  return()=>{cancelled=true};
 },[code]);
 const countryOptions=countries.filter(item=>existing?.countryCode===item.code||!usedCountries.includes(item.code)).map(item=>({value:item.code,label:`${item.name} (${item.code})`}));
 const country=countries.find(item=>item.code===draft.countryCode);
 const changeCountry=(countryCode:string)=>{
  const next=countries.find(item=>item.code===countryCode);
  setLinking(false);sessionRef.current="";
  setDraft(current=>({...current,countryCode,displayName:next?.name??"",phoneNumber:next?.callingCode??"",sessionRef:undefined}));
 };
 const submit=()=>{
  if(!draft.countryCode||!draft.displayName.trim()||(draft.provider!=="FUDIA_WEB"&&!/^\+?[1-9][0-9 ()-]{7,23}$/.test(value.phoneNumber.trim()))){setError("Completa el país, nombre y número con código de país.");return}
  if(draft.provider==="WHATSMEOW"&&draft.active&&!value.sessionRef&&(!existing||existing.provider!==draft.provider||existing.phoneNumber!==value.phoneNumber)){setError("Vincula el número antes de activar WhatsApp.");return}
  setError("");
  if(existing&&(existing.phoneNumber!==value.phoneNumber||existing.provider!==draft.provider)){setConfirm({...value});return}
  onSave(draft.provider==="FUDIA_WEB"?{...value,phoneNumber:""}:value);
 };
 return <><div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={onClose} className="crud-modal platform-channel-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="global-channel-title" aria-busy={pending}>
  <ModalHeader title={existing?"Editar canal":"Nuevo canal"} titleId="global-channel-title" onClose={onClose} busy={pending} closeLabel="Cerrar"/>
  <ModalForm onSubmit={event=>{event.preventDefault();submit()}}>
   <ModalBody><div className="form-grid platform-channel-fields">
    {error&&<p className="field-error span-2" role="alert">{error}</p>}
    {existing?.needsReconciliation&&<p className="form-help span-2" role="status">La configuración anterior requiere verificación. Revisa el número y el proveedor antes de activarla.</p>}
    <FormField as="div" label="País"><CountrySelect<CountryOption,false> inputId="platform-channel-country" instanceId="platform-channel-country-autocomplete" aria-label="País" options={countryOptions} value={countryOptions.find(item=>item.value===draft.countryCode)??null} onChange={option=>changeCountry(option?.value??"")} isDisabled={Boolean(existing)||pending} isSearchable placeholder="Buscar país..." noOptionsMessage={()=>"Todos los países disponibles ya tienen una configuración"} styles={countryStyles} className="react-select-container platform-country-autocomplete" classNamePrefix="rs" menuPortalTarget={typeof document==="undefined"?undefined:document.body} menuPosition="fixed"/></FormField>
    <FormField label="Moneda asociada" help="Se asigna automáticamente según el país."><Input value={country?.defaultCurrency??""} readOnly tabIndex={-1}/></FormField>
    <FormField className="span-2" label="Nombre visible"><Input value={draft.displayName} disabled={pending} onChange={e=>setDraft({...draft,displayName:e.target.value})}/></FormField>
    <FormField className="span-2" label="Proveedor"><Select value={draft.provider} disabled={pending} onChange={e=>{setLinking(false);sessionRef.current="";setDraft({...draft,provider:e.target.value as PlatformWhatsAppChannelDraft["provider"],phoneNumber:e.target.value==="FUDIA_WEB"?"":draft.phoneNumber,sessionRef:undefined})}}>{providers.map(item=><option key={item.value} value={item.value}>{item.label}</option>)}</Select></FormField>
    {draft.provider!=="FUDIA_WEB"&&<FormField className="span-2" label="Número WhatsApp" help={draft.provider==="META_CLOUD_API"?"Al activar, verificaremos que el número esté registrado en una cuenta Meta autorizada.":"La vinculación completará este campo con el número real del dispositivo."}><Input type="tel" autoComplete="tel" value={value.phoneNumber} disabled={pending||Boolean(linking&&link.data?.state==="connected")} onChange={e=>setDraft({...draft,phoneNumber:e.target.value,sessionRef:undefined})} placeholder="+51 987 654 321"/></FormField>}
    {draft.provider==="FUDIA_WEB"&&<p className="form-help span-2">El QR abrirá directamente el chat del restaurante en el navegador. No requiere un número de WhatsApp.</p>}
    {draft.provider==="WHATSMEOW"&&<section className="whatsapp-link span-2" aria-label="Vinculación de dispositivo">
     <p>En WhatsApp, abre Dispositivos vinculados y escanea este QR.</p>
     <p className="form-help">Este QR privado vincula el número de Fudia. Los QR de los clientes se conservan.</p>
     <Button kind="secondary" loading={link.isFetching} disabled={pending||link.isFetching} onClick={()=>{if(linking){if(["unlinked","disconnected"].includes(link.data?.state??""))sessionRef.current="";void link.refetch()}else{setQrError("");setLinking(true)}}}>{link.isFetching?"Consultando…":linking?"Consultar vinculación":"Vincular WhatsApp"}</Button>
     {link.isFetching&&!link.data&&<Skeleton className="whatsapp-device-placeholder"/>}
     {link.isError&&<ErrorState title="No pudimos consultar WhatsApp" text={link.error.message} onRetry={()=>void link.refetch()}/>}
     {link.data&&<p role="status">{connectionLabel(link.data.state)}{link.data.state==="connected"&&` · ${link.data.phoneNumber}`}</p>}
     {code&&link.data?.state!=="connected"&&<><canvas ref={canvas} className="whatsapp-device-qr" aria-label="QR privado para vincular WhatsApp" role="img"/><p className="form-help">El QR se actualiza automáticamente mientras esperas. No lo compartas.</p></>}
     {qrError&&<p className="field-error" role="alert">{qrError}</p>}
    </section>}
    <label className="switch-row span-2"><input type="checkbox" checked={draft.active} disabled={pending} onChange={e=>setDraft({...draft,active:e.target.checked})}/><span aria-hidden="true"/><b>Activar canal para este país</b></label>
    {existing&&<details className="whatsapp-history span-2"><summary>Historial de configuración</summary>{history.isLoading?<Skeleton className="whatsapp-history-placeholder"/>:history.isError?<ErrorState title="No pudimos cargar el historial" onRetry={()=>void history.refetch()}/>:!history.data?.items.length?<p>Aún no hay cambios registrados.</p>:<ol>{history.data.items.map((item,index)=><li key={`${item.createdAt}-${index}`}><time>{formatRegionalDateTime(item.createdAt,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium",timeStyle:"short"})}</time><p>{providers.find(provider=>provider.value===item.after.provider)?.label??"Configuración anterior"} · {item.after.phoneNumber??"Sin número"} · {item.after.active?"Activo":"Inactivo"}</p></li>)}</ol>}</details>}
   </div></ModalBody>
   <ModalFooter><Button kind="ghost" disabled={pending} onClick={onClose}>Cancelar</Button><Button type="submit" disabled={pending||!providers.length} loading={pending}>{pending?"Guardando…":"Guardar"}</Button></ModalFooter>
  </ModalForm>
 </Dialog></div>
 <ConfirmDialog open={Boolean(confirm)} title="Cambiar canal del país" description={`Los QR impresos abrirán ${confirm?.provider==="FUDIA_WEB"?"el chat web del restaurante":confirm?.phoneNumber??value.phoneNumber} cuando el canal esté activo. Los chats y avisos anteriores conservarán su canal original; los pedidos permanecen intactos.`} tone="success" confirmLabel="Confirmar cambio" pending={pending} onCancel={()=>setConfirm(null)} onConfirm={()=>{if(confirm){onSave(confirm);setConfirm(null)}}}/>
 </>;
}
