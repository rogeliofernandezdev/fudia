"use client";
import {useState} from "react";
import {useMutation,useQuery} from "@tanstack/react-query";
import {Icon,IconName} from "@/design-system/icons";
import {LocationMap} from "@/design-system/location-map";
import {Input,PageHeader,Select} from "@/design-system/page-header";
import {useFeedback} from "@/providers/feedback-provider";

type Country={code:string;name:string;defaultCurrency:string};
type Currency={code:string;name:string;symbol:string;decimals:number};
type Context={countryOptions:Country[];currencyOptions:Currency[]};

type Draft={
  legalName:string;tradeName:string;taxId:string;timezone:string;
  country:string;currency:string;currencyPosition:"before"|"after";taxName:string;taxRate:string;taxIncluded:boolean;
  locationName:string;locationCode:string;address:string;locationPhone:string;locationHours:string;latitude:string;longitude:string;
  adminName:string;adminEmail:string;adminPassword:string;
};

const blank:Draft={legalName:"",tradeName:"",taxId:"",timezone:"America/Lima",country:"PE",currency:"PEN",currencyPosition:"before",taxName:"IGV",taxRate:"18",taxIncluded:false,locationName:"",locationCode:"",address:"",locationPhone:"",locationHours:"",latitude:"",longitude:"",adminName:"",adminEmail:"",adminPassword:""};

const steps:Array<{key:string;title:string;icon:IconName;desc:string}>=[
  {key:"empresa",title:"Empresa",icon:"store",desc:"Razón social, RUC y zona horaria"},
  {key:"fiscal",title:"Fiscal y moneda",icon:"receipt",desc:"País, moneda e impuesto"},
  {key:"local",title:"Primer local",icon:"box",desc:"Dirección, teléfono y horario"},
  {key:"admin",title:"Administrador",icon:"users",desc:"Usuario y contraseña del admin"},
];

async function api<T>(path:string,init?:RequestInit):Promise<T>{const r=await fetch(path,{...init,headers:{"Content-Type":"application/json",...init?.headers}});const b=await r.json();if(!r.ok)throw new Error(b.message??"No pudimos registrar la empresa.");return b}

