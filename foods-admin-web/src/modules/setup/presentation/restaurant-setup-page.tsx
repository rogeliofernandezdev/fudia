"use client";

import "./restaurant-setup.css";
import Link from "next/link";
import {useRouter,useSearchParams} from "next/navigation";
import {useMutation,useQueryClient} from "@tanstack/react-query";
import {Button,Icon,PageHeader,Status} from "@/design-system";
import {useFeedback} from "@/providers/feedback-provider";
import {firstAccessibleRoute} from "@/shell/navigation";
import {pageRoutes} from "@/shared/routing/page-routes";
import {useRestaurantSetup} from "../application/use-restaurant-setup";
import type {RestaurantSetup} from "../domain/types";
import {completeRestaurantSetup,updateRestaurantServiceMode} from "../infrastructure/setup-api";
import {buildSetupSteps,setupGuideHref,setupOverviewHref,type SetupStep} from "./setup-steps";

const serviceModes=[
  {value:"counter",title:"Mostrador",description:"Pedidos para llevar, sin asignar una mesa.",icon:"store"},
  {value:"dine_in",title:"Salón",description:"Atención en mesas y zonas de tu local.",icon:"grid"},
  {value:"mixed",title:"Mixta",description:"Combina salón y pedidos para llevar.",icon:"utensils"},
] as const;

