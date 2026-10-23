"use client";
import {DataTable} from "@/design-system/data-table";
import "./platform-companies.css";
import "./platform-skeletons.css";
import {useMemo,useState} from "react";
import Link from "next/link";
import {useQuery,type UseQueryResult} from "@tanstack/react-query";
import {EmptyState,ErrorState,PageHeader,Pagination,RowActionButton,SearchField,Select,Status,TableSkeleton} from "@/design-system";
import {Icon,type IconName} from "@/design-system/icons";
import {useSession} from "@/providers";
import {useDebouncedValue} from "@/shared/hooks/use-debounced-value";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {companySettingsPath} from "@/shared/routing/page-routes";
import {listConciergeUsage,listPlatformOrganizations} from "../infrastructure/platform-api";
import {ConciergeUsageCell,ConciergeUsageNotice} from "./platform-concierge-usage";
import {CompanyDetailDialog} from "./platform-company-detail-dialog";
import {paymentStandingTone} from "../domain/subscription-status";
import type {ConciergeUsagePage,PaymentStanding,PlatformOrganization,PlatformOrganizationPage} from "../domain/types";

const DAY=86_400_000;
const columns=["EMPRESA","PLAN","PRECIO","PRÓXIMO VENCIMIENTO","ÚLTIMO PAGO","SITUACIÓN","CONCIERGE DEL MES","ACCIONES"];
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
  <PageHeader title="Empresas y planes" description="Plan contratado y situación de pago de cada empresa cliente."/>
  <div className="panel management"><EmptyState icon="lock" title="Acceso restringido" text="Solo el administrador de plataforma puede consultar las empresas y sus suscripciones."/></div>
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
 const organizationIds=useMemo(()=>data?.items.map(company=>company.id)??[],[data]);
 /** Uso de Fudia Concierge solo de la página visible; `summary` cuenta toda la cartera. */
 const conciergeUsage=useQuery({
  queryKey:["platform-concierge-usage",organizationIds],
  queryFn:()=>listConciergeUsage(organizationIds),
  enabled:organizationIds.length>0,
 });
 const filtered=Boolean(debouncedQ.trim()||planId||standing);
 const pickStanding=(value:string)=>{setStanding(value);setPage(1)};

 return <>
  <PageHeader title="Empresas y planes" description="Plan contratado, vencimientos y situación de pago de cada empresa cliente."/>
  <CompaniesSummary data={data} loading={list.isLoading} active={standing} onPick={pickStanding}/>
  {organizationIds.length>0&&<ConciergeUsageNotice usage={conciergeUsage}/>}
  <section className="panel management standardized-management platform-companies-panel">
   <div className="toolbar">
    <SearchField value={q} onChange={event=>{setQ(event.target.value);setPage(1)}} placeholder="Buscar por nombre, razón social o RUC..." label="Buscar empresas"/>
    <Select aria-label="Filtrar por plan" value={planId} onChange={event=>{setPlanId(event.target.value);setPage(1)}} disabled={!data}>
     <option value="">Todos los planes</option>
     {data?.planOptions.map(option=><option key={option.value} value={option.value}>{option.label}</option>)}
    </Select>
    <Select aria-label="Filtrar por situación de pago" value={standing} onChange={event=>pickStanding(event.target.value)} disabled={!data}>
     <option value="">Toda situación de pago</option>
     {data?.standingOptions.map(option=><option key={option.value} value={option.value}>{option.label}</option>)}
    </Select>
   </div>
   {list.isLoading?<CompaniesTableSkeleton/>
    :list.isError?<ErrorState title="No pudimos cargar las empresas" text={list.error.message} retrying={list.isFetching} onRetry={()=>void list.refetch()}/>
    :!data?.items.length?<EmptyState icon="building" title={filtered?"Sin coincidencias":"Aún no hay empresas"} text={filtered?"Ninguna empresa coincide con la búsqueda o los filtros aplicados.":"Las empresas registradas desde «Registrar empresa» aparecerán aquí con su plan."}/>
    :<CompaniesTable data={data} busy={list.isFetching} conciergeUsage={conciergeUsage}/>}
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

