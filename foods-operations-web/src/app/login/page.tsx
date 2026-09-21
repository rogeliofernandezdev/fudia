"use client";

import {useState} from "react";
import {useRouter} from "next/navigation";
import {Icon} from "@/components/icon";
import {Logo} from "@/components/logo";
import {Button,Input,Label} from "@/components/ui/controls";

const trustItems=[
  {icon:"lock" as const,label:"Conexión segura"},
  {icon:"store" as const,label:"Acceso por local"},
  {icon:"check" as const,label:"Usuario validado"},
];

export default function LoginPage(){
  const router=useRouter();
  const[showPassword,setShowPassword]=useState(false);
  const[isSubmitting,setIsSubmitting]=useState(false);
  const[email,setEmail]=useState("");
  const[password,setPassword]=useState("");
  const[error,setError]=useState("");

  async function submit(event:React.FormEvent<HTMLFormElement>){
    event.preventDefault();
    if(isSubmitting)return;
    setIsSubmitting(true);setError("");
    try{
      const response=await fetch("/api/auth/login",{
        method:"POST",
        headers:{"Content-Type":"application/json"},
        body:JSON.stringify({email,password}),
      });
      const data=await response.json().catch(()=>({}));
      if(!response.ok)throw new Error(data?.message??"No se pudo iniciar sesión.");
      router.replace("/turno");
      router.refresh();
    }catch(e){setError(e instanceof Error?e.message:"No se pudo iniciar sesión.");}
    finally{setIsSubmitting(false);}
  }

  return <main className="login-page login-bodegas-pattern">
    <section className="login-center">
      <header className="login-brand"><Logo/><small>RESTAURANT OPERATIONS</small></header>
      <div className="login-card">
        <div className="login-accent" aria-hidden="true"><i/><i/><i/></div>
        <div className="login-card-body">
          <div className="login-intro"><h1>Iniciar sesión</h1><p>Ingresa tus credenciales para acceder al centro de operaciones.</p></div>
          <form className="login-form" onSubmit={submit}>
            <Label htmlFor="login-email">Correo electrónico<div className="auth-field"><Icon name="user"/><Input className="auth-input" id="login-email" autoComplete="username" placeholder="nombre@restaurante.com" value={email} onChange={event=>setEmail(event.target.value)} type="email" required/></div></Label>
            <Label htmlFor="login-password">Contraseña<div className="auth-field"><Icon name="lock"/><Input className="auth-input auth-password" id="login-password" autoComplete="current-password" placeholder="Ingresa tu contraseña" value={password} onChange={event=>setPassword(event.target.value)} type={showPassword?"text":"password"} required/><Button type="button" className="auth-eye" layout="icon" aria-label={showPassword?"Ocultar contraseña":"Mostrar contraseña"} aria-pressed={showPassword} onClick={()=>setShowPassword(value=>!value)}><Icon name="eye"/></Button></div></Label>
            <div className="login-options"><Label><Input type="checkbox" defaultChecked/> Recordarme</Label><a href="#recuperar">¿Olvidaste tu contraseña?</a></div>
            {error&&<div className="missing-card" role="alert"><span>{error}</span></div>}
            <Button type="submit" tone="operational" className="login-submit" disabled={isSubmitting} aria-live="polite">{isSubmitting?<><span className="login-spinner" aria-hidden="true"/>Verificando acceso…</>:<>Ingresar al sistema <Icon name="fingerprint"/></>}</Button>
          </form>
          <div className="login-trust">{trustItems.map(item=><div key={item.label}><Icon name={item.icon} size={15}/><span>{item.label}</span></div>)}</div>
        </div>
      </div>
      <footer className="login-footer"><Icon name="lock" size={13}/> © 2026 fudIA. Todos los derechos reservados.</footer>
    </section>
  </main>;
}
