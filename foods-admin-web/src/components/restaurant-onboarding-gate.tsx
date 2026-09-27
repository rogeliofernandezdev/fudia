"use client";

import {useState} from "react";
import Link from "next/link";
import {useSession} from "@/providers";

const steps=[
  {title:"Revisa tu restaurante",text:"Confirma la información principal de tu empresa y local."},
  {title:"Configura tu operación",text:"Define cómo atenderás pedidos: salón, mostrador o mixto."},
  {title:"Prepara tu catálogo",text:"Agrega productos para comenzar a recibir pedidos."},
  {title:"Activa tu equipo",text:"Invita usuarios y prepara la operación diaria."},
];

export function RestaurantOnboardingGate({children}:{children:React.ReactNode}){
  const {setupRequired,isLoading}=useSession();
  const [started,setStarted]=useState(false);
  if(isLoading||!setupRequired||started)return <>{children}</>;

  return <div className="restaurant-onboarding-overlay" role="dialog" aria-label="Primer recorrido del restaurante">
    <section>
      <span>BIENVENIDO A FUDIA</span>
      <h1>Preparemos tu restaurante para operar</h1>
      <p>Tu empresa ya está creada. Te acompañaremos en un recorrido rápido para dejar lista la operación.</p>
      <ol>{steps.map(step=><li key={step.title}><b>{step.title}</b><small>{step.text}</small></li>)}</ol>
      <div>
        <button type="button" onClick={()=>setStarted(true)}>Comenzar recorrido</button>
        <Link href="/configuracion/puesta-en-marcha">Ver configuración completa</Link>
      </div>
    </section>
  </div>;
}
