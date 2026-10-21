"use client";
import "./platform-companies.css";
import "./platform-skeletons.css";
import {useState} from "react";
import Link from "next/link";
import {useQuery} from "@tanstack/react-query";
import {Button,PageHeader,Pagination,RowActionButton} from "@/design-system";
import {Icon,type IconName} from "@/design-system/icons";
import {useSession} from "@/providers";
import {useDebouncedValue} from "@/shared/hooks/use-debounced-value";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {companySettingsPath} from "@/shared/routing/page-routes";
import {listPlatformOrganizations} from "../infrastructure/platform-api";
import {CompanyDetailDialog} from "./platform-company-detail-dialog";
import type {PaymentStanding,PlatformOrganizationPage} from "../domain/types";

const DAY=86_400_000;
const summaryCards:Array<{key:"total"|PaymentStanding;label:string;icon:IconName}>=[
 {key:"total",label:"Empresas",icon:"building"},
 {key:"up_to_date",label:"Al día",icon:"circleCheck"},
 {key:"due_soon",label:"Por vencer",icon:"calendar"},
 {key:"overdue",label:"Vencidas",icon:"alert"},
 {key:"trial",label:"En prueba",icon:"contract"},
];

export function PlatformCompaniesPage(){
 const{user}=useSession();
 if(!user?.platformAdmin)return <RestrictedCompanies/>;
 return <CompaniesDirectory/>;
}

export function RestrictedCompanies(){
 return <>
  <PageHeader eyebrow="CONFIGURACIÓN" title="Empresas y planes" description="Plan contratado y situación de pago de cada empresa cliente."/>
  <div className="panel management"><div className="catalog-state"><span><Icon name="lock" size={24}/></span><b>Acceso restringido</b><p>Solo el administrador de plataforma puede consultar las empresas y sus suscripciones.</p></div></div>
 </>;
}

function CompaniesDirectory(){
 const[q,setQ]=useState("");
 const debouncedQ=useDebouncedValue(q);
 const[planId,setPlanId]=useState("");
 const[standing,setStanding]=useState("");
 const[page,setPage]=useState(1);
 const[pageSize,setPageSize]=useState(10);
 const list=useQuery({
  queryKey:["platform-organizations",debouncedQ,planId,standing,page,pageSize],
  queryFn:()=>listPlatformOrganizations({q:debouncedQ,planId,standing,page,pageSize}),
  placeholderData:previous=>previous,
 });
 const data=list.data;
 const filtered=Boolean(debouncedQ.trim()||planId||standing);
 const pickStanding=(value:string)=>{setStanding(value);setPage(1)};

 return <>
  <PageHeader eyebrow="CONFIGURACIÓN" title="Empresas y planes" description="Plan contratado, vencimientos y situación de pago de cada empresa cliente."/>
  <CompaniesSummary data={data} loading={list.isLoading} active={standing} onPick={pickStanding}/>
  <section className="panel management standardized-management platform-companies-panel">
   <div className="toolbar">
    <label><Icon name="search" size={18}/><input value={q} onChange={event=>{setQ(event.target.value);setPage(1)}} placeholder="Buscar por nombre, razón social o RUC..." aria-label="Buscar empresas"/></label>
    <select aria-label="Filtrar por plan" value={planId} onChange={event=>{setPlanId(event.target.value);setPage(1)}} disabled={!data}>
     <option value="">Todos los planes</option>
     {data?.planOptions.map(option=><option key={option.value} value={option.value}>{option.label}</option>)}
    </select>
    <select aria-label="Filtrar por situación de pago" value={standing} onChange={event=>pickStanding(event.target.value)} disabled={!data}>
     <option value="">Toda situación de pago</option>
     {data?.standingOptions.map(option=><option key={option.value} value={option.value}>{option.label}</option>)}
    </select>
   </div>
   {list.isLoading?<CompaniesTableSkeleton/>
    :list.isError?<div className="catalog-state error"><span><Icon name="alert"/></span><b>No pudimos cargar las empresas</b><p>{list.error.message}</p><Button kind="ghost" onClick={()=>void list.refetch()}>Reintentar</Button></div>
    :!data?.items.length?<div className="catalog-state"><span><Icon name="building"/></span><b>{filtered?"Sin coincidencias":"Aún no hay empresas"}</b><p>{filtered?"Ninguna empresa coincide con la búsqueda o los filtros aplicados.":"Las empresas registradas desde «Registrar empresa» aparecerán aquí con su plan."}</p></div>
    :<CompaniesTable data={data} busy={list.isFetching}/>}
   {data&&data.total>0&&<Pagination page={page} size={pageSize} total={data.total} onPage={setPage} onSize={value=>{setPageSize(value);setPage(1)}}/>}
  </section>
 </>;
}

