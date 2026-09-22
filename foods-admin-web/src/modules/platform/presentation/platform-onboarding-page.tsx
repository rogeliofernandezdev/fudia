"use client";
import "./platform-onboarding.css";
import Link from "next/link";
import {useState} from "react";
import {useMutation,useQuery} from "@tanstack/react-query";
import {Icon,IconName} from "@/design-system/icons";
import {LocationMap} from "@/design-system/location-map";
import {Input,PageHeader,Select} from "@/design-system/page-header";
import {useFeedback} from "@/providers/feedback-provider";
import type {PlatformOnboardingDraft} from "../domain/types";
import {createPlatformOrganization,getPlatformOnboardingContext} from "../infrastructure/platform-api";

const blank:PlatformOnboardingDraft={
 legalName:"",tradeName:"",taxId:"",timezone:"America/Lima",
 planId:"",billingCycle:"monthly",termsAccepted:false,
 country:"PE",currency:"PEN",currencyPosition:"before",taxName:"IGV",taxRate:"18",taxIncluded:false,
 locationName:"",locationCode:"",address:"",locationPhone:"",locationHours:"",latitude:"",longitude:"",
 adminName:"",adminEmail:"",adminPassword:"",
};

const steps:Array<{key:string;title:string;icon:IconName;desc:string}>=[
  {key:"empresa",title:"Empresa",icon:"store",desc:"Razón social, RUC y zona horaria"},
  {key:"plan",title:"Plan y contrato",icon:"settings",desc:"Suscripción, ciclo y condiciones"},
  {key:"fiscal",title:"Fiscal y moneda",icon:"receipt",desc:"País, moneda e impuesto"},
  {key:"local",title:"Primer local",icon:"box",desc:"Dirección, teléfono y horario"},
  {key:"admin",title:"Administrador",icon:"users",desc:"Responsable principal de la organización"},
];

