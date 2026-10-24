"use client";
import Link from "next/link";
import {useQuery} from "@tanstack/react-query";
import {ErrorState,Status,ModalHeader,ModalFooter} from "@/design-system";
import {Dialog} from "@/design-system/dialog";
import {Icon} from "@/design-system/icons";
import {useSession} from "@/providers";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {companySettingsPath} from "@/shared/routing/page-routes";
import {getPlatformOrganization} from "../infrastructure/platform-api";
import type {Option,PlatformOrganizationDetail} from "../domain/types";
import {paymentStandingTone,subscriptionStatusLabel,subscriptionStatusTone} from "../domain/subscription-status";
import {ConciergeUsageDialogSection} from "./platform-concierge-usage";

export function CompanyDetailDialog({organizationId,standingOptions,close}:{organizationId:string;standingOptions:Option[];close:()=>void}){
 const detail=useQuery({queryKey:["platform-organization",organizationId],queryFn:()=>getPlatformOrganization(organizationId)});
 const company=detail.data;
 return <div className="modal-backdrop modal-overlay-in">
  <Dialog onResponseClose={close} className="crud-modal company-detail modal-panel-in" role="dialog" aria-modal="true" aria-busy={detail.isLoading} aria-labelledby="company-detail-title">

   <ModalHeader title={company?.tradeName??(detail.isLoading?"Cargando empresa…":"Empresa")} titleId="company-detail-title" icon="building" subtitle="Detalle de la empresa" onClose={close} closeLabel="Cerrar detalle"/>
   {detail.isLoading?<CompanyDetailSkeleton/>
    :detail.isError||!company?<div className="company-detail-body"><ErrorState title="No pudimos cargar el detalle" text={detail.error?.message} retrying={detail.isFetching} onRetry={()=>void detail.refetch()}/></div>
    :<CompanyDetailContent company={company} standingOptions={standingOptions}/>}
  </Dialog>
 </div>;
}

