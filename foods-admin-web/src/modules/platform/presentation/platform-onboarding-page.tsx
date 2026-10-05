"use client";
import "./platform-onboarding.css";
import Link from "next/link";
import {useEffect,useMemo,useState} from "react";
import {useForm,useWatch} from "react-hook-form";
import {useMutation,useQuery} from "@tanstack/react-query";
import {Icon,IconName} from "@/design-system/icons";
import {LocationMap} from "@/design-system/location-map";
import {ApiClientError} from "@/shared/api/client";
import {FormField as Field,Input,PageHeader,Select} from "@/design-system";
import {useFeedback} from "@/providers/feedback-provider";
import type {PlatformOnboardingDraft} from "../domain/types";
import {onboardingResolver,onboardingStepFields} from "../domain/onboarding-schemas";
import {createPlatformOrganization,getPlatformOnboardingContext} from "../infrastructure/platform-api";
import {PlatformOnboardingSkeleton} from "./platform-skeletons";

const blank:PlatformOnboardingDraft={
 legalName:"",tradeName:"",taxId:"",timezone:"America/Lima",
 planId:"",billingCycle:"monthly",termsAccepted:false,
 country:"PE",currency:"PEN",currencyPosition:"before",taxName:"IGV",taxRate:"18",taxIncluded:false,
 locationName:"",locationCode:"",address:"",locationPhone:"",locationHours:"",latitude:"",longitude:"",
 adminName:"",adminEmail:"",adminPassword:"",
};

const steps:Array<{key:string;title:string;icon:IconName;heading:string}>=[
  {key:"empresa",title:"Empresa",icon:"building",heading:"Datos legales"},
  {key:"plan",title:"Plan y contrato",icon:"contract",heading:"Suscripción SaaS"},
  {key:"fiscal",title:"Fiscal",icon:"receipt",heading:"Perfil fiscal del primer local"},
  {key:"local",title:"Primer local",icon:"mapPin",heading:"Sede inicial"},
  {key:"admin",title:"Administrador",icon:"userCheck",heading:"Administrador de empresa"},
];
const lastStep=steps.length-1;

function listTimezones(current:string){
  let zones:string[]=[];
  try{zones=Intl.supportedValuesOf("timeZone")}catch{zones=[]}
  return zones.includes(current)?zones:[current,...zones];
}