function CompaniesTable({data,busy,conciergeUsage}:{data:PlatformOrganizationPage;busy:boolean;conciergeUsage:UseQueryResult<ConciergeUsagePage>}){
 const{location}=useSession();
 const[now]=useState(()=>Date.now());
 const[detailId,setDetailId]=useState<string|null>(null);
 const standingLabel=(value:PaymentStanding)=>data.standingOptions.find(option=>option.value===value)?.label??value;
 const date=(value:string)=>formatRegionalDateTime(value,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium"});
 const standing=(company:PlatformOrganization)=><Status tone={paymentStandingTone(company.paymentStanding)}>{standingLabel(company.paymentStanding)}</Status>;
 const companyNote=(company:PlatformOrganization)=>[company.taxId?"RUC "+company.taxId:"",company.active?"":"Inactiva"].filter(Boolean).join(" · ")||"Sin RUC";
 const price=(company:PlatformOrganization)=>company.subscription?<b className="platform-company-amount">{company.subscription.currency} {Number(company.subscription.priceAmount).toFixed(2)}</b>:"—";
 const dueDate=(company:PlatformOrganization)=>{
  const subscription=company.subscription;
  const due=subscription?.status==="trial"?subscription.trialEndsAt??subscription.renewsAt:subscription?.renewsAt??null;
  return due&&subscription?.status!=="cancelled"?<div className="platform-company-cell"><b>{date(due)}</b><small className={company.paymentStanding}>{relativeDays(due,now,subscription?.status==="trial")}</small></div>:<span className="platform-company-muted">—</span>;
 };
 const lastPayment=(company:PlatformOrganization)=>company.lastPayment?<div className="platform-company-cell"><b className="platform-company-amount">{company.lastPayment.currency} {Number(company.lastPayment.amount).toFixed(2)}</b><small>{company.lastPayment.paidAt?date(company.lastPayment.paidAt):"Sin fecha"}</small></div>:<span className="platform-company-muted">Sin pagos</span>;
 const plan=(company:PlatformOrganization)=>company.subscription?<div className="platform-company-cell"><b>{company.subscription.plan.name}</b><small>{company.subscription.billingCycle==="annual"?"Anual":"Mensual"}{company.subscription.autoRenew?" · Renovación automática":""}</small></div>:<span className="platform-company-muted">Sin plan</span>;
 const actions=(company:PlatformOrganization)=><><RowActionButton action="view" label="Ver detalle" onClick={()=>setDetailId(company.id)}/><Link href={companySettingsPath(company.id)} className="ds-icon-button" aria-label={"Gestionar suscripción de "+company.tradeName} data-tooltip="Gestionar suscripción"><Icon name="receipt" size={18}/></Link></>;

 return <><DataTable label="Empresas registradas" containerClassName="hover-scroll" className="platform-companies-table" aria-busy={busy}>
   <thead><tr>{columns.map(column=><th key={column}>{column}</th>)}</tr></thead>
   <tbody>{data.items.map(company=><tr key={company.id} className={company.active?"":"inactive"}>
    <td><div className="platform-company-name"><span className="row-icon"><Icon name="building" size={18}/></span><div><Link className="platform-company-link" href={companySettingsPath(company.id)}>{company.tradeName}</Link><small>{companyNote(company)}</small></div></div></td>
    <td>{plan(company)}</td>
    <td>{price(company)}</td>
    <td>{dueDate(company)}</td>
    <td>{lastPayment(company)}</td>
    <td>{standing(company)}</td>
    <td><ConciergeUsageCell organizationId={company.id} usage={conciergeUsage}/></td>
    <td><div className="table-actions">{actions(company)}</div></td>
   </tr>)}</tbody>
  </DataTable>
 <div className="management-cards" aria-busy={busy}>{data.items.map(company=><article key={company.id}>
  <header><span className="row-icon r0"><Icon name="building"/></span><div><Link className="platform-company-link" href={companySettingsPath(company.id)}>{company.tradeName}</Link><small>{companyNote(company)}</small></div>{standing(company)}</header>
  <dl><div><dt>Plan</dt><dd>{plan(company)}</dd></div><div><dt>Precio</dt><dd>{price(company)}</dd></div><div><dt>Próximo vencimiento</dt><dd>{dueDate(company)}</dd></div><div><dt>Último pago</dt><dd>{lastPayment(company)}</dd></div><div><dt>Concierge del mes</dt><dd><ConciergeUsageCell organizationId={company.id} usage={conciergeUsage}/></dd></div></dl>
  <footer>{actions(company)}</footer>
 </article>)}</div>
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
 return <TableSkeleton columns={columns} rows={5} label="Cargando empresas"/>;
}
