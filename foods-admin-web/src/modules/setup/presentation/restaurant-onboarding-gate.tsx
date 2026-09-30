"use client";

import {useState} from "react";
import {useSession} from "@/providers";

const steps=[
  {target:"restaurant",title:"Revisa tu restaurante",text:"Confirma la información principal de tu empresa y local."},
  {target:"operation",title:"Configura tu operación",text:"Define cómo atenderás pedidos: salón, mostrador o mixto."},
  {target:"catalog",title:"Prepara tu catálogo",text:"Agrega productos para comenzar a recibir pedidos."},
  {target:"team",title:"Activa tu equipo",text:"Invita usuarios y prepara la operación diaria."},
];

export function RestaurantOnboardingGate({children}:{children:React.ReactNode}){
  const {setupRequired,isLoading}=useSession();
  const [started,setStarted]=useState(false);
  const [step,setStep]=useState(0);
  if(isLoading||!setupRequired)return <>{children}</>;
  if(started)return <><div className="restaurant-tour-tooltip" data-tour-target={steps[step].target} role="dialog"><span>RECORRIDO FUDIA</span><h2>{steps[step].title}</h2><p>{steps[step].text}</p><button type="button" onClick={()=>step===steps.length-1?setStarted(false):setStep(step+1)}>{step===steps.length-1?"Comenzar a operar":"Continuar"}</button></div>{children}</>;

  return <div className="restaurant-onboarding-overlay" role="dialog" aria-label="Primer recorrido del restaurante">
    <section>
      <span>BIENVENIDO A FUDIA</span>
      <h1>Preparemos tu restaurante para operar</h1>
      <p>Tu empresa ya está creada. Te acompañaremos en un recorrido rápido para conocer dónde comenzar.</p>
      <ol>{steps.map(item=><li key={item.target}><b>{item.title}</b><small>{item.text}</small></li>)}</ol>
      <button type="button" onClick={()=>setStarted(true)}>Comenzar recorrido</button>
    </section>
  </div>;
}