export function PlatformOnboardingPage(){
  const{notify}=useFeedback();
  const[step,setStep]=useState(0);
  const[reached,setReached]=useState(0);
  const context=useQuery({queryKey:["platform-onboarding-context"],queryFn:getPlatformOnboardingContext});
  const ctx=context.data;
  const loadError=context.error instanceof Error?context.error.message:"";
  const firstPlanId=ctx?.plans[0]?.id??"";

  const{register,control,setValue,getValues,trigger,reset,handleSubmit,formState:{errors}}=useForm<PlatformOnboardingDraft>({
    defaultValues:blank,resolver:onboardingResolver,mode:"onSubmit",reValidateMode:"onChange",
  });
  const[planId,billingCycle,timezone,address,latitude,longitude]=useWatch({control,name:["planId","billingCycle","timezone","address","latitude","longitude"]});
  const selectedPlan=ctx?.plans.find(plan=>plan.id===planId)??null;
  const timezones=useMemo(()=>listTimezones(timezone),[timezone]);

  useEffect(()=>{if(firstPlanId&&!getValues("planId"))setValue("planId",firstPlanId)},[firstPlanId,getValues,setValue]);
  useEffect(()=>{
    if(!ctx?.countryOptions.length||!ctx.currencyOptions.length)return;
    const currentCountry=getValues("country");
    const selectedCountry=ctx.countryOptions.find(country=>country.code===currentCountry)??ctx.countryOptions[0];
    const countryChanged=currentCountry!==selectedCountry.code;
    if(countryChanged)setValue("country",selectedCountry.code);
    const currentCurrency=getValues("currency");
    const defaultCurrency=ctx.currencyOptions.some(currency=>currency.code===selectedCountry.defaultCurrency)
      ?selectedCountry.defaultCurrency
      :ctx.currencyOptions[0].code;
    if(countryChanged||!ctx.currencyOptions.some(currency=>currency.code===currentCurrency))setValue("currency",defaultCurrency);
  },[ctx,getValues,setValue]);

  const save=useMutation({
    mutationFn:(values:PlatformOnboardingDraft)=>createPlatformOrganization(values),
    onSuccess:()=>{notify({tone:"success",title:"Empresa registrada",message:"La empresa, su suscripción, el local y el Administrador de empresa quedaron listos."});reset({...blank,planId:firstPlanId});setStep(0);setReached(0)},
    onError:e=>{const message=e instanceof ApiClientError&&e.status>=500&&e.correlationId?`${e.message} Código de seguimiento: ${e.correlationId}`:e.message;notify({tone:"danger",title:"No se pudo registrar",message})},
  });

  function goTo(index:number){if(index<=reached)setStep(index)}
  async function next(){
    if(!await trigger(onboardingStepFields[step],{shouldFocus:true}))return;
    const target=step+1;
    setStep(target);setReached(r=>Math.max(r,target));
  }
  const submit=handleSubmit(values=>save.mutate(values),invalid=>{
    const index=onboardingStepFields.findIndex(fields=>fields.some(field=>invalid[field]));
    if(index>=0&&index!==step)setStep(index);
  });
  const err=(field:keyof PlatformOnboardingDraft)=>errors[field]?.message;
  const countryField=register("country",{onChange:e=>{const c=ctx?.countryOptions.find(x=>x.code===e.target.value);if(c&&ctx?.currencyOptions.some(currency=>currency.code===c.defaultCurrency))setValue("currency",c.defaultCurrency)}});
  const current=steps[step];

  return <><PageHeader eyebrow="PLATAFORMA" title="Registrar empresa" description="Alta única y transaccional: empresa, contrato, fiscalidad, primer local y Administrador de empresa."/>
  {loadError?<div className="catalog-state error"><span><Icon name="alert"/></span><b>No pudimos cargar los catálogos</b><p>{loadError}</p><button className="button secondary" onClick={()=>void context.refetch()}>Reintentar</button></div>:
  context.isLoading?<PlatformOnboardingSkeleton stepCount={steps.length}/>:
  !ctx?.plans.length?<div className="catalog-state"><span><Icon name="settings"/></span><b>Primero crea un plan comercial</b><p>El alta de una empresa exige un plan activo con precio, límites, condiciones y módulos definidos.</p><Link className="button primary" href="/platform/plans"><Icon name="plus" size={16}/>Crear plan</Link></div>:
  !ctx.countryOptions.length||!ctx.currencyOptions.length?<div className="catalog-state"><span><Icon name="settings"/></span><b>Configura un país con WhatsApp operativo</b><p>El onboarding necesita al menos un país activo, una moneda activa y un canal de WhatsApp con phone_number_id y token configurados.</p><Link className="button primary" href="/platform/configuracion-global"><Icon name="settings" size={16}/>Abrir Configuración Global</Link></div>:
  <div className="onboarding-wizard">
    <nav className="wizard-steps" aria-label="Pasos del registro">
      {steps.map((s,i)=><button key={s.key} type="button" className={"wizard-step"+(i===step?" active":i<reached?" done":"")} onClick={()=>goTo(i)} disabled={i>reached||save.isPending} aria-current={i===step?"step":undefined}>
        <span className="wizard-step-icon">{i<reached&&i!==step?<Icon name="check" size={18}/>:<Icon name={s.icon} size={18}/>}</span>
        <span className="wizard-step-label"><small>{i===step?"En curso":i<reached?"Completado":"Pendiente"}</small><b>{s.title}</b></span>
      </button>)}
    </nav>

    <form className="onboarding-form" noValidate onSubmit={e=>{e.preventDefault();if(step===lastStep)void submit();else void next()}}>
      <section className="panel management">
        <header><span className="modal-title-icon"><Icon name={current.icon} size={18}/></span><h2>{current.heading}</h2></header>

        {step===0&&<div className="form-grid">
          <Field className="span-2" label="Razón social" error={err("legalName")}><Input maxLength={180} {...register("legalName")} placeholder="Restaurante Perú SAC" autoComplete="organization"/></Field>
          <Field label="Nombre comercial" error={err("tradeName")}><Input maxLength={180} {...register("tradeName")} placeholder="Mi restaurante"/></Field>
          <Field label="Identificación fiscal" error={err("taxId")}><Input maxLength={32} inputMode="numeric" {...register("taxId")} placeholder="RUC 20512345678"/></Field>
          <Field className="span-2" label="Zona horaria" error={err("timezone")}><Select {...register("timezone")}>{timezones.map(z=><option key={z} value={z}>{z.replaceAll("_"," ")}</option>)}</Select></Field>
        </div>}

        {step===1&&<>
          <div className="form-grid">
            <Field className="span-2" label="Plan contratado" error={err("planId")}><Select {...register("planId")}>{ctx.plans.map(plan=><option key={plan.id} value={plan.id}>{plan.name+" · "+plan.code}</option>)}</Select></Field>
            <Field label="Ciclo de facturación"><Select {...register("billingCycle")}><option value="monthly">Mensual</option><option value="annual">Anual</option></Select></Field>
            <Field label="Precio contratado"><Input readOnly tabIndex={-1} value={selectedPlan?(selectedPlan.currency+" "+Number(billingCycle==="annual"?selectedPlan.annualPrice:selectedPlan.monthlyPrice).toFixed(2)):""}/></Field>
          </div>
          {selectedPlan&&<dl className="onboarding-plan-summary">
            <div><dt>Prueba gratuita</dt><dd>{selectedPlan.trialDays?selectedPlan.trialDays+" días":"No incluida"}</dd></div>
            <div><dt>Límite de locales</dt><dd>{selectedPlan.maxLocations??"Sin límite"}</dd></div>
            <div><dt>Límite de usuarios</dt><dd>{selectedPlan.maxUsers??"Sin límite"}</dd></div>
            <div><dt>Módulos</dt><dd>{selectedPlan.moduleKeys.length}</dd></div>
            <div><dt>Condiciones</dt><dd>{selectedPlan.termsVersion}</dd></div>
          </dl>}
          <label className={"onboarding-terms"+(errors.termsAccepted?" has-error":"")}>
            <input type="checkbox" {...register("termsAccepted")}/><span aria-hidden="true"><Icon name="check" size={13}/></span>
            <div><b>Condiciones aceptadas por el cliente</b>{err("termsAccepted")&&<small className="field-error" role="alert">{err("termsAccepted")}</small>}</div>
          </label>
        </>}

        {step===2&&<div className="form-grid">
          <Field label="País" error={err("country")}><Select {...countryField}>{ctx.countryOptions.map(c=><option key={c.code} value={c.code}>{c.name+" ("+c.code+")"}</option>)}</Select></Field>
          <Field label="Moneda" error={err("currency")}><Select {...register("currency")}>{ctx.currencyOptions.map(c=><option key={c.code} value={c.code}>{c.code+" — "+c.name}</option>)}</Select></Field>
          <Field label="Nombre del impuesto" error={err("taxName")}><Input maxLength={30} {...register("taxName")} placeholder="IGV"/></Field>
          <Field label="Porcentaje %" error={err("taxRate")}><Input type="number" inputMode="decimal" min="0" max="100" step="0.0001" {...register("taxRate")} placeholder="18"/></Field>
          <Field label="Posición del símbolo"><Select {...register("currencyPosition")}><option value="before">Antes del monto</option><option value="after">Después del monto</option></Select></Field>
          <label className="switch-row compact"><input type="checkbox" {...register("taxIncluded")}/><span/><b>Impuesto incluido en el precio</b></label>
        </div>}

        {step===3&&<div className="form-grid">
          <Field className="span-2" label="Nombre del local" error={err("locationName")}><Input {...register("locationName")} placeholder="Sede principal"/></Field>
          <Field label="Teléfono"><Input type="tel" {...register("locationPhone")} placeholder="+51 999 888 777"/></Field>
          <Field label="Horario de atención"><Input {...register("locationHours")} placeholder="Lun-Dom 12:00-23:00"/></Field>
          <div className={"form-field span-2"+(errors.address?" has-error":"")}>
            <LocationMap latitude={latitude?Number(latitude):null} longitude={longitude?Number(longitude):null} address={address} onAddressChange={a=>setValue("address",a,{shouldValidate:Boolean(errors.address)})} onChange={c=>{setValue("latitude",c.lat.toFixed(7));setValue("longitude",c.lng.toFixed(7))}}/>
            {err("address")&&<small className="field-error" role="alert">{err("address")}</small>}
          </div>
        </div>}

        {step===4&&<div className="form-grid">
          <Field className="span-2" label="Nombre completo" error={err("adminName")}><Input {...register("adminName")} placeholder="Juan Pérez" autoComplete="off"/></Field>
          <Field className="span-2" label="Correo electrónico" error={err("adminEmail")}><Input type="email" {...register("adminEmail")} placeholder="admin@restaurante.com" autoComplete="off"/></Field>
          <Field className="span-2" label="Contraseña" error={err("adminPassword")}><Input type="password" {...register("adminPassword")} placeholder="Mínimo 8 caracteres" autoComplete="new-password"/></Field>
        </div>}
      </section>

      <div className="wizard-footer">
        <div>{step>0&&<button type="button" className="button ghost" onClick={()=>setStep(s=>s-1)} disabled={save.isPending}><Icon name="chevronLeft" size={16}/>Atrás</button>}</div>
        <div>{step<lastStep
          ?<button className="button primary" type="submit">Siguiente<Icon name="chevron" size={16}/></button>
          :<button className="button primary" type="submit" disabled={save.isPending}><Icon name="check" size={16}/>{save.isPending?"Guardando…":"Guardar"}</button>}</div>
      </div>
    </form>
  </div>}
  </>;
}
