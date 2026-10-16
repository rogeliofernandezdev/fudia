"use client";
import Link from "next/link";
import {useQuery} from "@tanstack/react-query";
import {Button,PageHeader} from "@/design-system";
import {Icon} from "@/design-system/icons";
import {useSession} from "@/providers";
import {ApiClientError} from "@/shared/api/client";
import {pageRoutes} from "@/shared/routing/page-routes";
import {getOrganizationSubscription,listSubscriptionPlans} from "../infrastructure/platform-api";
import {PlatformSubscriptionSkeleton} from "./platform-skeletons";
import {SubscriptionWorkspace} from "./platform-subscription-page";
import {RestrictedCompanies} from "./platform-companies-page";

export function PlatformCompanyDetailPage({organizationId}:{organizationId:string}){
 const{user}=useSession();
 if(!user?.platformAdmin)return <RestrictedCompanies/>;
 return <CompanySubscription organizationId={organizationId}/>;
}

function CompanySubscription({organizationId}:{organizationId:string}){
 const{location}=useSession();
 const subscription=useQuery({queryKey:["organization-subscription",organizationId],queryFn:()=>getOrganizationSubscription(organizationId)});
 const plans=useQuery({queryKey:["platform-plans"],queryFn:listSubscriptionPlans});
 const company=subscription.data?.organization;

 const back=<Link href={pageRoutes.companiesSettings} className="button secondary"><Icon name="chevronLeft" size={16}/>Empresas y planes</Link>;
 const header=<PageHeader
  eyebrow="CONFIGURACIÓN · EMPRESAS Y PLANES"
  title={company?.tradeName??"Suscripción de empresa"}
  description={company?[company.taxId?"RUC "+company.taxId:"",company.active?"":"Empresa inactiva"].filter(Boolean).join(" · ")||"Contrato, plan y cobros de la empresa.":"Contrato, plan y cobros de la empresa."}
  action={back}
 />;

 if(subscription.isLoading||plans.isLoading)return <>{header}<PlatformSubscriptionSkeleton/></>;
 if(subscription.error instanceof ApiClientError&&subscription.error.status===404)return <>{header}<div className="panel catalog-state"><span><Icon name="building"/></span><b>Sin suscripción</b><p>{subscription.error.message}</p></div></>;
 if(subscription.isError||plans.isError||!subscription.data)return <>{header}<div className="panel catalog-state error"><span><Icon name="alert"/></span><b>No pudimos cargar la suscripción.</b><p>{subscription.error?.message??plans.error?.message}</p><Button kind="ghost" onClick={()=>{void subscription.refetch();void plans.refetch()}}>Reintentar</Button></div></>;

 const data=subscription.data;
 const key=[data.id,data.plan.id,data.billingCycle,data.status,data.autoRenew,data.renewsAt].join(":");
 return <>{header}<SubscriptionWorkspace key={key} organizationId={organizationId} current={data} plans={plans.data?.items??[]} country={location?.country} timeZone={location?.timezone}/></>;
}
