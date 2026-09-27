"use client";

import "./restaurant-setup-tour.css";
import Link from "next/link";
import {usePathname,useSearchParams} from "next/navigation";
import {useEffect,useState} from "react";
import {Button,Icon} from "@/design-system";
import {useRestaurantSetup} from "../application/use-restaurant-setup";
import {buildSetupSteps,setupGuideHref,setupOverviewHref,setupPath} from "./setup-steps";

export function RestaurantSetupTour(){
  const path=usePathname();
  const params=useSearchParams();
  const enabled=params.get("guia")==="puesta-en-marcha";
  const {query,session,canVisit}=useRestaurantSetup(enabled);
  const [collapsedStep,setCollapsedStep]=useState<string|null>(null);
  const steps=query.data?buildSetupSteps(query.data).filter(step=>step.key!=="review"&&canVisit(step.href)):[];
  const index=steps.findIndex(step=>step.href===path&&(step.href!==setupPath||step.key===(params.get("paso")??"operation")));
  const step=steps[index];
  const active=enabled&&Boolean(step);
  const collapsed=collapsedStep===step?.key;
  const targetSelector=step?.key==="operation"?'[data-setup-target="operation"]':".content .page-header";

  useEffect(()=>{
    if(!active||collapsed)return;
    let target:Element|null=null;
    const highlight=()=>{
      const next=document.querySelector(targetSelector);
      if(next===target)return;
      target?.classList.remove("setup-tour-target");
      target=next;
      target?.classList.add("setup-tour-target");
    };
    highlight();
    // Pages can first render a skeleton while their own data is loading.
    const observer=new MutationObserver(highlight);
    observer.observe(document.querySelector(".content")??document.body,{childList:true,subtree:true});
    return()=>{observer.disconnect();target?.classList.remove("setup-tour-target")};
  },[active,collapsed,targetSelector,path]);

  if(!enabled)return null;
  if(query.isLoading||query.isError)return <aside className="setup-tour" aria-label="Recorrido guiado"><div className="setup-tour-status" role="status"><Icon name={query.isError?"alert":"clock"} size={20}/><p>{query.isError?"No pudimos cargar la guía.":"Preparando tu recorrido…"}</p>{query.isError&&<Button kind="secondary" icon="refresh" onClick={()=>void query.refetch()}>Reintentar</Button>}<Link href={setupOverviewHref("operation")}>Volver al resumen</Link></div></aside>;
  if(!step)return null;
  const previous=steps[index-1];
  const next=steps[index+1];
  const exitHref=path===setupPath?setupOverviewHref(step.key):path;

  return <aside className={`setup-tour${collapsed?" is-collapsed":""}`} aria-label={`Recorrido guiado: ${step.title}`}>
    {collapsed?<Button icon="arrowRightCircle" onClick={()=>setCollapsedStep(null)}>Continuar guía · {step.title}</Button>:<div className="setup-tour-card">
      <header><span><Icon name="arrowRightCircle" size={15}/>RECORRIDO GUIADO</span><button type="button" aria-label="Minimizar guía" onClick={()=>setCollapsedStep(step.key)}><Icon name="minus" size={18}/></button></header>
      <div className="setup-tour-body"><div className="setup-tour-count"><span>Paso {index+1} de {steps.length}</span><span>{step.required?"Esencial":"Opcional"}</span></div><h2>{step.title}</h2><p>{step.description}</p><ol>{step.instructions.map(instruction=><li key={instruction}>{instruction}</li>)}</ol>{!session.can(step.permission)&&<p className="setup-tour-permission"><Icon name="lock" size={14}/>Puedes consultar esta sección. Para registrar cambios, solicita el permiso correspondiente.</p>}</div>
      <div className="setup-tour-track" aria-hidden="true">{steps.map((item,i)=><span key={item.key} className={i<=index?"is-visited":""}/>)}</div>
      <footer>{previous?<Link className="button secondary" href={setupGuideHref(previous)}><Icon name="chevronLeft" size={15}/>Anterior</Link>:<Link className="setup-tour-exit" href={exitHref}>Salir de la guía</Link>}<Link className="button primary" href={next?setupGuideHref(next):setupOverviewHref("review")}>{next?"Siguiente":"Ver revisión"}<Icon name="chevron" size={15}/></Link></footer>
      <Link className="setup-tour-summary" href={setupOverviewHref(step.key)}>Volver a mi puesta en marcha</Link>
    </div>}
  </aside>;
}
