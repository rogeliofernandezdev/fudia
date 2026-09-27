"use client";
import "./platform-subscription.css";
import {useMemo,useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,Input,PageHeader,Select} from "@/design-system";
import {Icon} from "@/design-system/icons";
import {useFeedback,useSession} from "@/providers";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {changeOrganizationSubscription,getCurrentOrganizationSubscription,listSubscriptionPlans,recordSubscriptionPayment} from "../infrastructure/platform-api";
import type {OrganizationSubscription,SubscriptionPlan} from "../domain/types";
import {PlatformSubscriptionSkeleton} from "./platform-skeletons";

type SubscriptionDraft={planId:string;billingCycle:"monthly"|"annual";status:OrganizationSubscription["status"];autoRenew:boolean;termsAccepted:boolean};
type PaymentDraft={amount:string;currency:string;status:"pending"|"paid"|"failed"|"refunded";provider:string;externalReference:string;paidAt:string};

export function PlatformSubscriptionPage(){
 const{organization,location}=useSession();
 const subscription=useQuery({queryKey:["organization-subscription"],queryFn:getCurrentOrganizationSubscription});
 const plans=useQuery({queryKey:["platform-plans"],queryFn:listSubscriptionPlans});

 if(subscription.isLoading||plans.isLoading)return <><Header organization={organization?.name}/><PlatformSubscriptionSkeleton/></>;
 if(subscription.isError||plans.isError||!subscription.data)return <><Header organization={organization?.name}/><div className="panel subscription-state error"><b>No pudimos cargar la suscripción.</b><Button kind="secondary" onClick={()=>{void subscription.refetch();void plans.refetch()}}>Reintentar</Button></div></>;

 const key=[subscription.data.id,subscription.data.plan.id,subscription.data.billingCycle,subscription.data.status,subscription.data.autoRenew,subscription.data.renewsAt].join(":");
 return <><Header organization={organization?.name}/><SubscriptionWorkspace key={key} current={subscription.data} plans={plans.data?.items??[]} country={location?.country} timeZone={location?.timezone}/></>;
}

