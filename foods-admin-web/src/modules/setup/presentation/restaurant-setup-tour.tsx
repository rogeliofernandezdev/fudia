"use client";

import "./restaurant-setup-tour.css";
import Link from "next/link";
import {useSearchParams} from "next/navigation";
import {useEffect,useState} from "react";
import {Button,Icon} from "@/design-system";
import {useRestaurantSetup} from "../application/use-restaurant-setup";
import {buildSetupSteps,setupOverviewHref} from "./setup-steps";

const tourTargets:Record<string,string>={
  operation:'[data-tour="operation"]',
  tables:'[data-tour="tables"]',
  catalog:'[data-tour="catalog"]',
  cash:'[data-tour="cash"]',
  team:'[data-tour="team"]',
};

export function RestaurantSetupTour(){
  const params=useSearchParams();
  const enabled=params.get("guia")==="puesta-en-marcha";
  const {query,canVisit}=useRestaurantSetup(enabled);
  const [closed,setClosed]=useState(false);
  const [index,setIndex]=useState(0);

  const steps=query.data
    ? buildSetupSteps(query.data).filter(step=>step.key!=="review"&&canVisit(step.href))
    : [];

  const step=steps[index];

  useEffect(()=>{
    if(!enabled||!step||closed)return;
    const selector=tourTargets[step.key]??".page-header";
    const target=document.querySelector(selector);
    target?.classList.add("setup-tour-target");
    return()=>target?.classList.remove("setup-tour-target");
  },[enabled,step,closed]);

  if(!enabled||closed||!step)return null;

  return <aside className="setup-tour" aria-label="Recorrido guiado del restaurante">
    <div className="setup-tour-card">
      <header>
        <span><Icon name="arrowRightCircle" size={15}/>RECORRIDO DEL RESTAURANTE</span>
        <button type="button" onClick={()=>setClosed(true)} aria-label="Cerrar recorrido">
          <Icon name="close" size={16}/>
        </button>
      </header>

      <div className="setup-tour-body">
        <div className="setup-tour-count">{index+1} de {steps.length}</div>
        <h2>{step.title}</h2>
        <p>{step.description}</p>
        <div className="setup-tooltip">
          <Icon name="lightbulb" size={18}/>
          <span>{step.advice}</span>
        </div>
      </div>

      <div className="setup-tour-track" aria-hidden="true">
        {steps.map((item,i)=><span key={item.key} className={i<=index?"is-visited":""}/>) }
      </div>

      <footer>
        <Button kind="ghost" disabled={index===0} onClick={()=>setIndex(value=>Math.max(0,value-1))}>
          Anterior
        </Button>
        {index<steps.length-1 ? (
          <Button onClick={()=>setIndex(value=>value+1)}>
            Siguiente
          </Button>
        ) : (
          <Link className="button primary" href={setupOverviewHref("review")}>
            Finalizar recorrido
          </Link>
        )}
      </footer>

      <Link className="setup-tour-summary" href={setupOverviewHref(step.key)}>
        Abrir esta sección
      </Link>
    </div>
  </aside>;
}