export function RestaurantSetupPage(){
  const router=useRouter();
  const searchParams=useSearchParams();
  const qc=useQueryClient();
  const {notify}=useFeedback();
  const {query,queryKey,session,canVisit}=useRestaurantSetup();
  const setup=query.data;
  const steps=setup?buildSetupSteps(setup):[];
  const activeIndex=Math.max(0,steps.findIndex(step=>step.key===searchParams.get("paso")));
  const active=steps[activeIndex];
  const required=steps.filter(step=>step.required);
  const completed=required.filter(step=>step.complete).length;
  const canManage=session.can("organizations.manage");
  const guideStep=steps.find(step=>step.key!=="review"&&canVisit(step.href));
  const configured=Boolean(setup?.completedAt&&setup.coreReady);
  const homeHref=session.user&&session.modules?firstAccessibleRoute({...session,user:session.user,modules:session.modules,setupRequired:false}):pageRoutes.noAccess;

  const saveMode=useMutation({
    mutationFn:updateRestaurantServiceMode,
    onSuccess:data=>{qc.setQueryData(queryKey,data);void qc.invalidateQueries({queryKey:["session-context"]});notify({tone:"success",title:"Forma de atención guardada",message:"La guía ya está adaptada a la operación de tu restaurante."})},
    onError:error=>notify({tone:"danger",title:"No se pudo guardar",message:error.message}),
  });
  const finish=useMutation({
    mutationFn:completeRestaurantSetup,
    onSuccess:async data=>{qc.setQueryData(queryKey,data);await qc.invalidateQueries({queryKey:["session-context"]});notify({tone:"success",title:"Restaurante listo",message:"La configuración esencial está completa. Ya puedes comenzar a atender."})},
    onError:error=>notify({tone:"danger",title:"Aún faltan datos",message:error.message}),
  });

  if(query.isLoading||session.isLoading)return <SetupSkeleton/>;
  if(query.isError)return <div className="restaurant-setup"><SetupHeader/><section className="setup-state" role="alert"><Icon name="alert" size={28}/><h2>No pudimos cargar tu avance</h2><p>{query.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>void query.refetch()}>Reintentar</Button></section></div>;
  if(!setup||!active)return <div className="restaurant-setup"><SetupHeader/><section className="setup-state"><Icon name="store" size={28}/><h2>Selecciona un local</h2><p>Elige la empresa y el local que quieres preparar.</p></section></div>;

  function selectStep(key:string){router.replace(setupOverviewHref(key),{scroll:false})}

  return <div className="restaurant-setup">
    <SetupHeader/>
    <section className="setup-welcome" aria-label="Resumen de puesta en marcha">
      <div className="setup-welcome-copy"><span className="setup-eyebrow"><Icon name="arrowRightCircle" size={15}/>A TU RITMO, PASO A PASO</span><h2>{configured?"Todo listo para tu próximo servicio":"Un buen comienzo para tu restaurante"}</h2><p>{configured?"La configuración esencial está completa. Puedes revisar tus datos y seguir preparando cada detalle.":"Prepara lo esencial y descubre qué registrar en cada sección. Te acompañamos durante el recorrido."}</p><div className="setup-welcome-actions">{guideStep&&<Link className="button primary" href={setupGuideHref(guideStep)}><Icon name="arrowRightCircle" size={18}/><span>Iniciar recorrido guiado</span></Link>}<span><Icon name="check" size={15}/>Tu avance se conserva</span></div></div>
      <div className="setup-progress-summary"><span>CONFIGURACIÓN ESENCIAL</span><div><strong>{completed}<small> / {required.length}</small></strong><Status tone={configured?"green":setup.coreReady?"blue":"gray"}>{configured?"Completada":setup.coreReady?"Lista para finalizar":"En progreso"}</Status></div><progress value={completed} max={required.length} aria-label={`${completed} de ${required.length} requisitos esenciales completos`}/><p>{completed===required.length?"Los requisitos esenciales están listos.":`${required.length-completed} ${required.length-completed===1?"paso pendiente":"pasos pendientes"} para comenzar a atender.`}</p></div>
    </section>

    <div className="setup-workspace">
      <nav className="setup-step-nav" aria-label="Pasos de puesta en marcha">
        <div className="setup-nav-heading"><span>TU HOJA DE RUTA</span><small>{steps.length} pasos</small></div>
        <ol>{steps.map((step,index)=><li key={step.key}><button type="button" className={`setup-step${step.key===active.key?" is-current":""}${step.complete?" is-complete":""}`} onClick={()=>selectStep(step.key)} aria-current={step.key===active.key?"step":undefined}><span className="setup-step-number">{step.complete?<Icon name="check" size={15}/>:String(index+1).padStart(2,"0")}</span><span className="setup-step-label"><b>{step.title}</b><small>{step.key==="review"?"Antes de comenzar":step.complete?"Listo":step.required?"Esencial":"Opcional"}</small></span><Icon name="chevron" size={14}/></button></li>)}</ol>
        <p className="setup-nav-note"><Icon name="lock" size={15}/><span>Los pasos opcionales se pueden completar más adelante.</span></p>
      </nav>

      <section className="setup-detail" aria-labelledby="setup-detail-title">
        <header className="setup-detail-header"><div className="setup-detail-meta"><span>PASO {String(activeIndex+1).padStart(2,"0")} <i>/ {String(steps.length).padStart(2,"0")}</i></span><Status tone={active.complete?"green":"gray"}>{active.complete?"Listo":active.key==="review"?"Resumen":active.required?"Esencial":"Opcional"}</Status></div><div className="setup-detail-title"><span><Icon name={active.icon} size={23}/></span><div><h2 id="setup-detail-title">{active.title}</h2><p>{active.description}</p></div></div></header>
        <div className="setup-detail-body">
          {active.key==="operation"?<><fieldset className="setup-service-modes" disabled={!canManage||saveMode.isPending} data-setup-target="operation"><legend>¿Cómo atenderás a tus clientes?</legend>{serviceModes.map(mode=><label className={`setup-service-mode${setup.serviceMode===mode.value?" is-selected":""}`} key={mode.value}><input type="radio" name="service-mode" value={mode.value} checked={setup.serviceMode===mode.value} onChange={()=>saveMode.mutate(mode.value)}/><span className="setup-mode-icon"><Icon name={mode.icon} size={24}/></span><span className="setup-mode-check"><Icon name="check" size={12}/></span><b>{mode.title}</b><small>{mode.description}</small></label>)}</fieldset><p className="setup-save-note" aria-live="polite">{saveMode.isPending?"Guardando tu selección…":!canManage?"Necesitas permiso de administración para cambiar la forma de atención.":"La selección se guarda automáticamente para tu restaurante."}</p></>:
          active.key==="review"?<><SetupReview steps={steps} setup={setup} canVisit={canVisit} selectStep={selectStep}/>{!canManage&&!configured&&<p className="setup-permission"><Icon name="lock" size={16}/>Necesitas permiso de administración para finalizar la puesta en marcha.</p>}</>:<SetupRequirements step={active} setup={setup}/>}

          {active.key!=="review"&&<><div className="setup-instructions"><h3>{active.key==="operation"?"Así te acompañaremos":"Qué debes registrar"}</h3><ol>{active.instructions.map((instruction,index)=><li key={instruction}><span>{index+1}</span><p>{instruction}</p></li>)}</ol></div><aside className="setup-advice"><Icon name="chefHat" size={20}/><div><b>Para empezar con buen pie</b><p>{active.advice}</p></div></aside></>}

          {active.key!=="operation"&&active.key!=="review"&&<div className="setup-open-section">{canVisit(active.href)?<><Link className="button primary" href={setupGuideHref(active)}><Icon name={active.icon} size={17}/><span>{active.complete?"Revisar con guía":"Configurar con guía"}</span><Icon name="chevron" size={15}/></Link><span>Te acompañamos en la pantalla de {active.title.toLocaleLowerCase("es")}.</span></>:<p className="setup-permission"><Icon name="lock" size={16}/>Pide al administrador acceso a esta sección para completar el paso.</p>}</div>}
        </div>
        <footer className="setup-detail-footer"><Button kind="ghost" icon="chevronLeft" disabled={activeIndex===0} onClick={()=>selectStep(steps[activeIndex-1].key)}>Anterior</Button>{active.key==="review"?configured?<Link className="button primary" href={canVisit("/pos")?"/pos":homeHref}>Comenzar a trabajar<Icon name="chevron" size={16}/></Link>:<Button icon="check" disabled={!setup.coreReady||!canManage||finish.isPending} onClick={()=>finish.mutate()}>{finish.isPending?"Finalizando…":"Finalizar puesta en marcha"}</Button>:<Button kind="secondary" onClick={()=>selectStep(steps[activeIndex+1].key)}>Siguiente paso<Icon name="chevron" size={16}/></Button>}</footer>
      </section>
    </div>
    <p className="setup-footer-note"><Icon name="refresh" size={14}/>El avance se actualiza con los datos que registras en cada sección.{query.isFetching&&<span> Actualizando…</span>}</p>
  </div>;
}