function SubscriptionWorkspace({current,plans,country,timeZone}:{current:OrganizationSubscription;plans:SubscriptionPlan[];country?:string;timeZone?:string}){
 const{notify}=useFeedback();
 const client=useQueryClient();
 const[draft,setDraft]=useState<SubscriptionDraft>({
  planId:current.plan.id,
  billingCycle:current.billingCycle,
  status:current.status,
  autoRenew:current.autoRenew,
  termsAccepted:false,
 });
 const[payment,setPayment]=useState<PaymentDraft>({
  amount:current.priceAmount,
  currency:current.currency,
  status:"paid",
  provider:"manual",
  externalReference:"",
  paidAt:"",
 });

 const selectedPlan=useMemo(()=>plans.find(plan=>plan.id===draft.planId)??null,[plans,draft.planId]);
 const availablePlans=plans.filter(plan=>(plan.active&&plan.code!=="legacy")||plan.id===current.plan.id);
 const planChanged=draft.planId!==current.plan.id;
 const termsChanged=selectedPlan?.termsVersion!==current.termsVersion;
 const needsAcceptance=planChanged||termsChanged;

 const save=useMutation({
  mutationFn:()=>changeOrganizationSubscription(draft),
  onSuccess:data=>{
   setDraft({planId:data.plan.id,billingCycle:data.billingCycle,status:data.status,autoRenew:data.autoRenew,termsAccepted:false});
   setPayment(value=>({...value,amount:data.priceAmount,currency:data.currency}));
   client.setQueryData(["organization-subscription"],data);
   void client.invalidateQueries({queryKey:["session-context"]});
   notify({tone:"success",title:"Suscripción actualizada",message:"El plan, estado y módulos de la empresa quedaron sincronizados."});
  },
  onError:e=>notify({tone:"danger",title:"No se pudo actualizar",message:e.message}),
 });
 const pay=useMutation({
  mutationFn:()=>recordSubscriptionPayment({...payment,paidAt:payment.paidAt?new Date(payment.paidAt).toISOString():""}),
  onSuccess:()=>{
   setPayment(value=>({...value,status:"paid",provider:"manual",externalReference:"",paidAt:""}));
   void client.invalidateQueries({queryKey:["organization-subscription"]});
   notify({tone:"success",title:"Pago registrado",message:"El movimiento quedó asociado a la suscripción."});
  },
  onError:e=>notify({tone:"danger",title:"No se pudo registrar",message:e.message}),
 });

 return <>
  <SubscriptionSummary current={current} country={country} timeZone={timeZone}/>
  <div className="platform-subscription-grid">
   <section className="panel subscription-editor">
    <header><span className="subscription-header-icon"><Icon name="contract" size={18}/></span><div><small>Contrato</small><h2>Plan y ciclo</h2></div></header>
    <form onSubmit={e=>{e.preventDefault();save.mutate()}}>
     <div className="form-grid">
      <label className="span-2">Plan<Select value={draft.planId} onChange={e=>setDraft({...draft,planId:e.target.value,termsAccepted:false})}>{availablePlans.map(plan=><option value={plan.id} key={plan.id}>{plan.name+" · "+plan.code}</option>)}</Select></label>
      <label>Ciclo<Select value={draft.billingCycle} onChange={e=>setDraft({...draft,billingCycle:e.target.value as "monthly"|"annual"})}><option value="monthly">Mensual</option><option value="annual">Anual</option></Select></label>
      <label>Estado<Select value={draft.status} onChange={e=>setDraft({...draft,status:e.target.value as OrganizationSubscription["status"]})}><option value="trial">Prueba</option><option value="active">Activa</option><option value="past_due">Pago pendiente</option><option value="cancelled">Cancelada</option></Select></label>
      <label>Precio del plan<Input readOnly value={selectedPlan?(selectedPlan.currency+" "+Number(draft.billingCycle==="annual"?selectedPlan.annualPrice:selectedPlan.monthlyPrice).toFixed(2)):""}/></label>
      <label className="switch-row compact"><input type="checkbox" checked={draft.autoRenew} onChange={e=>setDraft({...draft,autoRenew:e.target.checked})}/><span/><b>Renovación automática</b></label>
     </div>
     {selectedPlan&&<dl className="subscription-plan-preview"><div><dt>Prueba</dt><dd>{selectedPlan.trialDays?selectedPlan.trialDays+" días":"No"}</dd></div><div><dt>Locales</dt><dd>{selectedPlan.maxLocations??"Sin límite"}</dd></div><div><dt>Usuarios</dt><dd>{selectedPlan.maxUsers??"Sin límite"}</dd></div><div><dt>Módulos</dt><dd>{selectedPlan.moduleKeys.length}</dd></div><div><dt>Condiciones</dt><dd>{selectedPlan.termsVersion}</dd></div></dl>}
     {needsAcceptance&&<label className="subscription-accept"><input type="checkbox" checked={draft.termsAccepted} onChange={e=>setDraft({...draft,termsAccepted:e.target.checked})}/><span><Icon name="check" size={13}/></span><div><b>El cliente aceptó las nuevas condiciones</b><small>Se registrará la aceptación de la versión {selectedPlan?.termsVersion} al aplicar el cambio de plan.</small></div></label>}
     <footer><Button type="submit" icon="check" disabled={save.isPending||(needsAcceptance&&!draft.termsAccepted)}>{save.isPending?"Guardando…":"Aplicar cambio"}</Button></footer>
    </form>
   </section>

   <section className="panel subscription-editor">
    <header><span className="subscription-header-icon"><Icon name="payment" size={18}/></span><div><small>Cobro SaaS</small><h2>Registrar pago</h2></div></header>
    <form onSubmit={e=>{e.preventDefault();pay.mutate()}}>
     <div className="form-grid">
      <label>Monto<Input required type="number" min="0" step="0.01" value={payment.amount} onChange={e=>setPayment({...payment,amount:e.target.value})}/></label>
      <label>Moneda<Input readOnly value={payment.currency}/></label>
      <label>Estado<Select value={payment.status} onChange={e=>setPayment({...payment,status:e.target.value as PaymentDraft["status"]})}><option value="paid">Pagado</option><option value="pending">Pendiente</option><option value="failed">Fallido</option><option value="refunded">Reembolsado</option></Select></label>
      <label>Proveedor<Input required value={payment.provider} onChange={e=>setPayment({...payment,provider:e.target.value})} placeholder="manual / proveedor"/></label>
      <label>Referencia externa<Input value={payment.externalReference} onChange={e=>setPayment({...payment,externalReference:e.target.value})} placeholder="ID del procesador o recibo"/></label>
      <label>Fecha del pago<Input type="datetime-local" value={payment.paidAt} onChange={e=>setPayment({...payment,paidAt:e.target.value})}/></label>
     </div>
     <footer><Button type="submit" icon="check" disabled={pay.isPending||!payment.amount||!payment.provider}>{pay.isPending?"Registrando…":"Registrar pago"}</Button></footer>
    </form>
   </section>
  </div>

  <section className="panel subscription-history">
   <header><span className="subscription-header-icon"><Icon name="ledger" size={18}/></span><div><small>Historial</small><h2>Pagos de suscripción</h2></div><b>{current.payments.length}</b></header>
   {current.payments.length?<div className="table-wrap"><table><thead><tr><th>FECHA</th><th>MONTO</th><th>ESTADO</th><th>PROVEEDOR</th><th>REFERENCIA</th></tr></thead><tbody>{current.payments.map(item=><tr key={item.id}><td>{dateLabel(item.paidAt??item.createdAt,country,timeZone)}</td><td><b>{item.currency+" "+Number(item.amount).toFixed(2)}</b></td><td><span className={"subscription-pay-status "+item.status}>{paymentStatus(item.status)}</span></td><td>{item.provider}</td><td>{item.externalReference??"—"}</td></tr>)}</tbody></table></div>:<div className="subscription-empty">Todavía no hay pagos registrados para esta empresa.</div>}
  </section>
 </>;
}

