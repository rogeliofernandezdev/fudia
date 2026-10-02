"use client";
import "./platform-subscription.css";
import {useMemo} from "react";
import {useForm,useWatch} from "react-hook-form";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,FormField,Input,PageHeader,Select} from "@/design-system";
import {Icon} from "@/design-system/icons";
import {useFeedback,useSession} from "@/providers";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {changeOrganizationSubscription,getCurrentOrganizationSubscription,listSubscriptionPlans,recordSubscriptionPayment} from "../infrastructure/platform-api";
import type {OrganizationSubscription,SubscriptionPlan} from "../domain/types";
import {subscriptionChangeSchema,subscriptionPaymentSchema} from "../domain/plan-schema";
import {zodResolver} from "@/shared/forms/zod-resolver";
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
 const change=useForm<SubscriptionDraft>({defaultValues:{planId:current.plan.id,billingCycle:current.billingCycle,status:current.status,autoRenew:current.autoRenew,termsAccepted:false},resolver:zodResolver<SubscriptionDraft>(subscriptionChangeSchema),mode:"onSubmit",reValidateMode:"onChange"});
 const paymentForm=useForm<PaymentDraft>({defaultValues:{amount:current.priceAmount,currency:current.currency,status:"paid",provider:"manual",externalReference:"",paidAt:""},resolver:zodResolver<PaymentDraft>(subscriptionPaymentSchema),mode:"onSubmit",reValidateMode:"onChange"});
 const[planId,billingCycle]=useWatch({control:change.control,name:["planId","billingCycle"]});
 const paymentStatusValue=useWatch({control:paymentForm.control,name:"status"});

 const selectedPlan=useMemo(()=>plans.find(plan=>plan.id===planId)??null,[plans,planId]);
 const availablePlans=plans.filter(plan=>(plan.active&&plan.code!=="legacy")||plan.id===current.plan.id);
 const planChanged=planId!==current.plan.id;
 const termsChanged=selectedPlan?.termsVersion!==current.termsVersion;
 const needsAcceptance=planChanged||termsChanged;

 const save=useMutation({
  mutationFn:(draft:SubscriptionDraft)=>changeOrganizationSubscription(draft),
  onSuccess:data=>{
   change.reset({planId:data.plan.id,billingCycle:data.billingCycle,status:data.status,autoRenew:data.autoRenew,termsAccepted:false});
   paymentForm.setValue("amount",data.priceAmount);paymentForm.setValue("currency",data.currency);
   client.setQueryData(["organization-subscription"],data);
   void client.invalidateQueries({queryKey:["session-context"]});
   notify({tone:"success",title:"Suscripción actualizada",message:"El plan, estado y módulos de la empresa quedaron sincronizados."});
  },
  onError:e=>notify({tone:"danger",title:"No se pudo actualizar",message:e.message}),
 });
 const pay=useMutation({
  mutationFn:(payment:PaymentDraft)=>recordSubscriptionPayment({...payment,paidAt:payment.paidAt?new Date(payment.paidAt).toISOString():""}),
  onSuccess:(_,payment)=>{
   paymentForm.reset({amount:payment.amount,currency:payment.currency,status:"paid",provider:"manual",externalReference:"",paidAt:""});
   void client.invalidateQueries({queryKey:["organization-subscription"]});
   notify({tone:"success",title:"Pago registrado",message:"El movimiento quedó asociado a la suscripción."});
  },
  onError:e=>notify({tone:"danger",title:"No se pudo registrar",message:e.message}),
 });

 return <>
  <div className="platform-subscription-grid">
   <section className="panel subscription-editor">
    <header><div><small>Contrato</small><h2>Plan y ciclo</h2></div><span className={"subscription-state-badge "+current.status}>{statusLabel(current.status)}</span></header>
    <form onSubmit={change.handleSubmit(values=>{if(needsAcceptance&&!values.termsAccepted){change.setError("termsAccepted",{type:"custom",message:"Registra la aceptación de las nuevas condiciones."});return}save.mutate(values)})} noValidate>
     <div className="form-grid">
      <FormField className="span-2" label="Plan" error={change.formState.errors.planId?.message}><Select {...change.register("planId",{onChange:()=>change.setValue("termsAccepted",false)})}>{availablePlans.map(plan=><option value={plan.id} key={plan.id}>{plan.name}</option>)}</Select></FormField>
      <FormField label="Ciclo"><Select {...change.register("billingCycle")}><option value="monthly">Mensual</option><option value="annual">Anual</option></Select></FormField>
      <FormField label="Estado"><Select {...change.register("status")}><option value="trial">Prueba</option><option value="active">Activa</option><option value="past_due">Pago pendiente</option><option value="cancelled">Cancelada</option></Select></FormField>
      <FormField label="Precio del plan"><Input readOnly tabIndex={-1} value={selectedPlan?(selectedPlan.currency+" "+Number(billingCycle==="annual"?selectedPlan.annualPrice:selectedPlan.monthlyPrice).toFixed(2)):""}/></FormField>
      <label className="switch-row compact"><input type="checkbox" {...change.register("autoRenew")}/><span/><b>Renovación automática</b></label>
     </div>
     {selectedPlan&&<dl className="subscription-plan-preview"><div><dt>Prueba</dt><dd>{selectedPlan.trialDays?selectedPlan.trialDays+" días":"No"}</dd></div><div><dt>Locales</dt><dd>{selectedPlan.maxLocations??"Sin límite"}<small>En uso: {current.usage.locations}</small></dd></div><div><dt>Usuarios</dt><dd>{selectedPlan.maxUsers??"Sin límite"}<small>En uso: {current.usage.users}</small></dd></div><div><dt>Módulos</dt><dd>{selectedPlan.moduleKeys.length}</dd></div><div><dt>Condiciones</dt><dd>{selectedPlan.termsVersion}</dd></div></dl>}
     {needsAcceptance&&<label className={"subscription-accept"+(change.formState.errors.termsAccepted?" has-error":"")}><input type="checkbox" {...change.register("termsAccepted")}/><span><Icon name="check" size={13}/></span><div><b>El cliente aceptó las nuevas condiciones</b><small>Se registrará la aceptación de la versión {selectedPlan?.termsVersion} al aplicar el cambio de plan.</small>{change.formState.errors.termsAccepted?.message&&<small className="field-error" role="alert">{change.formState.errors.termsAccepted.message}</small>}</div></label>}
     <footer><Button type="submit" icon="check" disabled={save.isPending}>{save.isPending?"Guardando…":"Aplicar cambio"}</Button></footer>
    </form>
   </section>

   <section className="panel subscription-editor">
    <header><div><small>Cobro SaaS</small><h2>Registrar pago</h2></div></header>
    <form onSubmit={paymentForm.handleSubmit(values=>pay.mutate(values))} noValidate>
     <div className="form-grid">
      <FormField label="Monto" error={paymentForm.formState.errors.amount?.message}><Input type="number" inputMode="decimal" min="0" step="0.01" {...paymentForm.register("amount")}/></FormField>
      <FormField label="Moneda"><Input readOnly tabIndex={-1} {...paymentForm.register("currency")}/></FormField>
      <FormField label="Estado"><Select {...paymentForm.register("status")}><option value="paid">Pagado</option><option value="pending">Pendiente</option><option value="failed">Fallido</option><option value="refunded">Reembolsado</option></Select></FormField>
      <FormField label="Proveedor" help="Pasarela o canal por el que se cobró." error={paymentForm.formState.errors.provider?.message}><Input {...paymentForm.register("provider")} placeholder="manual"/></FormField>
      <FormField label="Referencia externa" optional error={paymentForm.formState.errors.externalReference?.message}><Input {...paymentForm.register("externalReference")} placeholder="ID del procesador o recibo"/></FormField>
      <FormField label="Fecha del pago" optional={paymentStatusValue!=="paid"} error={paymentForm.formState.errors.paidAt?.message}><Input type="datetime-local" {...paymentForm.register("paidAt")}/></FormField>
     </div>
     <footer><Button type="submit" icon="check" disabled={pay.isPending}>{pay.isPending?"Registrando…":"Registrar pago"}</Button></footer>
    </form>
   </section>
  </div>

  <section className="panel subscription-history">
   <header><div><small>Historial</small><h2>Pagos de suscripción</h2></div><b>{current.payments.length}</b></header>
   {current.payments.length?<div className="table-wrap"><table><thead><tr><th>FECHA</th><th>MONTO</th><th>ESTADO</th><th>PROVEEDOR</th><th>REFERENCIA</th></tr></thead><tbody>{current.payments.map(item=><tr key={item.id}><td>{dateLabel(item.paidAt??item.createdAt,country,timeZone)}</td><td><b>{item.currency+" "+Number(item.amount).toFixed(2)}</b></td><td><span className={"subscription-pay-status "+item.status}>{paymentStatus(item.status)}</span></td><td>{item.provider}</td><td>{item.externalReference??"—"}</td></tr>)}</tbody></table></div>:<div className="subscription-empty">Todavía no hay pagos registrados para esta empresa.</div>}
  </section>
 </>;
}

function Header({organization}:{organization?:string}){return <PageHeader eyebrow="PLATAFORMA" title="Suscripción de empresa" description={organization?"Gestiona contrato, plan y cobros de "+organization+".":"Gestiona el contrato y cobros de la empresa activa."}/>;}
function statusLabel(status:OrganizationSubscription["status"]){return status==="trial"?"Prueba":status==="active"?"Activa":status==="past_due"?"Pago pendiente":"Cancelada";}
function paymentStatus(status:string){return status==="paid"?"Pagado":status==="pending"?"Pendiente":status==="failed"?"Fallido":"Reembolsado";}
function dateLabel(value:string,country?:string,timeZone?:string){return formatRegionalDateTime(value,{country,timeZone},{dateStyle:"medium"})}
