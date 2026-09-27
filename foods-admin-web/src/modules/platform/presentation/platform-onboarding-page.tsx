"use client";
import "./platform-onboarding.css";
import Link from "next/link";
import {useMemo,useRef,useState} from "react";
import {useMutation,useQuery} from "@tanstack/react-query";
import type {ZodType} from "zod";
import {Icon,IconName} from "@/design-system/icons";
import {LocationMap} from "@/design-system/location-map";
import {ApiClientError} from "@/shared/api/client";
import {Input,PageHeader,Select} from "@/design-system/page-header";
import {useFeedback} from "@/providers/feedback-provider";
import type {PlatformOnboardingDraft} from "../domain/types";
import {adminSchema,companySchema,fiscalSchema,locationSchema,planSchema} from "../domain/onboarding-schemas";
import {createPlatformOrganization,getPlatformOnboardingContext} from "../infrastructure/platform-api";
import {OnboardingFormField as Field} from "./onboarding-form-field";
import {PlatformOnboardingSkeleton} from "./platform-skeletons";

type Draft=PlatformOnboardingDraft;
type FieldKey=keyof Draft;
type Errors=Partial<Record<FieldKey,string>>;

const blank:Draft={
 legalName:"",tradeName:"",taxId:"",timezone:"America/Lima",
 planId:"",billingCycle:"monthly",termsAccepted:false,
 country:"PE",currency:"PEN",currencyPosition:"before",taxName:"IGV",taxRate:"18",taxIncluded:false,
 locationName:"",locationCode:"",address:"",locationPhone:"",locationHours:"",latitude:"",longitude:"",
 adminName:"",adminEmail:"",adminPassword:"",
};

type Step={key:string;title:string;icon:IconName;desc:string;heading:string;schema?:ZodType};
const steps:Step[]=[
  {key:"empresa",title:"Empresa",icon:"store",desc:"Datos legales",heading:"Datos legales de la empresa",schema:companySchema},
  {key:"plan",title:"Plan",icon:"settings",desc:"Suscripción y condiciones",heading:"Suscripción contratada",schema:planSchema},
  {key:"fiscal",title:"Fiscal",icon:"receipt",desc:"Moneda e impuesto",heading:"Perfil fiscal del primer local",schema:fiscalSchema},
  {key:"local",title:"Local",icon:"box",desc:"Sede inicial",heading:"Sede inicial",schema:locationSchema},
  {key:"admin",title:"Administrador",icon:"users",desc:"Responsable principal",heading:"Administrador de empresa",schema:adminSchema},
  {key:"revision",title:"Revisión",icon:"check",desc:"Confirmar y registrar",heading:"Revisa antes de registrar"},
];
const reviewIndex=steps.length-1;

function listTimezones(current:string){
  let zones:string[]=[];
  try{zones=Intl.supportedValuesOf("timeZone")}catch{zones=[]}
  return zones.includes(current)?zones:[current,...zones];
}

function validate(step:Step,draft:Draft):Errors{
  if(!step.schema)return{};
  const result=step.schema.safeParse(draft);
  if(result.success)return{};
  const errors:Errors={};
  for(const issue of result.error.issues){const key=issue.path[0] as FieldKey;if(key&&!errors[key])errors[key]=issue.message}
  return errors;
}

