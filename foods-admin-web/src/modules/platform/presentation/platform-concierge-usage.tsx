"use client";
import "./platform-concierge-usage.css";
import "./platform-skeletons.css";
import {useState,type ReactNode} from "react";
import {useMutation,useQuery,useQueryClient,type UseQueryResult} from "@tanstack/react-query";
import {Button,EmptyState,ErrorState,FormField,InlineNotice,Input,Skeleton,Status} from "@/design-system";
import {DataTable} from "@/design-system/data-table";
import {Icon} from "@/design-system/icons";
import {useFeedback,useSession} from "@/providers";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {conciergeAlertLabel,conciergeAlertStatus,conciergeAlertTone,conciergeLimitError,conciergeLimitOptions,conciergeLimitPayload,conciergeLimitSourceLabel,conciergeModeLabel,conciergeModeOptions,conciergeMonthLabel,conciergeSettingsError,conciergeSettingsPayload,conciergeTierLabel,conciergeUsageLabel,formatCount} from "../domain/concierge-usage";
import type {ConciergeLimitDraft,ConciergeUsageDetail,ConciergeUsagePage,ConciergeUsageSettings,ConciergeUsageSettingsDraft,ConciergeUsageSummary} from "../domain/types";
import {getConciergeUsageSettings,getOrganizationConciergeUsage,updateConciergeUsageSettings,updateOrganizationConciergeLimit} from "../infrastructure/platform-api";

/*
 * Fudia Concierge — tope mensual de conversaciones.
 * Todo este archivo es exclusivo del administrador de plataforma: las empresas
 * nunca ven su tope ni su consumo.
 */

const settingsKey=["platform-concierge-settings"] as const;
const detailKey=(organizationId:string)=>["platform-concierge-usage-detail",organizationId] as const;

function ConciergeChoice({name,value,checked,disabled,label,description,onChange}:{name:string;value:string;checked:boolean;disabled?:boolean;label:string;description:string;onChange:()=>void}){
 return <label className="concierge-choice"><input type="radio" name={name} value={value} checked={checked} disabled={disabled} onChange={onChange}/><span aria-hidden="true"/><div><b>{label}</b><small>{description}</small></div></label>;
}

/** País de la sesión para números y meses (`regional-format`), nunca un locale fijo. */
function useCountry(){return useSession().location?.country;}

function PanelHeader({title,id,aside}:{title:string;id:string;aside?:ReactNode}){
 return <header><div><small>Fudia Concierge</small><h2 id={id}>{title}</h2></div>{aside}</header>;
}

/* ───────── Configuración global: modo y porcentaje de aviso ───────── */

export function ConciergeUsageSettingsPanel(){
 const settings=useQuery({queryKey:settingsKey,queryFn:getConciergeUsageSettings});
 return <section className="panel concierge-panel concierge-settings" aria-labelledby="concierge-settings-title" aria-busy={settings.isLoading}>
  <PanelHeader id="concierge-settings-title" title="Tope mensual de conversaciones"/>
  {settings.isLoading?<ConciergeSettingsSkeleton/>
   :settings.isError||!settings.data?<ErrorState title="No pudimos cargar la configuración de Concierge" text={settings.error?.message} retrying={settings.isFetching} onRetry={()=>void settings.refetch()}/>
   :<ConciergeSettingsForm key={settings.data.updatedAt} value={settings.data}/>}
 </section>;
}

