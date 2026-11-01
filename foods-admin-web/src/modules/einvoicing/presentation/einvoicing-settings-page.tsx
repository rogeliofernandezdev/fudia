"use client";
import Link from "next/link";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,EmptyState,ErrorState,FormField,Icon,InlineNotice,Input,ModalBody,ModalFooter,ModalForm,ModalHeader,PageHeader,RowActionButton,Select,Status,TableSkeleton} from "@/design-system";
import {DataTable} from "@/design-system/data-table";
import {Dialog} from "@/design-system/dialog";
import {useFeedback,useSession} from "@/providers";
import {pageRoutes} from "@/shared/routing/page-routes";
import {formatRegionalCalendarDate} from "@/shared/i18n/regional-format";
import type {CredentialField,EInvoicingSettings,FiscalDocumentType,LocationSetting,ProfileSetting,ProfileSettingsInput} from "../domain/types";
import {getEInvoicingSettings,saveCredentials,saveLocationSettings,saveProfileSettings} from "../infrastructure/einvoicing-api";
import "./einvoicing.css";

const locationColumns=["Local","Series","Al cobrar","Acciones"];
const defaultLabels:Record<FiscalDocumentType,string>={none:"Sin comprobante",receipt:"Boleta",invoice:"Factura"};

export function EInvoicingSettingsPage(){
 const{organization,can}=useSession();
 const canManage=can("organizations.manage");
 const[editing,setEditing]=useState<LocationSetting|null>(null);
 const[credentials,setCredentials]=useState<ProfileSetting|null>(null);
 const query=useQuery({queryKey:["einvoicing-settings",organization?.id],queryFn:getEInvoicingSettings});
 const data=query.data;
 return <>
  <PageHeader title="Facturación" description="Comprobantes electrónicos por país y series de cada local."/>
  {query.isLoading?<section className="panel management"><TableSkeleton columns={locationColumns} label="Cargando facturación"/></section>:
   query.isError||!data?<section className="panel management"><ErrorState title="No pudimos cargar la facturación" text={query.error?.message??""} onRetry={()=>void query.refetch()} retrying={query.isFetching}/></section>:
   !data.entitled?<section className="panel management"><EmptyState icon="receipt" title="Facturación electrónica no incluida" text="Está disponible desde el plan Impulso. Mientras tanto, los cobros se registran con su ticket de pago."/></section>:
   <div className="einvoicing-layout">
    {data.profiles.map(profile=><ProfilePanel key={profile.fiscalProfileId} profile={profile} settings={data} canManage={canManage} onCredentials={()=>setCredentials(profile)}/>)}
    <section className="panel management catalog-panel">
     <header className="einvoicing-section-header"><h2 className="modal-section-title">Series por local</h2></header>
     {!data.locations.length?<EmptyState icon="store" title="Sin locales activos" text="Registra un local para configurar sus series."/>:
     <DataTable label="Series por local" containerClassName="hover-scroll"><thead><tr>{locationColumns.map(column=><th key={column}>{column}</th>)}</tr></thead><tbody>{data.locations.map((location,i)=><tr className={i%2?"alternate":""} key={location.locationId}>
      <td><span className={`row-icon r${i%3}`}><Icon name="store" size={18}/></span><b>{location.name}</b>{data.profiles.length>1&&<small>{data.profiles.find(profile=>profile.fiscalProfileId===location.fiscalProfileId)?.countryName??location.country}</small>}</td>
      <td className="einvoicing-series-cell">{!location.supported?"No disponible en este país":location.series.map(series=><span key={series.documentType}>{series.label} <b>{series.series||"Sin serie"}</b><br/></span>)}</td>
      <td>{defaultLabels[location.defaultDocument]}</td>
      <td><div className="table-actions">{canManage&&location.supported&&<RowActionButton action="edit" onClick={()=>setEditing(location)}/>}</div></td>
     </tr>)}</tbody></DataTable>}
    </section>
   </div>}
  {credentials&&data&&<CredentialsDialog key={credentials.fiscalProfileId} profile={credentials} providerLabel={data.providerOptions.find(option=>option.value===credentials.provider)?.label??credentials.provider} close={()=>setCredentials(null)}/>}
  {editing&&data&&<LocationSeriesDialog key={editing.locationId} location={editing} documentTypes={editing.series} close={()=>setEditing(null)}/>}
 </>;
}