function SubscriptionSummary({current,country,timeZone}:{current:OrganizationSubscription;country?:string;timeZone?:string}){
 const price=current.currency+" "+Number(current.priceAmount).toFixed(2)+(current.billingCycle==="annual"?" / año":" / mes");
 const nextDate=current.status==="trial"?current.trialEndsAt:current.renewsAt??current.currentPeriodEndsAt;
 const nextLabel=current.status==="trial"?"Fin de prueba":current.autoRenew?"Próxima renovación":"Fin del periodo";
 const lastPayment=current.payments.find(item=>item.status==="paid")??null;
 return <section className="subscription-summary">
  <div className="subscription-summary-main">
   <span className={"subscription-state-badge "+current.status}>{statusLabel(current.status)}</span>
   <h2>{current.plan.name}</h2>
   <p>{price} · {current.billingCycle==="annual"?"Facturación anual":"Facturación mensual"}{current.autoRenew?" · Renovación automática":""}</p>
  </div>
  <dl className="subscription-summary-facts">
   <div><dt>{nextLabel}</dt><dd>{nextDate?dateLabel(nextDate,country,timeZone):"—"}</dd></div>
   <div><dt>Condiciones</dt><dd>{current.termsVersion??"—"}{current.termsAcceptedAt?<small>Aceptadas el {dateLabel(current.termsAcceptedAt,country,timeZone)}</small>:<small>Sin aceptación registrada</small>}</dd></div>
   <div><dt>Último pago</dt><dd>{lastPayment?lastPayment.currency+" "+Number(lastPayment.amount).toFixed(2):"—"}{lastPayment&&<small>{dateLabel(lastPayment.paidAt??lastPayment.createdAt,country,timeZone)}</small>}</dd></div>
  </dl>
  <div className="subscription-usage">
   <UsageMeter icon="store" label="Locales" used={current.usage.locations} max={current.plan.maxLocations}/>
   <UsageMeter icon="users" label="Usuarios" used={current.usage.users} max={current.plan.maxUsers}/>
  </div>
 </section>;
}
function UsageMeter({icon,label,used,max}:{icon:"store"|"users";label:string;used:number;max:number|null}){
 const ratio=max?Math.min(1,used/max):0;
 const tone=max&&used>=max?"full":max&&ratio>=.8?"near":"";
 return <div className={"subscription-meter "+tone}>
  <span className="subscription-meter-icon"><Icon name={icon} size={16}/></span>
  <div>
   <div className="subscription-meter-copy"><b>{label}</b><span>{used}{max?" de "+max:" · sin límite"}</span></div>
   <div className="subscription-meter-bar" role="progressbar" aria-label={label} aria-valuemin={0} aria-valuemax={max??undefined} aria-valuenow={used}><i style={{width:(max?ratio*100:0)+"%"}}/></div>
  </div>
 </div>;
}
function Header({organization}:{organization?:string}){return <PageHeader eyebrow="PLATAFORMA" title="Suscripción de empresa" description={organization?"Gestiona contrato, plan y cobros de "+organization+".":"Gestiona el contrato y cobros de la empresa activa."}/>;}
function statusLabel(status:OrganizationSubscription["status"]){return status==="trial"?"Prueba":status==="active"?"Activa":status==="past_due"?"Pago pendiente":"Cancelada";}
function paymentStatus(status:string){return status==="paid"?"Pagado":status==="pending"?"Pendiente":status==="failed"?"Fallido":"Reembolsado";}
function dateLabel(value:string,country?:string,timeZone?:string){return formatRegionalDateTime(value,{country,timeZone},{dateStyle:"medium"})}