export function PlatformOnboardingPage(){
  const{notify}=useFeedback();
  const[draft,setDraft]=useState<Draft>(blank);
  const[step,setStep]=useState(0);
  const[reached,setReached]=useState(0);
  const[errors,setErrors]=useState<Errors>({});
  const[showPassword,setShowPassword]=useState(false);
  const formRef=useRef<HTMLFormElement>(null);
  const context=useQuery({queryKey:["platform-onboarding-context"],queryFn:getPlatformOnboardingContext});
  const ctx=context.data;
  const loadError=context.error instanceof Error?context.error.message:"";
  const effectivePlanId=draft.planId||ctx?.plans[0]?.id||"";
  const effectiveDraft={...draft,planId:effectivePlanId};
  const selectedPlan=ctx?.plans.find(plan=>plan.id===effectivePlanId)??null;
  const timezones=useMemo(()=>listTimezones(draft.timezone),[draft.timezone]);
  const price=selectedPlan?Number(draft.billingCycle==="annual"?selectedPlan.annualPrice:selectedPlan.monthlyPrice).toFixed(2):"";
  const countryName=ctx?.countryOptions.find(c=>c.code===draft.country)?.name??draft.country;

  const save=useMutation({
    mutationFn:()=>createPlatformOrganization(effectiveDraft),
    onSuccess:()=>{notify({tone:"success",title:"Empresa registrada",message:"La empresa, su suscripción, el local y el Administrador de empresa quedaron listos."});setDraft(blank);setStep(0);setReached(0);setErrors({})},
    onError:e=>{const message=e instanceof ApiClientError&&e.status>=500&&e.correlationId?`${e.message} Código de seguimiento: ${e.correlationId}`:e.message;notify({tone:"danger",title:"No se pudo registrar",message})},
  });

  function set<K extends FieldKey>(k:K,v:Draft[K]){
    setDraft(current=>({...current,[k]:v}));
    if(errors[k])setErrors(current=>{const next={...current};delete next[k];return next});
  }
  function selectCountry(code:string){const c=ctx?.countryOptions.find(x=>x.code===code);if(c)set("currency",c.defaultCurrency);set("country",code)}
  function focusFirstError(found:Errors){
    const first=Object.keys(found)[0];
    if(first)requestAnimationFrame(()=>formRef.current?.querySelector<HTMLElement>(`[name="${first}"]`)?.focus());
  }
  function goTo(index:number){
    if(index>reached||index===step)return;
    setErrors({});setStep(index);
  }
  function next(){
    const found=validate(steps[step],effectiveDraft);
    if(Object.keys(found).length){setErrors(found);focusFirstError(found);return}
    setErrors({});
    const target=step+1;
    setStep(target);setReached(r=>Math.max(r,target));
  }
  function submit(){
    for(let i=0;i<reviewIndex;i++){
      const found=validate(steps[i],effectiveDraft);
      if(Object.keys(found).length){setStep(i);setErrors(found);focusFirstError(found);return}
    }
    save.mutate();
  }
  const err=(k:FieldKey)=>errors[k];
  const invalid=(k:FieldKey)=>errors[k]?true:undefined;
  const current=steps[step];

  return <><PageHeader eyebrow="PLATAFORMA" title="Registrar empresa" description="Alta única y transaccional: empresa, contrato, fiscalidad, primer local y Administrador de empresa."/>
  {loadError?<div className="catalog-state error"><span><Icon name="alert"/></span><b>No pudimos cargar los catálogos</b><p>{loadError}</p><button className="button secondary" onClick={()=>void context.refetch()}>Reintentar</button></div>:
  context.isLoading?<PlatformOnboardingSkeleton stepCount={steps.length}/>:
  !ctx?.plans.length?<div className="catalog-state"><span><Icon name="settings"/></span><b>Primero crea un plan comercial</b><p>El alta de una empresa exige un plan activo con precio, límites, condiciones y módulos definidos.</p><Link className="button primary" href="/platform/plans"><Icon name="plus" size={16}/>Crear plan</Link></div>:
  <div className="onb-layout">
    <nav className="onb-steps" aria-label="Pasos del registro">
      <ol>
        {steps.map((s,i)=>{
          const state=i===step?"active":i<reached?"done":i===reached?"open":"locked";
          return <li key={s.key}>
            <button type="button" className={"onb-step "+state} onClick={()=>goTo(i)} disabled={i>reached} aria-current={i===step?"step":undefined}>
              <span className="onb-step-icon">{state==="done"?<Icon name="check" size={16}/>:<Icon name={s.icon} size={16}/>}</span>
              <span className="onb-step-copy"><b>{s.title}</b><small>{s.desc}</small></span>
            </button>
          </li>;
        })}
      </ol>
    </nav>

    <form ref={formRef} className="onb-card" noValidate onSubmit={e=>{e.preventDefault();if(step===reviewIndex)submit();else next()}}>
      <header className="onb-card-head">
        <span className="onb-card-icon"><Icon name={current.icon} size={18}/></span>
        <h2>{current.heading}</h2>
      </header>

      <div className="onb-card-body">
      {step===0&&<div className="onb-grid">
        <Field wide label="Razón social" error={err("legalName")}><Input name="legalName" aria-invalid={invalid("legalName")} maxLength={180} value={draft.legalName} onChange={e=>set("legalName",e.target.value)} placeholder="Restaurante Perú SAC" autoComplete="organization"/></Field>
        <Field label="Nombre comercial" description="Se muestra al personal y en comprobantes." error={err("tradeName")}><Input name="tradeName" aria-invalid={invalid("tradeName")} maxLength={180} value={draft.tradeName} onChange={e=>set("tradeName",e.target.value)} placeholder="Mi restaurante"/></Field>
        <Field label="Identificación fiscal" description="RUC u otro identificador tributario." error={err("taxId")}><Input name="taxId" aria-invalid={invalid("taxId")} maxLength={32} inputMode="numeric" value={draft.taxId} onChange={e=>set("taxId",e.target.value)} placeholder="20512345678"/></Field>
        <Field wide label="Zona horaria" description="Define cortes de caja, reportes y horarios." error={err("timezone")}><Select name="timezone" aria-invalid={invalid("timezone")} value={draft.timezone} onChange={e=>set("timezone",e.target.value)}>{timezones.map(z=><option key={z} value={z}>{z.replaceAll("_"," ")}</option>)}</Select></Field>
      </div>}

      {step===1&&<>
        <div className="onb-grid">
          <Field wide label="Plan contratado" error={err("planId")}><Select name="planId" aria-invalid={invalid("planId")} value={effectivePlanId} onChange={e=>set("planId",e.target.value)}>{ctx.plans.map(plan=><option key={plan.id} value={plan.id}>{plan.name+" · "+plan.code}</option>)}</Select></Field>
          <div className="onb-field wide" role="radiogroup" aria-label="Ciclo de facturación">
            <span className="onb-field-label">Ciclo de facturación</span>
            <div className="onb-segment">
              {(["monthly","annual"] as const).map(cycle=><button key={cycle} type="button" role="radio" aria-checked={draft.billingCycle===cycle} className={draft.billingCycle===cycle?"active":""} onClick={()=>set("billingCycle",cycle)}>
                <b>{cycle==="monthly"?"Mensual":"Anual"}</b>
                {selectedPlan&&<small>{selectedPlan.currency} {Number(cycle==="annual"?selectedPlan.annualPrice:selectedPlan.monthlyPrice).toFixed(2)}</small>}
              </button>)}
            </div>
          </div>
        </div>
        {selectedPlan&&<dl className="onb-facts">
          <div><dt>Prueba gratuita</dt><dd>{selectedPlan.trialDays?selectedPlan.trialDays+" días":"No incluida"}</dd></div>
          <div><dt>Locales</dt><dd>{selectedPlan.maxLocations??"Sin límite"}</dd></div>
          <div><dt>Usuarios</dt><dd>{selectedPlan.maxUsers??"Sin límite"}</dd></div>
          <div><dt>Módulos</dt><dd>{selectedPlan.moduleKeys.length}</dd></div>
        </dl>}
        <label className={"onb-terms"+(err("termsAccepted")?" has-error":"")}>
          <input name="termsAccepted" type="checkbox" checked={draft.termsAccepted} onChange={e=>set("termsAccepted",e.target.checked)}/>
          <span className="onb-check" aria-hidden="true"><Icon name="check" size={14}/></span>
          <span><b>El cliente aceptó las condiciones {selectedPlan?.termsVersion?`(versión ${selectedPlan.termsVersion})`:"vigentes"}</b><small>Confírmalo antes de crear la empresa.</small>{err("termsAccepted")&&<small className="onb-field-error" role="alert">{err("termsAccepted")}</small>}</span>
        </label>
      </>}

      {step===2&&<div className="onb-grid">
        <Field label="País" error={err("country")}><Select name="country" value={draft.country} onChange={e=>selectCountry(e.target.value)}>{ctx.countryOptions.map(c=><option key={c.code} value={c.code}>{c.name}</option>)}</Select></Field>
        <Field label="Moneda" error={err("currency")}><Select name="currency" value={draft.currency} onChange={e=>set("currency",e.target.value)}>{ctx.currencyOptions.map(c=><option key={c.code} value={c.code}>{c.code+" — "+c.name}</option>)}</Select></Field>
        <Field label="Impuesto" error={err("taxName")}><Input name="taxName" aria-invalid={invalid("taxName")} maxLength={30} value={draft.taxName} onChange={e=>set("taxName",e.target.value)} placeholder="IGV"/></Field>
        <Field label="Tasa (%)" error={err("taxRate")}><Input name="taxRate" aria-invalid={invalid("taxRate")} type="number" inputMode="decimal" min="0" max="100" step="0.0001" value={draft.taxRate} onChange={e=>set("taxRate",e.target.value)} placeholder="18"/></Field>
        <Field label="Símbolo de moneda"><Select name="currencyPosition" value={draft.currencyPosition} onChange={e=>set("currencyPosition",e.target.value as "before"|"after")}><option value="before">Antes del monto (S/ 10.00)</option><option value="after">Después del monto (10.00 S/)</option></Select></Field>
        <label className="onb-switch"><input type="checkbox" checked={draft.taxIncluded} onChange={e=>set("taxIncluded",e.target.checked)}/><span aria-hidden="true"/><span><b>Precios con impuesto incluido</b><small>Los precios de la carta ya contienen el impuesto.</small></span></label>
      </div>}

      {step===3&&<div className="onb-grid">
        <Field wide label="Nombre del local" error={err("locationName")}><Input name="locationName" aria-invalid={invalid("locationName")} value={draft.locationName} onChange={e=>set("locationName",e.target.value)} placeholder="Sede principal"/></Field>
        <Field optional label="Teléfono"><Input name="locationPhone" type="tel" inputMode="tel" value={draft.locationPhone} onChange={e=>set("locationPhone",e.target.value)} placeholder="+51 999 888 777"/></Field>
        <Field optional label="Horario de atención"><Input name="locationHours" value={draft.locationHours} onChange={e=>set("locationHours",e.target.value)} placeholder="Lun-Dom 12:00-23:00"/></Field>
        <div className={"onb-field wide"+(err("address")?" has-error":"")}>
          <span className="onb-field-label">Dirección y ubicación</span>
          <LocationMap latitude={draft.latitude?Number(draft.latitude):null} longitude={draft.longitude?Number(draft.longitude):null} address={draft.address} onAddressChange={a=>set("address",a)} onChange={c=>{set("latitude",c.lat.toFixed(7));set("longitude",c.lng.toFixed(7))}}/>
          {err("address")&&<small className="onb-field-error" role="alert">{err("address")}</small>}
        </div>
      </div>}

      {step===4&&<div className="onb-grid">
        <Field wide label="Nombre completo" error={err("adminName")}><Input name="adminName" aria-invalid={invalid("adminName")} value={draft.adminName} onChange={e=>set("adminName",e.target.value)} placeholder="Juan Pérez" autoComplete="off"/></Field>
        <Field wide label="Correo electrónico" description="Será su usuario de acceso." error={err("adminEmail")}><Input name="adminEmail" aria-invalid={invalid("adminEmail")} type="email" inputMode="email" value={draft.adminEmail} onChange={e=>set("adminEmail",e.target.value)} placeholder="admin@restaurante.com" autoComplete="off"/></Field>
        <Field wide label="Contraseña inicial" description="Mínimo 8 caracteres. Compártela por un canal seguro." error={err("adminPassword")}>
          <span className="onb-password">
            <Input name="adminPassword" aria-invalid={invalid("adminPassword")} type={showPassword?"text":"password"} value={draft.adminPassword} onChange={e=>set("adminPassword",e.target.value)} autoComplete="new-password"/>
            <button type="button" className="onb-password-toggle" onClick={e=>{e.preventDefault();setShowPassword(v=>!v)}} aria-label={showPassword?"Ocultar contraseña":"Mostrar contraseña"} aria-pressed={showPassword}><Icon name={showPassword?"lock":"eye"} size={18}/></button>
          </span>
        </Field>
      </div>}

      {step===reviewIndex&&<div className="onb-review">
        <ReviewBlock title="Empresa" onEdit={()=>goTo(0)} rows={[["Razón social",draft.legalName],["Nombre comercial",draft.tradeName],["Identificación fiscal",draft.taxId],["Zona horaria",draft.timezone.replaceAll("_"," ")]]}/>
        <ReviewBlock title="Plan" onEdit={()=>goTo(1)} rows={[["Plan",selectedPlan?selectedPlan.name:"—"],["Ciclo",draft.billingCycle==="annual"?"Anual":"Mensual"],["Precio",selectedPlan?`${selectedPlan.currency} ${price}`:"—"],["Condiciones",draft.termsAccepted?"Aceptadas":"Pendientes"]]}/>
        <ReviewBlock title="Fiscal" onEdit={()=>goTo(2)} rows={[["País",countryName],["Moneda",draft.currency],["Impuesto",`${draft.taxName} ${draft.taxRate}%`],["Precios",draft.taxIncluded?"Con impuesto incluido":"Sin impuesto incluido"]]}/>
        <ReviewBlock title="Local" onEdit={()=>goTo(3)} rows={[["Nombre",draft.locationName],["Dirección",draft.address],["Teléfono",draft.locationPhone||"—"],["Horario",draft.locationHours||"—"]]}/>
        <ReviewBlock title="Administrador" onEdit={()=>goTo(4)} rows={[["Nombre",draft.adminName],["Correo",draft.adminEmail]]}/>
      </div>}
      </div>

      <footer className="onb-footer">
        {step>0?<button type="button" className="button ghost" onClick={()=>goTo(step-1)} disabled={save.isPending}><Icon name="chevronLeft" size={16}/>Atrás</button>:<span/>}
        {step<reviewIndex
          ?<button className="button primary" type="submit">{step===reviewIndex-1?"Revisar":"Continuar"}<Icon name="chevron" size={16}/></button>
          :<button className="button primary" type="submit" disabled={save.isPending}><Icon name="save" size={16}/>{save.isPending?"Registrando…":"Registrar empresa"}</button>}
      </footer>
    </form>
  </div>}
  </>;
}

function ReviewBlock({title,rows,onEdit}:{title:string;rows:Array<[string,string]>;onEdit:()=>void}){
  return <section className="onb-review-block">
    <header><h3>{title}</h3><button type="button" className="onb-review-edit" onClick={onEdit} aria-label={`Editar ${title}`}><Icon name="edit" size={16}/>Editar</button></header>
    <dl>{rows.map(([label,value])=><div key={label}><dt>{label}</dt><dd>{value||"—"}</dd></div>)}</dl>
  </section>;
}