export function PlatformOnboardingPage(){
  const{notify}=useFeedback();
  const[draft,setDraft]=useState<PlatformOnboardingDraft>(blank);
  const[step,setStep]=useState(0);
  const context=useQuery({queryKey:["platform-onboarding-context"],queryFn:getPlatformOnboardingContext});
  const ctx=context.data;
  const loadError=context.error instanceof Error?context.error.message:"";
  const effectivePlanId=draft.planId||ctx?.plans[0]?.id||"";
  const selectedPlan=ctx?.plans.find(plan=>plan.id===effectivePlanId)??null;

  const save=useMutation({
    mutationFn:()=>createPlatformOrganization({...draft,planId:effectivePlanId}),
    onSuccess:()=>{notify({tone:"success",title:"Empresa registrada",message:"La empresa, su suscripción, el local y el Administrador de empresa quedaron listos."});setDraft(blank);setStep(0)},
    onError:e=>notify({tone:"danger",title:"No se pudo registrar",message:e.message}),
  });

  function set<K extends keyof PlatformOnboardingDraft>(k:K,v:PlatformOnboardingDraft[K]){setDraft(current=>({...current,[k]:v}))}
  function selectCountry(code:string){const c=ctx?.countryOptions.find(x=>x.code===code);if(c)set("currency",c.defaultCurrency);set("country",code)}
  const isLast=step===steps.length-1;
  const canNext=step<steps.length-1;
  const canPrev=step>0;

  return <><PageHeader eyebrow="PLATAFORMA" title="Registrar empresa" description="Alta única y transaccional: empresa, contrato, fiscalidad, primer local y Administrador de empresa."/>
  {loadError?<div className="catalog-state error"><span><Icon name="alert"/></span><b>No pudimos cargar los catálogos</b><p>{loadError}</p><button className="button secondary" onClick={()=>void context.refetch()}>Reintentar</button></div>:
  context.isLoading?<div className="catalog-state"><b>Cargando configuración…</b></div>:
  !ctx?.plans.length?<div className="catalog-state"><span><Icon name="settings"/></span><b>Primero crea un plan comercial</b><p>El alta de una empresa exige un plan activo con precio, límites, condiciones y módulos definidos.</p><Link className="button primary" href="/platform/plans"><Icon name="plus" size={16}/>Crear plan</Link></div>:
  <div className="onboarding-wizard">
    <nav className="wizard-steps" aria-label="Pasos del registro">
      {steps.map((s,i)=><button key={s.key} type="button" className={"wizard-step"+(i===step?" active":"")+(i<step?" done":"")} onClick={()=>setStep(i)} aria-current={i===step?"step":undefined}>
        <span className="wizard-step-icon">{i<step?<Icon name="check" size={16}/>:<Icon name={s.icon} size={16}/>}</span>
        <span className="wizard-step-label"><b>{s.title}</b><small>{s.desc}</small></span>
      </button>)}
    </nav>

    <form className="onboarding-form" onSubmit={e=>{e.preventDefault();if(isLast)save.mutate();else setStep(s=>s+1)}}>
      <div className="wizard-progress">Paso {step+1} de {steps.length}</div>
      {step===0&&<section className="panel management">
        <header><span className="modal-title-icon"><Icon name="store" size={18}/></span><div><small>PASO 1 · EMPRESA</small><h2>Datos legales</h2></div></header>
        <div className="form-grid">
          <label className="span-2">Razón social<Input required maxLength={180} value={draft.legalName} onChange={e=>set("legalName",e.target.value)} placeholder="Restaurante Perú SAC"/></label>
          <label>Nombre comercial<Input required maxLength={180} value={draft.tradeName} onChange={e=>set("tradeName",e.target.value)} placeholder="Mi restaurante"/></label>
          <label>Identificación fiscal<Input required minLength={6} maxLength={32} value={draft.taxId} onChange={e=>set("taxId",e.target.value)} placeholder="RUC 20512345678"/></label>
          <label className="span-2">Zona horaria<Input required value={draft.timezone} onChange={e=>set("timezone",e.target.value)} placeholder="America/Lima"/></label>
        </div>
      </section>}

      {step===1&&<section className="panel management">
        <header><span className="modal-title-icon"><Icon name="settings" size={18}/></span><div><small>PASO 2 · PLAN Y CONTRATO</small><h2>Suscripción SaaS</h2></div></header>
        <div className="form-grid">
          <label className="span-2">Plan contratado<Select required value={effectivePlanId} onChange={e=>set("planId",e.target.value)}>{ctx.plans.map(plan=><option key={plan.id} value={plan.id}>{plan.name+" · "+plan.code}</option>)}</Select></label>
          <label>Ciclo de facturación<Select value={draft.billingCycle} onChange={e=>set("billingCycle",e.target.value as "monthly"|"annual")}><option value="monthly">Mensual</option><option value="annual">Anual</option></Select></label>
          <label>Precio contratado<Input readOnly value={selectedPlan?(selectedPlan.currency+" "+Number(draft.billingCycle==="annual"?selectedPlan.annualPrice:selectedPlan.monthlyPrice).toFixed(2)):""}/></label>
        </div>
        {selectedPlan&&<div className="onboarding-plan-summary">
          <div><small>PRUEBA GRATUITA</small><b>{selectedPlan.trialDays?selectedPlan.trialDays+" días":"No incluida"}</b></div>
          <div><small>LÍMITE DE LOCALES</small><b>{selectedPlan.maxLocations??"Sin límite"}</b></div>
          <div><small>LÍMITE DE USUARIOS</small><b>{selectedPlan.maxUsers??"Sin límite"}</b></div>
          <div><small>MÓDULOS</small><b>{selectedPlan.moduleKeys.length}</b></div>
          <div><small>CONDICIONES</small><b>{selectedPlan.termsVersion}</b></div>
        </div>}
        <label className="onboarding-terms"><input required type="checkbox" checked={draft.termsAccepted} onChange={e=>set("termsAccepted",e.target.checked)}/><span><Icon name="check" size={13}/></span><div><b>Condiciones aceptadas por el cliente</b><small>Confirma que la contratación y la versión {selectedPlan?.termsVersion??"vigente"} fueron aceptadas antes de crear la empresa.</small></div></label>
      </section>}

      {step===2&&<section className="panel management">
        <header><span className="modal-title-icon"><Icon name="receipt" size={18}/></span><div><small>PASO 3 · FISCAL</small><h2>Perfil fiscal del primer local</h2></div></header>
        <div className="form-grid">
          <label>País<Select required value={draft.country} onChange={e=>selectCountry(e.target.value)}>{ctx.countryOptions.map(c=><option key={c.code} value={c.code}>{c.name+" ("+c.code+")"}</option>)}</Select></label>
          <label>Moneda<Select required value={draft.currency} onChange={e=>set("currency",e.target.value)}>{ctx.currencyOptions.map(c=><option key={c.code} value={c.code}>{c.code+" — "+c.name}</option>)}</Select></label>
          <label>Nombre del impuesto<Input required maxLength={30} value={draft.taxName} onChange={e=>set("taxName",e.target.value)} placeholder="IGV"/></label>
          <label>Porcentaje %<Input required type="number" min="0" max="100" step="0.0001" value={draft.taxRate} onChange={e=>set("taxRate",e.target.value)} placeholder="18"/></label>
          <label>Posición del símbolo<Select value={draft.currencyPosition} onChange={e=>set("currencyPosition",e.target.value as "before"|"after")}><option value="before">Antes del monto</option><option value="after">Después del monto</option></Select></label>
          <label className="switch-row compact"><input type="checkbox" checked={draft.taxIncluded} onChange={e=>set("taxIncluded",e.target.checked)}/><span/><b>Impuesto incluido en el precio</b></label>
        </div>
      </section>}

      {step===3&&<section className="panel management">
        <header><span className="modal-title-icon"><Icon name="box" size={18}/></span><div><small>PASO 4 · LOCAL</small><h2>Sede inicial</h2></div></header>
        <div className="form-grid">
          <label className="span-2">Nombre del local<Input required value={draft.locationName} onChange={e=>set("locationName",e.target.value)} placeholder="Sede principal"/></label>
          <label>Teléfono<Input value={draft.locationPhone} onChange={e=>set("locationPhone",e.target.value)} placeholder="+51 999 888 777"/></label>
          <label>Horario de atención<Input value={draft.locationHours} onChange={e=>set("locationHours",e.target.value)} placeholder="Lun-Dom 12:00-23:00"/></label>
          <div className="span-2"><LocationMap latitude={draft.latitude?Number(draft.latitude):null} longitude={draft.longitude?Number(draft.longitude):null} address={draft.address} onAddressChange={a=>set("address",a)} onChange={c=>{set("latitude",c.lat.toFixed(7));set("longitude",c.lng.toFixed(7))}}/></div>
        </div>
      </section>}

      {step===4&&<section className="panel management">
        <header><span className="modal-title-icon"><Icon name="users" size={18}/></span><div><small>PASO 5 · ADMINISTRADOR DE EMPRESA</small><h2>Responsable principal de la empresa</h2></div></header>
        <div className="form-grid">
          <label className="span-2">Nombre completo<Input required value={draft.adminName} onChange={e=>set("adminName",e.target.value)} placeholder="Juan Pérez"/></label>
          <label className="span-2">Correo electrónico<Input required type="email" value={draft.adminEmail} onChange={e=>set("adminEmail",e.target.value)} placeholder="admin@restaurante.com"/></label>
          <label className="span-2">Contraseña<Input required type="password" minLength={8} value={draft.adminPassword} onChange={e=>set("adminPassword",e.target.value)} placeholder="Mínimo 8 caracteres"/></label>
        </div>
      </section>}

      <div className="wizard-footer">
        <div>{canPrev&&<button type="button" className="button ghost" onClick={()=>setStep(s=>s-1)}><Icon name="chevronLeft" size={16}/>Atrás</button>}</div>
        <div>{canNext&&<button className="button primary" type="submit">Siguiente<Icon name="chevron" size={16}/></button>}{isLast&&<button className="button primary" type="submit" disabled={save.isPending||!draft.termsAccepted}>{save.isPending?"Guardando…":"Guardar"}</button>}</div>
      </div>
    </form>
  </div>}
  </>;
}