function CompaniesSummary({data,loading,active,onPick}:{data?:PlatformOrganizationPage;loading:boolean;active:string;onPick:(value:string)=>void}){
 if(loading||!data)return <div className="platform-companies-summary" aria-label="Cargando resumen" aria-busy="true">{summaryCards.map(card=><div className="platform-companies-kpi" key={card.key}><i className="platform-skeleton-block kpi-icon"/><div><i className="platform-skeleton-block kpi-value"/><i className="platform-skeleton-block kpi-label"/></div></div>)}</div>;
 return <div className="platform-companies-summary">
  {summaryCards.map(card=>{
   const value=card.key==="total"?"":card.key;
   const selected=active===value;
   return <button type="button" key={card.key} className={"platform-companies-kpi "+card.key+(selected?" selected":"")} aria-pressed={selected} onClick={()=>onPick(selected&&value?"":value)}>
    <span className="kpi-icon"><Icon name={card.icon} size={18}/></span>
    <span><b>{data.summary[card.key]??0}</b><small>{card.label}</small></span>
   </button>;
  })}
 </div>;
}

function CompaniesTable({data,busy}:{data:PlatformOrganizationPage;busy:boolean}){
 const{location}=useSession();
 const[now]=useState(()=>Date.now());
 const[detailId,setDetailId]=useState<string|null>(null);
 const standingLabel=(value:PaymentStanding)=>data.standingOptions.find(option=>option.value===value)?.label??value;
 const date=(value:string)=>formatRegionalDateTime(value,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium"});

 return <><div className="table-wrap hover-scroll" aria-busy={busy}>
  <table className="platform-companies-table">
   <thead><tr><th>EMPRESA</th><th>PLAN</th><th>PRECIO</th><th>PRÓXIMO VENCIMIENTO</th><th>ÚLTIMO PAGO</th><th>SITUACIÓN</th><th>ACCIONES</th></tr></thead>
   <tbody>{data.items.map(company=>{
    const subscription=company.subscription;
    const due=subscription?.status==="trial"?subscription.trialEndsAt??subscription.renewsAt:subscription?.renewsAt??null;
    return <tr key={company.id} className={company.active?"":"inactive"}>
     <td><div className="platform-company-name"><span className="row-icon"><Icon name="building" size={17}/></span><div><Link href={companySettingsPath(company.id)}>{company.tradeName}</Link><small>{[company.taxId?"RUC "+company.taxId:"",company.active?"":"Inactiva"].filter(Boolean).join(" · ")||"Sin RUC"}</small></div></div></td>
     <td>{subscription?<div className="platform-company-cell"><b>{subscription.plan.name}</b><small>{subscription.billingCycle==="annual"?"Anual":"Mensual"}{subscription.autoRenew?" · Renovación automática":""}</small></div>:<span className="platform-company-muted">Sin plan</span>}</td>
     <td>{subscription?<b className="platform-company-amount">{subscription.currency} {Number(subscription.priceAmount).toFixed(2)}</b>:"—"}</td>
     <td>{due&&subscription?.status!=="cancelled"?<div className="platform-company-cell"><b>{date(due)}</b><small className={company.paymentStanding}>{relativeDays(due,now,subscription?.status==="trial")}</small></div>:<span className="platform-company-muted">—</span>}</td>
     <td>{company.lastPayment?<div className="platform-company-cell"><b className="platform-company-amount">{company.lastPayment.currency} {Number(company.lastPayment.amount).toFixed(2)}</b><small>{company.lastPayment.paidAt?date(company.lastPayment.paidAt):"Sin fecha"}</small></div>:<span className="platform-company-muted">Sin pagos</span>}</td>
     <td><span className={"platform-standing "+company.paymentStanding}><i aria-hidden="true"/>{standingLabel(company.paymentStanding)}</span></td>
     <td><div className="platform-company-actions"><RowActionButton action="view" label="Ver detalle" onClick={()=>setDetailId(company.id)}/><Link href={companySettingsPath(company.id)} className="ds-icon-button" aria-label={"Gestionar suscripción de "+company.tradeName} data-tooltip="Gestionar suscripción"><Icon name="receipt" size={18}/></Link></div></td>
    </tr>;
   })}</tbody>
  </table>
 </div>
 {detailId&&<CompanyDetailDialog organizationId={detailId} standingOptions={data.standingOptions} close={()=>setDetailId(null)}/>}
 </>;
}

function relativeDays(value:string,now:number,trial:boolean){
 const days=Math.ceil((new Date(value).getTime()-now)/DAY);
 if(days<0)return `Venció hace ${-days} ${-days===1?"día":"días"}`;
 if(days===0)return trial?"La prueba termina hoy":"Vence hoy";
 return `${trial?"La prueba termina":"Vence"} en ${days} ${days===1?"día":"días"}`;
}

function CompaniesTableSkeleton(){
 return <div className="table-wrap" aria-label="Cargando empresas" aria-busy="true">
  <table className="platform-companies-table"><thead><tr><th>EMPRESA</th><th>PLAN</th><th>PRECIO</th><th>PRÓXIMO VENCIMIENTO</th><th>ÚLTIMO PAGO</th><th>SITUACIÓN</th><th>ACCIONES</th></tr></thead>
   <tbody>{Array.from({length:5},(_,row)=><tr key={row}>{Array.from({length:7},(_,cell)=><td key={cell}><i className="platform-skeleton-block table-line"/></td>)}</tr>)}</tbody>
  </table>
 </div>;
}