export function PlatformOnboardingPage(){
  const{notify}=useFeedback();
  const[draft,setDraft]=useState<Draft>(blank);
  const[step,setStep]=useState(0);
  const context=useQuery<Context>({
    queryKey:["platform-onboarding-context"],
    queryFn:async()=>{
      const r=await fetch("/api/admin/settings");
      const data=await r.json();
      if(!r.ok)throw new Error(data.message??"No pudimos cargar los catálogos.");
      return {countryOptions:data.countryOptions??[],currencyOptions:data.currencyOptions??[]};
    },
  });
  const ctx=context.data;
  const loadError=context.error instanceof Error?context.error.message:"";

  const save=useMutation({
    mutationFn:()=>{const payload={...draft,locationCode:draft.locationCode||draft.locationName.toUpperCase().replace(/[^A-Z0-9]/g,"").slice(0,8).padStart(3,"L"),taxRate:String(Number(draft.taxRate)/100),latitude:draft.latitude?Number(draft.latitude):null,longitude:draft.longitude?Number(draft.longitude):null};return api<{message:string}>("/api/platform/organizations",{method:"POST",body:JSON.stringify(payload)})},
    onSuccess:()=>{notify({tone:"success",title:"Empresa registrada",message:"La empresa, el local y el administrador quedaron listos."});setDraft(blank);setStep(0)},
    onError:e=>notify({tone:"danger",title:"No se pudo registrar",message:e.message}),
  });

  function set<K extends keyof Draft>(k:K,v:Draft[K]){setDraft(current=>({...current,[k]:v}))}
  function selectCountry(code:string){const c=ctx?.countryOptions.find(x=>x.code===code);if(c)set("currency",c.defaultCurrency);set("country",code)}
  const isLast=step===steps.length-1;
  const canNext=step<steps.length-1;
  const canPrev=step>0;

  return <><PageHeader eyebrow="PLATAFORMA" title="Registrar empresa" description="Completa los 4 pasos para crear una nueva empresa con su primer local y administrador."/>
  {loadError?<div className="catalog-state error"><span><Icon name="alert"/></span><b>No pudimos cargar los catálogos</b><p>{loadError}</p><button className="button secondary" onClick={()=>void context.refetch()}>Reintentar</button></div>:
  <div className="onboarding-wizard">
    <nav className="wizard-steps" aria-label="Pasos del registro">
      {steps.map((s,i)=><button key={s.key} type="button" className={`wizard-step${i===step?" active":""}${i<step?" done":""}`} onClick={()=>setStep(i)} aria-current={i===step?"step":undefined}>
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
        <header><span className="modal-title-icon"><Icon name="receipt" size={18}/></span><div><small>PASO 2 · FISCAL</small><h2>Perfil fiscal del primer local</h2></div></header>
        <div className="form-grid">
          <label>País<Select required value={draft.country} onChange={e=>selectCountry(e.target.value)}>{ctx?.countryOptions.map(c=><option key={c.code} value={c.code}>{c.name} ({c.code})</option>)}</Select></label>
          <label>Moneda<Select required value={draft.currency} onChange={e=>set("currency",e.target.value)}>{ctx?.currencyOptions.map(c=><option key={c.code} value={c.code}>{c.code} — {c.name}</option>)}</Select></label>
          <label>Nombre del impuesto<Input required maxLength={30} value={draft.taxName} onChange={e=>set("taxName",e.target.value)} placeholder="IGV"/></label>
          <label>Porcentaje %<Input required type="number" min="0" max="100" step="0.0001" value={draft.taxRate} onChange={e=>set("taxRate",e.target.value)} placeholder="18"/></label>
          <label>Posición del símbolo<Select value={draft.currencyPosition} onChange={e=>set("currencyPosition",e.target.value as "before"|"after")}><option value="before">Antes del monto</option><option value="after">Después del monto</option></Select></label>
          <label className="switch-row compact"><input type="checkbox" checked={draft.taxIncluded} onChange={e=>set("taxIncluded",e.target.checked)}/><span/><b>Impuesto incluido en el precio</b></label>
        </div>
      </section>}

      {step===2&&<section className="panel management">
        <header><span className="modal-title-icon"><Icon name="box" size={18}/></span><div><small>PASO 3 · LOCAL</small><h2>Sede inicial</h2></div></header>
        <div className="form-grid">
          <label className="span-2">Nombre del local<Input required value={draft.locationName} onChange={e=>set("locationName",e.target.value)} placeholder="Sede principal"/></label>
          <label>Teléfono<Input value={draft.locationPhone} onChange={e=>set("locationPhone",e.target.value)} placeholder="+51 999 888 777"/></label>
          <label>Horario de atención<Input value={draft.locationHours} onChange={e=>set("locationHours",e.target.value)} placeholder="Lun-Dom 12:00-23:00"/></label>
          <div className="span-2"><LocationMap latitude={draft.latitude?Number(draft.latitude):null} longitude={draft.longitude?Number(draft.longitude):null} address={draft.address} onAddressChange={a=>set("address",a)} onChange={c=>{set("latitude",c.lat.toFixed(7));set("longitude",c.lng.toFixed(7))}}/></div>
        </div>
      </section>}

      {step===3&&<section className="panel management">
        <header><span className="modal-title-icon"><Icon name="users" size={18}/></span><div><small>PASO 4 · ADMINISTRADOR</small><h2>Administrador de la empresa</h2></div></header>
        <div className="form-grid">
          <label className="span-2">Nombre completo<Input required value={draft.adminName} onChange={e=>set("adminName",e.target.value)} placeholder="Juan Pérez"/></label>
          <label className="span-2">Correo electrónico<Input required type="email" value={draft.adminEmail} onChange={e=>set("adminEmail",e.target.value)} placeholder="admin@restaurante.com"/></label>
          <label className="span-2">Contraseña<Input required type="password" minLength={8} value={draft.adminPassword} onChange={e=>set("adminPassword",e.target.value)} placeholder="Mínimo 8 caracteres"/></label>
        </div>
      </section>}

      <div className="wizard-footer">
        <div>{canPrev&&<button type="button" className="button ghost" onClick={()=>setStep(s=>s-1)}><Icon name="chevronLeft" size={16}/>Atrás</button>}</div>
        <div>{canNext&&<button className="button primary" type="submit">Siguiente<Icon name="chevron" size={16}/></button>}{isLast&&<button className="button primary" type="submit" disabled={save.isPending}>{save.isPending?"Guardando…":"Guardar"}</button>}</div>
      </div>
    </form>
  </div>}
  </>;
}