function ProfilePanel({profile,settings,canManage,onCredentials}:{profile:ProfileSetting;settings:EInvoicingSettings;canManage:boolean;onCredentials:()=>void}){
 const{notify}=useFeedback();const client=useQueryClient();
 const[draft,setDraft]=useState<ProfileSettingsInput>({enabled:profile.enabled,provider:profile.provider,environment:profile.environment});
 const dirty=draft.enabled!==profile.enabled||draft.provider!==profile.provider||draft.environment!==profile.environment;
 const save=useMutation({mutationFn:()=>saveProfileSettings(profile.fiscalProfileId,draft),onSuccess:()=>{void client.invalidateQueries({queryKey:["einvoicing-settings"]});void client.invalidateQueries({queryKey:["einvoicing-checkout"]});notify({tone:"success",title:"Facturación guardada",message:draft.enabled?"Los locales de este país ya pueden emitir comprobantes.":"Los cobros de este país ya no emiten comprobantes."})},onError:error=>notify({tone:"danger",title:"No se pudo guardar",message:error.message})});
 const locked=!canManage||!profile.supported||save.isPending;
 return <section className="panel management einvoicing-profile-panel">
  <form className="einvoicing-profile" onSubmit={event=>{event.preventDefault();if(dirty&&!locked)save.mutate()}}>
   <header className="einvoicing-profile-head"><div><h2 className="modal-section-title">{profile.countryName} · {profile.currency}</h2></div><Status tone={profile.enabled?"green":"gray"}>{profile.enabled?"Emitiendo":"Sin emitir"}</Status></header>
   {!profile.supported?<p className="pos-tip-help">La facturación electrónica aún no está disponible para este país.</p>:<>
    <label className="switch-row compact"><input type="checkbox" checked={draft.enabled} disabled={locked} onChange={event=>setDraft({...draft,enabled:event.target.checked})}/><span/><b>Emitir comprobantes electrónicos</b></label>
    <div className="einvoicing-profile-fields">
     <FormField label="Proveedor"><Select disabled={locked} value={draft.provider} onChange={event=>setDraft({...draft,provider:event.target.value})}>{settings.providerOptions.map(option=><option key={option.value} value={option.value}>{option.label}</option>)}</Select></FormField>
     <FormField label="Ambiente"><Select disabled={locked} value={draft.environment} onChange={event=>setDraft({...draft,environment:event.target.value as ProfileSettingsInput["environment"]})}>{Object.entries(settings.environmentLabels).map(([value,label])=><option key={value} value={value}>{label}</option>)}</Select></FormField>
    </div>
    {profile.issuerIssue&&<InlineNotice tone="warning" title="Completa los datos del emisor" action={<Link className="button secondary" href={pageRoutes.organizationSettings}>Ir a Mi restaurante</Link>}>{profile.issuerIssue}</InlineNotice>}
    <CredentialsSummary profile={profile} pendingProvider={draft.provider!==profile.provider} canManage={canManage} onEdit={onCredentials}/>
    {canManage&&dirty&&<div className="einvoicing-profile-actions"><Button kind="ghost" disabled={save.isPending} onClick={()=>setDraft({enabled:profile.enabled,provider:profile.provider,environment:profile.environment})}>Cancelar</Button><Button type="submit" icon="check" disabled={save.isPending} loading={save.isPending}>{save.isPending?"Guardando…":"Guardar"}</Button></div>}
   </>}
  </form>
 </section>;
}

