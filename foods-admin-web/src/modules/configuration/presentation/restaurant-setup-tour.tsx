"use client";

import "./restaurant-setup-tour.css";
import Link from "next/link";
import {usePathname,useSearchParams} from "next/navigation";
import {useEffect,useState} from "react";
import {Icon} from "@/design-system/icons";
import {useSession} from "@/providers/session-context";
import {canOpenNavigationItem,navigationItemForPath} from "@/shell/navigation";
import {guideSteps} from "./restaurant-setup-guide";

const tourHref=(href:string)=>`${href}?guia=puesta-en-marcha`;

export function RestaurantSetupTour(){
  const path=usePathname();
  const searchParams=useSearchParams();
  const{user,organization,location,modules,menuAccess,permissions,can}=useSession();
  const[collapsed,setCollapsed]=useState(false);
  const context=user&&modules?{user,modules,menuAccess,permissions}:null;
  const steps=context?guideSteps.filter(item=>{
    const nav=navigationItemForPath(item.href);
    return nav&&canOpenNavigationItem(nav,context);
  }):[];
  const index=steps.findIndex(item=>item.href===path);
  const step=steps[index];
  const active=searchParams.get("guia")==="puesta-en-marcha"&&Boolean(step&&organization&&location);

  useEffect(()=>{
    if(!active||!step||!user||!organization||!location)return;
    const heading=document.querySelector(".content .page-header");
    heading?.classList.add("setup-tour-target");
    try{
      const key=`fudia-setup-guide:${user.id}:${organization.id}:${location.id}`;
      window.localStorage.setItem(key,step.id);
      window.dispatchEvent(new Event("fudia-setup-progress"));
    }catch{/* Guidance remains available without browser storage. */}
    return()=>heading?.classList.remove("setup-tour-target");
  },[active,step,user,organization,location]);

  if(!active||!step)return null;
  const next=steps[index+1];
  const editPermission=step.id==="usuarios"?"users.manage":step.id==="mesas"?"tables.manage":step.id==="caja"?"cash.manage":step.id==="productos"?"menu.manage":"organizations.manage";
  function finish(){
    if(!user||!organization||!location)return;
    try{
      window.localStorage.setItem(`fudia-setup-guide:${user.id}:${organization.id}:${location.id}`,"done");
      window.dispatchEvent(new Event("fudia-setup-progress"));
    }catch{/* Finishing the tour does not depend on browser storage. */}
  }

  return <aside className={`setup-tour${collapsed?" collapsed":""}`} aria-label={`Asistente de puesta en marcha, paso ${index+1} de ${steps.length}`}>
    {collapsed?<button type="button" className="setup-tour-reopen" onClick={()=>setCollapsed(false)}><Icon name="check" size={17}/>Guía: {step.title}<Icon name="chevron" size={15}/></button>:
    <div className="setup-tour-tooltip">
      <header><span>GUÍA · {index+1} DE {steps.length}</span><button type="button" onClick={()=>setCollapsed(true)} aria-label="Minimizar guía"><Icon name="close" size={17}/></button></header>
      <div className="setup-tour-progress" aria-hidden="true"><span style={{width:`${(index+1)/steps.length*100}%`}}/></div>
      <h2><Icon name={step.icon} size={19}/>{step.title}</h2>
      <p>{step.tooltip}</p>
      <ul>{step.details.map(detail=><li key={detail}>{detail}</li>)}</ul>
      {!can(editPermission)&&<p className="setup-tour-permission">Para guardar cambios en esta opción, pide el permiso a un administrador.</p>}
      <footer><Link href={path} className="setup-tour-exit">Salir de la guía</Link><div>{index>0&&<Link href={tourHref(steps[index-1].href)} className="setup-tour-back">Anterior</Link>}<Link href={next?tourHref(next.href):"/configuracion/puesta-en-marcha"} className="setup-tour-next" onClick={next?undefined:finish}>{next?"Siguiente opción":"Terminar recorrido"}<Icon name="chevron" size={15}/></Link></div></footer>
    </div>}
  </aside>;
}