function CompanyDetailContent({company,standingOptions}:{company:PlatformOrganizationDetail;standingOptions:Option[]}){
 const{location}=useSession();
 const date=(value:string|null)=>value?formatRegionalDateTime(value,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium"}):"—";
 const country=countryName(company.countryCode);
 const subscription=company.subscription;
 const due=subscription?.status==="trial"?subscription.trialEndsAt:subscription?.renewsAt??null;
 const limit=(used:number,max:number|null)=>max===null?`${used} · sin límite`:`${used} de ${max}`;

 return <>
  <div className="company-detail-body">
   <section className="company-detail-summary">
    <div><small>Situación</small><Status tone={paymentStandingTone(company.paymentStanding)}>{standingOptions.find(option=>option.value===company.paymentStanding)?.label??company.paymentStanding}</Status></div>
    <div><small>Plan</small><b>{subscription?`${subscription.planName} · ${subscription.billingCycle==="annual"?"Anual":"Mensual"}`:"Sin plan"}</b></div>
    <div><small>{subscription?.status==="trial"?"FIN DE PRUEBA":"PRÓXIMO VENCIMIENTO"}</small><b>{subscription?.status==="cancelled"?"—":date(due)}</b></div>
    <div><small>Total pagado</small><b>{company.payments.paidCount?`${subscription?.currency??company.currency} ${Number(company.payments.totalPaid).toFixed(2)}`:"Sin pagos"}</b>{company.payments.paidCount>0&&<em>{company.payments.paidCount} {company.payments.paidCount===1?"pago":"pagos"} · último {date(company.payments.lastPaidAt)}</em>}</div>
   </section>

   <section className="company-detail-section">
    <header><small>Empresa</small><h3>Datos generales y fiscales</h3></header>
    <dl className="company-detail-fields">
     <div><dt>Razón social</dt><dd>{company.legalName}</dd></div>
     <div><dt>RUC</dt><dd>{company.taxId||"—"}</dd></div>
     <div><dt>País</dt><dd>{country}</dd></div>
     <div><dt>Moneda</dt><dd>{company.currency}</dd></div>
     <div><dt>Impuesto</dt><dd>{company.taxName} {Number(company.taxRate)}% · {company.taxIncluded?"incluido en precios":"no incluido"}</dd></div>
     <div><dt>Zona horaria</dt><dd>{company.timezone}</dd></div>
     <div><dt>Cliente desde</dt><dd>{date(company.createdAt)}</dd></div>
     <div><dt>Estado</dt><dd>{company.active?"Activa":"Inactiva"}</dd></div>
    </dl>
   </section>

   <section className="company-detail-section">
    <header><small>Contrato</small><h3>Suscripción y uso</h3></header>
    {subscription?<dl className="company-detail-fields">
     <div><dt>Estado</dt><dd><Status tone={subscriptionStatusTone(subscription.status)}>{subscriptionStatusLabel(subscription.status)}</Status></dd></div>
     <div><dt>Precio contratado</dt><dd>{subscription.currency} {Number(subscription.priceAmount).toFixed(2)} / {subscription.billingCycle==="annual"?"año":"mes"}</dd></div>
     <div><dt>Inicio</dt><dd>{date(subscription.startedAt)}</dd></div>
     <div><dt>Renovación automática</dt><dd>{subscription.autoRenew?"Sí":"No"}</dd></div>
     <div><dt>Locales</dt><dd>{limit(company.usage.locations,company.usage.maxLocations)}</dd></div>
     <div><dt>Usuarios</dt><dd>{limit(company.usage.users,company.usage.maxUsers)}</dd></div>
     <div><dt>Módulos del plan</dt><dd>{subscription.moduleCount}</dd></div>
     <div><dt>Condiciones</dt><dd>{subscription.termsVersion?`Versión ${subscription.termsVersion}${subscription.termsAcceptedAt?" · aceptadas el "+date(subscription.termsAcceptedAt):""}`:"Sin aceptación registrada"}</dd></div>
    </dl>:<p className="company-detail-empty">La empresa no tiene una suscripción registrada.</p>}
   </section>

   <ConciergeUsageDialogSection organizationId={company.id}/>

   <section className="company-detail-section">
    <header><small>Contacto</small><h3>Administradores</h3></header>
    {company.administrators.length?<ul className="company-detail-list">{company.administrators.map(admin=><li key={admin.email}>
     <div><b>{admin.name}</b>{!admin.active&&<Status active={false}>Inactivo</Status>}</div>
     <a href={"mailto:"+admin.email}>{admin.email}</a>
    </li>)}</ul>:<p className="company-detail-empty">No hay administradores asignados.</p>}
   </section>

   <section className="company-detail-section">
    <header><small>Operación</small><h3>Locales</h3></header>
    {company.locations.length?<ul className="company-detail-list">{company.locations.map(item=><li key={item.id}>
     <div><b>{item.name}</b><small>{item.code}</small>{!item.active&&<Status active={false}>Inactivo</Status>}</div>
     <p>{[item.address,item.phone].filter(Boolean).join(" · ")||"Sin dirección ni teléfono registrados."}</p>
    </li>)}</ul>:<p className="company-detail-empty">La empresa no tiene locales registrados.</p>}
   </section>
  </div>
  <ModalFooter className="company-detail-footer">
   <Link href={companySettingsPath(company.id)} className="button primary"><Icon name="receipt" size={16}/>Gestionar suscripción</Link>
  </ModalFooter>
 </>;
}

function countryName(code:string){
 try{return new Intl.DisplayNames(["es"],{type:"region"}).of(code)??code}catch{return code}
}

function CompanyDetailSkeleton(){
 return <div className="company-detail-body" aria-label="Cargando detalle de la empresa" aria-busy="true">
  <section className="company-detail-summary">{Array.from({length:4},(_,index)=><div key={index}><i className="platform-skeleton-block company-detail-skeleton-label"/><i className="platform-skeleton-block company-detail-skeleton-value"/></div>)}</section>
  {Array.from({length:3},(_,index)=><section className="company-detail-section" key={index}>
   <header><i className="platform-skeleton-block company-detail-skeleton-label"/><i className="platform-skeleton-block company-detail-skeleton-title"/></header>
   <div className="company-detail-fields">{Array.from({length:4},(_,field)=><div key={field}><i className="platform-skeleton-block company-detail-skeleton-label"/><i className="platform-skeleton-block company-detail-skeleton-value"/></div>)}</div>
  </section>)}
 </div>;
}
