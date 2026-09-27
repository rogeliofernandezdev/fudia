"use client";
import "./platform-plans.css";
import {useMemo,useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,Input,PageHeader,Select,Textarea} from "@/design-system";
import {Icon} from "@/design-system/icons";
import {useFeedback} from "@/providers";
import type {PlatformModule,SubscriptionPlan,SubscriptionPlanDraft} from "../domain/types";
import {getPlatformOnboardingContext,listReadyModules,listSubscriptionPlans,saveSubscriptionPlan} from "../infrastructure/platform-api";
import {PlatformPlansSkeleton} from "./platform-skeletons";

const blank:SubscriptionPlanDraft={
 code:"",name:"",description:"",currency:"PEN",monthlyPrice:"",annualPrice:"",trialDays:0,
 maxLocations:"",maxUsers:"",moduleKeys:[],termsVersion:"",active:true,
};

export function PlatformPlansPage(){
 const{notify}=useFeedback();
 const client=useQueryClient();
 const plans=useQuery({queryKey:["platform-plans"],queryFn:listSubscriptionPlans});
 const modules=useQuery({queryKey:["platform-ready-modules"],queryFn:listReadyModules});
 const catalogs=useQuery({queryKey:["platform-onboarding-context"],queryFn:getPlatformOnboardingContext});
 const[draft,setDraft]=useState<SubscriptionPlanDraft|null>(null);
 const[billing,setBilling]=useState<"monthly"|"annual">("monthly");
 const save=useMutation({
  mutationFn:saveSubscriptionPlan,
  onSuccess:()=>{setDraft(null);void client.invalidateQueries({queryKey:["platform-plans"]});void client.invalidateQueries({queryKey:["platform-onboarding-context"]});notify({tone:"success",title:"Plan guardado",message:"Precios, límites y módulos del plan quedaron actualizados."})},
  onError:e=>notify({tone:"danger",title:"No se pudo guardar",message:e.message}),
 });
 const visible=(plans.data?.items??[]).filter(plan=>plan.code!=="legacy");
 const moduleMap=new Map((modules.data??[]).map(module=>[module.key,module]));
 const edit=(plan:SubscriptionPlan)=>setDraft({
  id:plan.id,code:plan.code,name:plan.name,description:plan.description,currency:plan.currency,
  monthlyPrice:plan.monthlyPrice,annualPrice:plan.annualPrice,trialDays:plan.trialDays,
  maxLocations:plan.maxLocations?.toString()??"",maxUsers:plan.maxUsers?.toString()??"",
  moduleKeys:plan.moduleKeys,termsVersion:plan.termsVersion,active:plan.active,
 });

 return <>
  <PageHeader eyebrow="PLATAFORMA" title="Planes SaaS" description="Define la oferta comercial de FUDIA: precio, prueba, límites y módulos de cada nivel." action={<Button icon="plus" onClick={()=>setDraft({...blank})}>Nuevo plan</Button>}/>
  <div className="plan-toolbar">
    <b>Catálogo comercial</b>
    <div className="plan-cycle" role="group" aria-label="Ciclo de facturación">
      <button type="button" className={billing==="monthly"?"active":""} onClick={()=>setBilling("monthly")}>Mensual</button>
      <button type="button" className={billing==="annual"?"active":""} onClick={()=>setBilling("annual")}>Anual<span>ahorra ~2 meses</span></button>
    </div>
  </div>
  {(plans.isLoading||modules.isLoading||catalogs.isLoading)?<PlatformPlansSkeleton/>:
   (plans.isError||modules.isError||catalogs.isError)?<div className="panel plan-state error"><b>No pudimos cargar la configuración de planes.</b><Button kind="secondary" onClick={()=>{void plans.refetch();void modules.refetch();void catalogs.refetch()}}>Reintentar</Button></div>:
   <div className="plan-grid">{visible.length?visible.map(plan=>{
     const recommended=plan.code==="impulso";
     const price=billing==="annual"?Number(plan.annualPrice):Number(plan.monthlyPrice);
     const equivalent=billing==="annual"?price/12:price;
     const savings=billing==="annual"?Math.max(0,Number(plan.monthlyPrice)*12-price):0;
     const included=plan.moduleKeys.map(key=>moduleMap.get(key)).filter((module):module is PlatformModule=>Boolean(module)).sort((a,b)=>a.name.localeCompare(b.name,"es"));
     return <article className={"panel plan-card"+(recommended?" recommended":"")} key={plan.id}>
       <header>
         <div className="plan-heading">
           <span className="plan-icon"><Icon name={plan.code==="emprende"?"store":plan.code==="escala"?"grid":"sales"} size={20}/></span>
           <div><small>{plan.code.toUpperCase()}{recommended&&<em>Recomendado</em>}</small><h2>{plan.name}</h2></div>
         </div>
         <span className={plan.active?"plan-status active":"plan-status"}>{plan.active?"Disponible":"Inactivo"}</span>
       </header>
       <div className="plan-price">
         <div><span>{plan.currency}</span><b>{price.toFixed(2)}</b><em>{billing==="annual"?"/ año":"/ mes"}</em></div>
         <p>{billing==="annual"?"Equivale a "+plan.currency+" "+equivalent.toFixed(2)+" al mes"+(savings>0?" · ahorras "+plan.currency+" "+savings.toFixed(2):""):plan.trialDays?plan.trialDays+" días de prueba gratuita":"Sin periodo de prueba"}</p>
       </div>
       <p className="plan-capacity">
         <span><Icon name="store" size={14}/>{plan.maxLocations??"Sin límite de"} {plan.maxLocations===1?"local":"locales"}</span>
         <span><Icon name="users" size={14}/>{plan.maxUsers??"Sin límite de"} usuarios</span>
       </p>
       <section className="plan-includes">
         <header><h3>Incluye {included.length} módulos</h3></header>
         <ul className="plan-feature-list">{included.map(module=><li key={module.key} title={module.description}><Icon name="check" size={12}/>{module.name}</li>)}</ul>
       </section>
       <footer>
         <div><small>Condiciones</small><b>{plan.termsVersion}</b></div>
         <Button kind={recommended?"primary":"secondary"} icon="edit" onClick={()=>edit(plan)}>Editar plan</Button>
       </footer>
     </article>;
   }):<div className="panel plan-state"><b>No hay planes comerciales.</b><p>Crea el primer plan antes de registrar una empresa nueva.</p><Button icon="plus" onClick={()=>setDraft({...blank})}>Nuevo plan</Button></div>}</div>}
  {draft&&<PlanDialog value={draft} modules={modules.data??[]} currencies={catalogs.data?.currencyOptions??[]} busy={save.isPending} close={()=>setDraft(null)} save={value=>save.mutate(value)}/>}
 </>;
}