function SetupHeader(){return <PageHeader eyebrow="CONFIGURACIÓN / GUÍA DE INICIO" title="Puesta en marcha" description="Todo lo que necesitas para comenzar a atender, en un solo lugar."/>}

function SetupRequirements({step,setup}:{step:SetupStep;setup:RestaurantSetup}){
  const counts=setup.counts;
  const requirements:Record<string,Array<{title:string;count:number;ready:boolean;unit:string}>>={
    tables:[{title:"Mesas del local",count:counts.tables,ready:counts.tables>0,unit:"mesas activas"}],
    catalog:[{title:"Categorías",count:counts.categories,ready:counts.categories>0,unit:"categorías registradas"},{title:"Productos",count:counts.products,ready:counts.products>0,unit:"productos disponibles"}],
    cash:[{title:"Caja operativa",count:counts.cashRegisters,ready:counts.cashRegisters>0,unit:"cajas activas"}],
    team:[{title:"Usuarios del equipo",count:counts.users,ready:counts.users>1,unit:"usuarios activos"}],
    inventory:[{title:"Artículos de inventario",count:counts.inventoryItems,ready:counts.inventoryItems>0,unit:"artículos registrados"}],
    recipes:[{title:"Recetas",count:counts.recipes,ready:counts.recipes>0,unit:"recetas activas"}],
    purchases:[{title:"Proveedores",count:counts.suppliers,ready:counts.suppliers>0,unit:"proveedores activos"}],
  };
  return <div className="setup-requirements">{requirements[step.key]?.map(item=><article className="setup-requirement" key={item.title}><span className={`setup-requirement-icon${item.ready?" is-ready":""}`}><Icon name={item.ready?"check":step.icon} size={20}/></span><div><h3>{item.title}</h3><p>{item.ready?"Registrado y disponible":step.required?"Pendiente de registrar":"Puedes completarlo después"}</p></div><strong aria-label={`${item.count} ${item.unit}`}>{item.count}<small>{item.unit.split(" ")[0]}</small></strong></article>)}</div>;
}

function SetupReview({steps,setup,canVisit,selectStep}:{steps:SetupStep[];setup:RestaurantSetup;canVisit:(href:string)=>boolean;selectStep:(key:string)=>void}){
  const configured=Boolean(setup.completedAt&&setup.coreReady);
  return <div className="setup-review"><div className={`setup-review-message${setup.coreReady?" is-ready":""}`}><Icon name={setup.coreReady?"check":"clock"} size={22}/><div><h3>{configured?"Tu restaurante está preparado":setup.coreReady?"Ya tienes lo esencial para comenzar":"Completa los pasos esenciales"}</h3><p>{setup.coreReady?"Las opciones restantes se pueden completar cuando las necesites.":"Revisa los requisitos pendientes. Los pasos opcionales no impiden que comiences a atender."}</p></div></div><ul>{steps.filter(step=>step.key!=="review").map(step=><li key={step.key}><Icon name={step.complete?"check":"clock"} size={16}/><div><b>{step.title}</b><small>{step.required?"Esencial":"Opcional"}</small></div><span>{step.complete?"Listo":"Pendiente"}</span><Button kind="ghost" disabled={!canVisit(step.href)} aria-label={`Revisar ${step.title}`} onClick={()=>selectStep(step.key)}>Revisar</Button></li>)}</ul></div>;
}

function SetupSkeleton(){return <div className="restaurant-setup setup-loading" role="status" aria-label="Cargando puesta en marcha"><div className="setup-loading-title"><i/><i/></div><div className="setup-loading-welcome"><div><i/><i/><i/></div><i/></div><div className="setup-loading-workspace"><aside>{Array.from({length:6},(_,index)=><div key={index}><i/><span><i/><i/></span></div>)}</aside><section><i/><i/><div><i/><i/><i/></div><i/><i/></section></div></div>}