function ConciergeSettingsForm({value}:{value:ConciergeUsageSettings}){
 const{notify}=useFeedback();
 const{location}=useSession();
 const client=useQueryClient();
 const[draft,setDraft]=useState<ConciergeUsageSettingsDraft>({mode:value.mode,warnPercent:String(value.warnPercent)});
 const[error,setError]=useState("");
 const save=useMutation({
  mutationFn:updateConciergeUsageSettings,
  onSuccess:data=>{
   client.setQueryData(settingsKey,data);
   notify({tone:"success",title:"Configuración guardada",message:`Fudia Concierge quedó en «${conciergeModeLabel(data.mode)}» con aviso al ${data.warnPercent} %.`});
   void client.invalidateQueries({queryKey:["platform-concierge-usage"]});
   void client.invalidateQueries({queryKey:["platform-concierge-usage-detail"]});
  },
  onError:e=>notify({tone:"danger",title:"No se pudo guardar",message:e.message}),
 });
 const busy=save.isPending;
 const changed=draft.mode!==value.mode||draft.warnPercent.trim()!==String(value.warnPercent);
 const submit=()=>{
  const message=conciergeSettingsError(draft);
  setError(message);
  if(!message)save.mutate(conciergeSettingsPayload(draft));
 };
 const updated=formatRegionalDateTime(value.updatedAt,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium",timeStyle:"short"});
 return <form onSubmit={event=>{event.preventDefault();submit()}} noValidate>
  <div className="concierge-panel-body">
   <fieldset className="concierge-choice-group" disabled={busy}>
    <legend>Modo de aplicación</legend>
    {conciergeModeOptions.map(option=><ConciergeChoice key={option.value} name="concierge-mode" value={option.value} checked={draft.mode===option.value} label={option.label} description={option.description} onChange={()=>setDraft({...draft,mode:option.value})}/>)}
   </fieldset>
   <FormField label="Porcentaje de aviso" help="Desde este porcentaje del tope, la empresa figura «Cerca del tope»." error={error}><Input type="number" inputMode="numeric" min="1" max="100" step="1" value={draft.warnPercent} disabled={busy} onChange={event=>setDraft({...draft,warnPercent:event.target.value})}/></FormField>
   <InlineNotice tone="info" title="Las conversaciones en curso nunca se cortan">En «Bloquear conversaciones nuevas», al llegar al 100 % del tope ya no se abren conversaciones nuevas para esa empresa.</InlineNotice>
  </div>
  <footer className="concierge-panel-footer"><small>Actualizado el {updated}</small><Button type="submit" icon="check" disabled={busy||!changed} loading={busy}>{busy?"Guardando…":"Guardar"}</Button></footer>
 </form>;
}

function ConciergeSettingsSkeleton(){
 return <div className="concierge-panel-body" role="status" aria-label="Cargando configuración de Concierge" aria-busy="true">
  <i className="platform-skeleton-block concierge-skeleton-label"/>
  {Array.from({length:3},(_,index)=><i className="platform-skeleton-block concierge-skeleton-choice" key={index}/>)}
  <i className="platform-skeleton-block concierge-skeleton-label"/>
  <i className="platform-skeleton-block concierge-skeleton-control"/>
 </div>;
}

/* ───────── Empresas y planes: uso del mes por empresa ───────── */

/** Celda del listado: skeleton propio, error explícito y «0 / N» cuando no hay consumo. */
export function ConciergeUsageCell({organizationId,usage}:{organizationId:string;usage:UseQueryResult<ConciergeUsagePage>}){
 const country=useCountry();
 const item=usage.data?.items.find(entry=>entry.organizationId===organizationId);
 if(!item&&usage.isError)return <span className="concierge-usage-unavailable"><Icon name="alert" size={14}/>No disponible</span>;
 if(!item&&(usage.isPending||usage.isFetching))return <span className="concierge-usage-skeleton" role="status" aria-label="Cargando uso de Concierge"><Skeleton className="ds-skeleton-line"/><Skeleton className="ds-skeleton-line short"/></span>;
 if(!item)return <span className="platform-company-muted">Sin registro</span>;
 return <div className="platform-company-cell concierge-usage-cell">
  <b>{conciergeUsageLabel(item,country)}</b>
  <small>{item.usagePercent===null?"Sin tope":`${item.usagePercent} % del tope`}</small>
  {item.alert!=="none"&&<Status tone={conciergeAlertTone(item.alert)}>{conciergeAlertLabel(item.alert)}</Status>}
 </div>;
}

/** Aviso del administrador de plataforma: resume toda la cartera, no solo la página visible. */
export function ConciergeUsageNotice({usage}:{usage:UseQueryResult<ConciergeUsagePage>}){
 if(usage.isError&&!usage.data)return <InlineNotice tone="danger" className="concierge-usage-notice" title="No pudimos cargar el uso de Fudia Concierge" action={<Button kind="secondary" icon="refresh" onClick={()=>void usage.refetch()} disabled={usage.isFetching} loading={usage.isFetching}>{usage.isFetching?"Reintentando…":"Reintentar"}</Button>}>{usage.error?.message}</InlineNotice>;
 const summary=usage.data?.summary;
 if(!summary||summary.warning+summary.reached===0)return null;
 const parts=[
  summary.warning?`${summary.warning} ${summary.warning===1?"empresa cerca del tope":"empresas cerca del tope"}`:"",
  summary.reached?`${summary.reached} ${summary.reached===1?"empresa alcanzó su tope":"empresas alcanzaron su tope"}`:"",
 ].filter(Boolean);
 const enforce=usage.data?.settings.mode==="enforce";
 return <InlineNotice tone="warning" className="concierge-usage-notice" title={`Fudia Concierge: ${parts.join(" y ")} este mes`}>{enforce?"Al llegar al tope no se abren conversaciones nuevas; las conversaciones en curso continúan.":"El modo actual solo avisa: ninguna conversación se bloquea."}</InlineNotice>;
}

/* ───────── Detalle de empresa: ficha de solo lectura ───────── */

export function ConciergeUsageDialogSection({organizationId}:{organizationId:string}){
 const country=useCountry();
 const detail=useQuery({queryKey:detailKey(organizationId),queryFn:()=>getOrganizationConciergeUsage(organizationId)});
 const data=detail.data;
 return <section className="company-detail-section" aria-busy={detail.isLoading}>
  <header><small>FUDIA CONCIERGE</small><h3>Conversaciones del mes</h3></header>
  {detail.isLoading?<div className="company-detail-fields" role="status" aria-label="Cargando uso de Concierge">{Array.from({length:4},(_,index)=><div key={index}><i className="platform-skeleton-block company-detail-skeleton-label"/><i className="platform-skeleton-block company-detail-skeleton-value"/></div>)}</div>
   :detail.isError||!data?<ErrorState className="concierge-dialog-state" title="No pudimos cargar el uso de Concierge" text={detail.error?.message} retrying={detail.isFetching} onRetry={()=>void detail.refetch()}/>
   :<dl className="company-detail-fields">
    <div><dt>Uso de {conciergeMonthLabel(data.month,country)}</dt><dd>{conciergeUsageLabel(data,country)}{data.usagePercent!==null?` · ${data.usagePercent} %`:""}</dd></div>
    <div><dt>Situación</dt><dd><AlertStatus usage={data} mode={data.settings.mode}/></dd></div>
    <div><dt>Origen del tope</dt><dd>{conciergeLimitSourceLabel(data,country)}</dd></div>
    <div><dt>Conversaciones rechazadas</dt><dd>{data.rejectedConversations>0?formatCount(data.rejectedConversations,country):"Ninguna"}</dd></div>
   </dl>}
 </section>;
}

function AlertStatus({usage,mode}:{usage:ConciergeUsageSummary;mode:ConciergeUsageSettings["mode"]}){
 const status=conciergeAlertStatus(usage,mode);
 return <Status tone={status.tone}>{status.label}</Status>;
}

/* ───────── Detalle de empresa: uso, tokens, historial y excepción de tope ───────── */

export function ConciergeUsagePanel({organizationId}:{organizationId:string}){
 const country=useCountry();
 const detail=useQuery({queryKey:detailKey(organizationId),queryFn:()=>getOrganizationConciergeUsage(organizationId)});
 const data=detail.data;
 if(detail.isLoading)return <ConciergeUsagePanelSkeleton/>;
 if(detail.isError||!data)return <section className="panel concierge-panel concierge-usage-section" aria-labelledby="concierge-usage-title"><PanelHeader id="concierge-usage-title" title="Conversaciones del mes"/><ErrorState title="No pudimos cargar el uso de Concierge" text={detail.error?.message} retrying={detail.isFetching} onRetry={()=>void detail.refetch()}/></section>;
 return <div className="concierge-usage-section">
  <div className="concierge-usage-grid">
   <section className="panel concierge-panel" aria-labelledby="concierge-usage-title">
    <PanelHeader id="concierge-usage-title" title="Conversaciones del mes" aside={<AlertStatus usage={data} mode={data.settings.mode}/>}/>
    <div className="concierge-panel-body">
     <dl className="concierge-usage-metrics">
      <div><dt>Uso de {conciergeMonthLabel(data.month,country)}</dt><dd>{conciergeUsageLabel(data,country)}</dd></div>
      <div><dt>Uso del tope</dt><dd>{data.usagePercent===null?"Sin tope":`${data.usagePercent} %`}</dd></div>
      <div><dt>Origen del tope</dt><dd>{conciergeLimitSourceLabel(data,country)}</dd></div>
      {data.rejectedConversations>0&&<div><dt>Conversaciones rechazadas</dt><dd>{formatCount(data.rejectedConversations,country)}</dd></div>}
     </dl>
     {data.settings.mode==="measure"&&<InlineNotice tone="info">El modo global es «Solo medir»: se cuentan las conversaciones, sin avisos ni bloqueos.</InlineNotice>}
     <div className="concierge-tokens">
      <h3>Tokens del mes por nivel</h3>
      {data.tokens.length?<dl>{data.tokens.map(token=><div key={token.tier}><dt>{conciergeTierLabel(token.tier)}</dt><dd><span>Entrada <b>{formatCount(token.inputTokens,country)}</b></span><span>Salida <b>{formatCount(token.outputTokens,country)}</b></span></dd></div>)}</dl>
       :<p className="concierge-muted">Sin consumo de tokens este mes.</p>}
     </div>
    </div>
   </section>
   <ConciergeLimitEditor key={`${data.limitSource}:${data.maxConversations??""}`} organizationId={organizationId} detail={data}/>
  </div>
  <section className="panel concierge-panel concierge-history" aria-labelledby="concierge-history-title">
   <PanelHeader id="concierge-history-title" title="Historial de los últimos 6 meses"/>
   {data.history.length?<DataTable label="Historial mensual de Fudia Concierge"><thead><tr><th>MES</th><th>CONVERSACIONES</th><th>RECHAZADAS</th><th>TOKENS DE ENTRADA</th><th>TOKENS DE SALIDA</th></tr></thead><tbody>{data.history.map(month=><tr key={month.month}><td><b>{conciergeMonthLabel(month.month,country)}</b></td><td>{formatCount(month.conversations,country)}</td><td>{formatCount(month.rejectedConversations,country)}</td><td>{formatCount(month.inputTokens,country)}</td><td>{formatCount(month.outputTokens,country)}</td></tr>)}</tbody></DataTable>
    :<EmptyState icon="chat" title="Aún no hay historial" text="El consumo mensual aparecerá aquí cuando la empresa use Fudia Concierge."/>}
  </section>
 </div>;
}

function ConciergeLimitEditor({organizationId,detail}:{organizationId:string;detail:ConciergeUsageDetail}){
 const country=useCountry();
 const{notify}=useFeedback();
 const client=useQueryClient();
 const[draft,setDraft]=useState<ConciergeLimitDraft>({limitSource:detail.limitSource,maxConversations:detail.limitSource==="custom"&&detail.maxConversations!==null?String(detail.maxConversations):""});
 const[error,setError]=useState("");
 const save=useMutation({
  mutationFn:(value:ConciergeLimitDraft)=>updateOrganizationConciergeLimit(organizationId,conciergeLimitPayload(value)),
  onSuccess:data=>{
   client.setQueryData(detailKey(organizationId),data);
   notify({tone:"success",title:"Tope actualizado",message:`${data.organizationName}: ${data.maxConversations===null?"sin tope mensual":`${formatCount(data.maxConversations,country)} conversaciones al mes`}. Aplica desde la siguiente conversación.`});
   void client.invalidateQueries({queryKey:["platform-concierge-usage"]});
  },
  onError:e=>notify({tone:"danger",title:"No se pudo guardar",message:e.message}),
 });
 const busy=save.isPending;
 const changed=draft.limitSource!==detail.limitSource||(draft.limitSource==="custom"&&draft.maxConversations.trim()!==String(detail.maxConversations??""));
 const planLimit=detail.planMaxConversations===null?"sin tope mensual":`${formatCount(detail.planMaxConversations,country)} conversaciones al mes`;
 const descriptions:Record<ConciergeLimitDraft["limitSource"],string>={
  plan:`${detail.planName?`Plan ${detail.planName}`:"El plan contratado"}: ${planLimit}.`,
  custom:"Un tope solo para esta empresa, distinto al de su plan.",
  unlimited:"Sin tope mensual para esta empresa.",
 };
 const submit=()=>{
  const message=conciergeLimitError(draft);
  setError(message);
  if(!message)save.mutate(draft);
 };
 return <section className="panel concierge-panel" aria-labelledby="concierge-limit-title">
  <PanelHeader id="concierge-limit-title" title="Tope de la empresa"/>
  <form onSubmit={event=>{event.preventDefault();submit()}} noValidate>
   <div className="concierge-panel-body">
    <fieldset className="concierge-choice-group" disabled={busy}>
     <legend>Tope mensual de conversaciones</legend>
     {conciergeLimitOptions.map(option=><ConciergeChoice key={option.value} name="concierge-limit" value={option.value} checked={draft.limitSource===option.value} label={option.label} description={descriptions[option.value]} onChange={()=>{setError("");setDraft({...draft,limitSource:option.value})}}/>)}
    </fieldset>
    {draft.limitSource==="custom"&&<FormField label="Conversaciones al mes" error={error}><Input type="number" inputMode="numeric" min="1" step="1" value={draft.maxConversations} disabled={busy} onChange={event=>setDraft({...draft,maxConversations:event.target.value})} placeholder="Ej. 2000"/></FormField>}
   </div>
   <footer className="concierge-panel-footer"><small>Aplica desde la siguiente conversación.</small><Button type="submit" icon="check" disabled={busy||!changed} loading={busy}>{busy?"Guardando…":"Guardar tope"}</Button></footer>
  </form>
 </section>;
}

function ConciergeUsagePanelSkeleton(){
 return <div className="concierge-usage-section" role="status" aria-label="Cargando uso de Concierge" aria-busy="true">
  <div className="concierge-usage-grid">
   {Array.from({length:2},(_,panel)=><section className="panel concierge-panel" key={panel}>
    <header><div className="concierge-skeleton-heading"><i className="platform-skeleton-block concierge-skeleton-label"/><i className="platform-skeleton-block concierge-skeleton-title"/></div></header>
    <div className="concierge-panel-body">{Array.from({length:3},(_,index)=><i className="platform-skeleton-block concierge-skeleton-choice" key={index}/>)}</div>
   </section>)}
  </div>
  <section className="panel concierge-panel concierge-history">
   <header><div className="concierge-skeleton-heading"><i className="platform-skeleton-block concierge-skeleton-label"/><i className="platform-skeleton-block concierge-skeleton-title"/></div></header>
   <div className="concierge-panel-body">{Array.from({length:4},(_,index)=><i className="platform-skeleton-block concierge-skeleton-row" key={index}/>)}</div>
  </section>
 </div>;
}