function PlanDialog({value:initial,modules,currencies,busy,close,save}:{value:SubscriptionPlanDraft;modules:PlatformModule[];currencies:{code:string;name:string}[];busy:boolean;close:()=>void;save:(value:SubscriptionPlanDraft)=>void}){
 const[value,setValue]=useState(initial);
 const groups=useMemo(()=>Array.from(new Set(modules.map(module=>module.category))),[modules]);
 const toggle=(key:string)=>setValue(current=>({...current,moduleKeys:current.moduleKeys.includes(key)?current.moduleKeys.filter(item=>item!==key):[...current.moduleKeys,key]}));
 const valid=Boolean(value.code.trim().length>=2&&value.name.trim()&&value.monthlyPrice!==""&&value.annualPrice!==""&&value.termsVersion.trim()&&value.moduleKeys.length>0);
 return <div className="modal-backdrop"><section className="crud-modal plan-dialog" role="dialog" aria-modal="true" aria-labelledby="plan-dialog-title"><div className="modal-accent"/><header><span className="modal-title-icon"><Icon name="settings"/></span><div><small>{value.id?"EDITAR PLAN":"NUEVO PLAN"}</small><h2 id="plan-dialog-title">Configuración comercial</h2></div><button aria-label="Cerrar" disabled={busy} onClick={close}><Icon name="close"/></button></header>
  <form onSubmit={e=>{e.preventDefault();if(valid)save(value)}}>
   <div className="plan-form">
    <div className="form-grid">
     <label>Código<Input required minLength={2} maxLength={40} value={value.code} onChange={e=>setValue({...value,code:e.target.value.toLowerCase().replace(/[^a-z0-9_-]/g,"")})} placeholder="ej. crecimiento"/></label>
     <label>Nombre<Input required maxLength={120} value={value.name} onChange={e=>setValue({...value,name:e.target.value})} placeholder="Crecimiento"/></label>
     <label className="span-2">Descripción<Textarea maxLength={500} rows={2} value={value.description} onChange={e=>setValue({...value,description:e.target.value})}/></label>
     <label>Moneda<Select required value={value.currency} onChange={e=>setValue({...value,currency:e.target.value})}>{currencies.map(item=><option key={item.code} value={item.code}>{item.code+" — "+item.name}</option>)}</Select></label>
     <label>Versión de condiciones<Input required maxLength={80} value={value.termsVersion} onChange={e=>setValue({...value,termsVersion:e.target.value})} placeholder="2026-09"/></label>
     <label>Precio mensual<Input required type="number" min="0" step="0.01" value={value.monthlyPrice} onChange={e=>setValue({...value,monthlyPrice:e.target.value})}/></label>
     <label>Precio anual<Input required type="number" min="0" step="0.01" value={value.annualPrice} onChange={e=>setValue({...value,annualPrice:e.target.value})}/></label>
     <label>Días de prueba<Input required type="number" min="0" max="365" step="1" value={value.trialDays} onChange={e=>setValue({...value,trialDays:Number(e.target.value)})}/></label>
     <label>Máx. locales<Input type="number" min="1" step="1" value={value.maxLocations} onChange={e=>setValue({...value,maxLocations:e.target.value})} placeholder="Vacío = sin límite"/></label>
     <label>Máx. usuarios<Input type="number" min="1" step="1" value={value.maxUsers} onChange={e=>setValue({...value,maxUsers:e.target.value})} placeholder="Vacío = sin límite"/></label>
     <label className="switch-row compact"><input type="checkbox" checked={value.active} onChange={e=>setValue({...value,active:e.target.checked})}/><span/><b>Plan disponible para nuevas empresas</b></label>
    </div>
    <section className="plan-modules"><header><div><small>ENTITLEMENTS</small><h3>Módulos incluidos</h3></div><b>{value.moduleKeys.length}</b></header>{groups.map(group=><div className="plan-module-group" key={group}><strong>{group}</strong><div>{modules.filter(module=>module.category===group).map(module=><label key={module.key} className={value.moduleKeys.includes(module.key)?"selected":""}><input type="checkbox" checked={value.moduleKeys.includes(module.key)} onChange={()=>toggle(module.key)}/><span><Icon name="check" size={12}/></span><div><b>{module.name}</b><small>{module.description}</small></div></label>)}</div></div>)}</section>
   </div>
   <footer><Button kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy||!valid}>{busy?"Guardando…":"Guardar"}</Button></footer>
  </form>
 </section></div>;
}