function LocationSeriesDialog({location,documentTypes,close}:{location:LocationSetting;documentTypes:LocationSetting["series"];close:()=>void}){
 const{notify}=useFeedback();const client=useQueryClient();
 const[series,setSeries]=useState<Record<string,string>>(Object.fromEntries(documentTypes.map(item=>[item.documentType,item.series||item.suggested])));
 const[defaultDocument,setDefaultDocument]=useState<FiscalDocumentType>(location.defaultDocument);
 const save=useMutation({mutationFn:()=>saveLocationSettings(location.locationId,{defaultDocument,series}),onSuccess:()=>{void client.invalidateQueries({queryKey:["einvoicing-settings"]});void client.invalidateQueries({queryKey:["einvoicing-checkout"]});close();notify({tone:"success",title:"Series guardadas",message:`${location.name} ya tiene sus series de comprobantes.`})},onError:error=>notify({tone:"danger",title:"No se pudo guardar",message:error.message})});
 const busy=save.isPending;
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={()=>{if(!busy)close()}} className="crud-modal compact modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="einvoicing-location-title">
  <ModalHeader title={location.name} titleId="einvoicing-location-title" icon="receipt" subtitle="Series de comprobantes" onClose={close} busy={busy} closeLabel="Cerrar"/>
  <ModalForm onSubmit={event=>{event.preventDefault();if(!busy)save.mutate()}} noValidate inert={busy}>
   <ModalBody><div className="form-grid">
    {documentTypes.map(item=><FormField key={item.documentType} icon={false} label={`Serie de ${item.label.toLowerCase()}`} help={item.lastNumber>0?`Ya emitió ${item.lastNumber} comprobantes; no se puede cambiar.`:undefined}><Input readOnly={item.lastNumber>0} maxLength={8} value={series[item.documentType]??""} onChange={event=>setSeries({...series,[item.documentType]:event.target.value.toUpperCase()})} placeholder={item.suggested}/></FormField>)}
    <FormField className="span-2" label="Comprobante sugerido al cobrar"><Select value={defaultDocument} onChange={event=>setDefaultDocument(event.target.value as FiscalDocumentType)}><option value="none">Sin comprobante</option>{documentTypes.map(item=><option key={item.documentType} value={item.documentType}>{item.label}</option>)}</Select></FormField>
   </div></ModalBody>
   <ModalFooter><Button action="cancel" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" icon="check" disabled={busy} loading={busy}>{busy?"Guardando…":"Guardar"}</Button></ModalFooter>
  </ModalForm>
 </Dialog></div>;
}

const credentialFacts:{key:string;label:string;date?:boolean}[]=[{key:"solUser",label:"Usuario"},{key:"certificateSubject",label:"Titular del certificado"},{key:"certificateExpiresAt",label:"Vence",date:true}];

/** Credentials are write-only: the page shows whether they exist and their non-secret metadata. */
function CredentialsSummary({profile,pendingProvider,canManage,onEdit}:{profile:ProfileSetting;pendingProvider:boolean;canManage:boolean;onEdit:()=>void}){
 const{location}=useSession();
 const status=profile.credentials;
 if(!status.fields?.length)return null;
 if(pendingProvider)return <p className="pos-tip-help">Guarda el proveedor para configurar sus credenciales.</p>;
 const metadata=status.metadata??{};
 const facts=credentialFacts.filter(fact=>metadata[fact.key]);
 return <div className="einvoicing-credentials">
  <div className="einvoicing-credentials-head">
   <div><b>Credenciales</b><small>{status.configured?"Guardadas cifradas; no se vuelven a mostrar.":status.required?"Obligatorias para emitir en producción.":"En pruebas se usa el usuario público de SUNAT."}</small></div>
   <Status tone={status.configured?"green":status.required?"orange":"gray"}>{status.configured?"Configuradas":status.required?"Pendientes":"Opcionales"}</Status>
  </div>
  {facts.length>0&&<dl className="einvoicing-credentials-facts">{facts.map(fact=><div key={fact.key}><dt>{fact.label}</dt><dd>{fact.date?formatRegionalCalendarDate(metadata[fact.key].slice(0,10),location?.country,{dateStyle:"medium"}):metadata[fact.key]}</dd></div>)}</dl>}
  {canManage&&<div><Button kind="secondary" icon="key" onClick={onEdit}>{status.configured?"Actualizar credenciales":"Configurar credenciales"}</Button></div>}
 </div>;
}

function readBase64(file:File){
 return new Promise<string>((resolve,reject)=>{const reader=new FileReader();reader.onload=()=>resolve(String(reader.result).split(",")[1]??"");reader.onerror=()=>reject(new Error("No pudimos leer el archivo."));reader.readAsDataURL(file)});
}

function CredentialsDialog({profile,providerLabel,close}:{profile:ProfileSetting;providerLabel:string;close:()=>void}){
 const{notify}=useFeedback();const client=useQueryClient();
 const fields=profile.credentials.fields??[];const configured=profile.credentials.configured;
 const[values,setValues]=useState<Record<string,string>>({});
 const[fileNames,setFileNames]=useState<Record<string,string>>({});
 const[error,setError]=useState("");
 const save=useMutation({mutationFn:()=>saveCredentials(profile.fiscalProfileId,{provider:profile.provider,values}),onSuccess:()=>{void client.invalidateQueries({queryKey:["einvoicing-settings"]});close();notify({tone:"success",title:"Credenciales guardadas",message:`${providerLabel} quedó configurado para ${profile.countryName}.`})},onError:failure=>setError(failure.message)});
 const busy=save.isPending;
 const missing=configured?[]:fields.filter(field=>field.required&&!values[field.key]);
 async function pick(field:CredentialField,file:File|undefined){
  if(!file)return;
  try{const content=await readBase64(file);setValues(current=>({...current,[field.key]:content}));setFileNames(current=>({...current,[field.key]:file.name}));setError("")}
  catch(failure){setError(failure instanceof Error?failure.message:"No pudimos leer el archivo.")}
 }
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={()=>{if(!busy)close()}} className="crud-modal compact modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="einvoicing-credentials-title">
  <ModalHeader title="Credenciales" titleId="einvoicing-credentials-title" icon="key" subtitle={`${providerLabel} · ${profile.countryName}`} onClose={close} busy={busy} closeLabel="Cerrar"/>
  <ModalForm onSubmit={event=>{event.preventDefault();if(busy)return;if(missing.length){setError(`Completa: ${missing.map(field=>field.label).join(", ")}.`);return}save.mutate()}} noValidate inert={busy}>
   <ModalBody>
    <p className="pos-tip-help einvoicing-credentials-intro">Se guardan cifradas y nunca se vuelven a mostrar.{configured?" Deja un campo vacío para conservar el valor guardado.":""}</p>
    <div className="form-grid">
     {fields.map(field=>field.type==="file"?
      <FormField key={field.key} as="div" className="span-2" icon={false} label={field.label} optional={configured||!field.required}><label className="einvoicing-file"><input type="file" accept={field.accept} onChange={event=>{void pick(field,event.target.files?.[0]);event.target.value=""}}/><span className="button secondary"><Icon name="upload" size={16}/>{fileNames[field.key]||configured?"Cambiar archivo":"Elegir archivo"}</span><small>{fileNames[field.key]??(configured?"Guardado":"Ningún archivo elegido")}</small></label></FormField>:
      <FormField key={field.key} icon={false} label={field.label} optional={configured||!field.required}><Input type={field.type==="secret"?"password":"text"} autoComplete={field.type==="secret"?"new-password":"off"} value={values[field.key]??""} placeholder={configured?"Sin cambios":undefined} onChange={event=>{setValues({...values,[field.key]:event.target.value});setError("")}}/></FormField>)}
    </div>
    {error&&<p className="field-error" role="alert">{error}</p>}
   </ModalBody>
   <ModalFooter><Button action="cancel" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" icon="check" disabled={busy} loading={busy}>{busy?"Guardando…":"Guardar"}</Button></ModalFooter>
  </ModalForm>
 </Dialog></div>;
}
