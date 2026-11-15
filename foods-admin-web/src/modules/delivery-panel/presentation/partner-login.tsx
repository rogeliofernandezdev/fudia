"use client";

import "@/modules/auth/presentation/login.css";
import "./partner-login.css";
import {useEffect,useState,type FormEvent} from "react";
import Image from "next/image";
import {useRouter} from "next/navigation";
import {useQueryClient} from "@tanstack/react-query";
import {FormField,FullScreenLoader,Icon,Input} from "@/design-system";
import {changePartnerPassword,loadPartnerSession,loginPartner} from "../infrastructure/panel-api";
import {passwordChangeProblem} from "../domain/panel";

type Step="checking"|"login"|"password"|"entering";

/** Campo de contraseña con el mismo diseño del acceso de la aplicación. */
function PasswordInput({id,label,value,onChange,autoComplete,error,disabled,autoFocus}:{id:string;label:string;value:string;onChange:(value:string)=>void;autoComplete:string;error?:string;disabled:boolean;autoFocus?:boolean}){
  const [visible,setVisible]=useState(false);
  return <FormField label={label} error={error}>
    <div className={"admin-auth-field"+(error?" has-error":"")}>
      <Icon name="lock" size={18}/>
      <Input className="admin-auth-input admin-auth-input-password" id={id} type={visible?"text":"password"} autoComplete={autoComplete} autoFocus={autoFocus} aria-invalid={Boolean(error)} disabled={disabled} value={value} onChange={event=>onChange(event.target.value)}/>
      <button type="button" disabled={disabled} onClick={()=>setVisible(!visible)} aria-label={visible?"Ocultar contraseña":"Mostrar contraseña"} aria-pressed={visible}><Icon name="eye" size={18}/></button>
    </div>
  </FormField>;
}

/**
 * Acceso de los operadores de una central de Delivery Fudia a su panel. Con la
 * contraseña temporal que entrega Fudia, el operador crea la suya antes de
 * entrar. `hasSession` lo resuelve el servidor desde la cookie.
 */
export function PartnerLogin({hasSession=false}:{hasSession?:boolean}){
  const router=useRouter();
  const queryClient=useQueryClient();
  const [step,setStep]=useState<Step>(hasSession?"checking":"login");
  const [email,setEmail]=useState("");
  const [password,setPassword]=useState("");
  const [current,setCurrent]=useState("");
  const [next,setNext]=useState("");
  const [repeat,setRepeat]=useState("");
  const [operator,setOperator]=useState("");
  const [error,setError]=useState("");
  const [touched,setTouched]=useState(false);
  const [busy,setBusy]=useState(false);
  // Tras recargar con la sesión temporal no conocemos la contraseña temporal: se vuelve a pedir.
  const [askCurrent,setAskCurrent]=useState(false);

  function enter(){
    setStep("entering");
    queryClient.removeQueries({queryKey:["delivery-partner-panel"]});
    router.replace("/delivery-panel");
    router.refresh();
  }

  useEffect(()=>{
    if(!hasSession)return;
    let cancelled=false;
    void loadPartnerSession().then(session=>{
      if(cancelled)return;
      if(!session.operator.mustChangePassword){enter();return}
      setOperator(session.operator.name);
      setAskCurrent(true);
      setStep("password");
    }).catch(()=>{if(!cancelled)setStep("login")});
    return()=>{cancelled=true};
  // eslint-disable-next-line react-hooks/exhaustive-deps
  },[hasSession]);

  const emailError=touched&&!email.trim()?"Ingresa tu correo electrónico.":touched&&!/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(email.trim())?"Ingresa un correo válido.":"";
  const passwordError=touched&&!password?"Ingresa tu contraseña.":"";

  async function signIn(event:FormEvent){
    event.preventDefault();
    setTouched(true);
    if(busy||!email.trim()||!password||!/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(email.trim()))return;
    setBusy(true);
    setError("");
    try{
      const session=await loginPartner(email.trim(),password);
      if(session.operator.mustChangePassword){
        setOperator(session.operator.name);
        setCurrent(password);
        setPassword("");
        setTouched(false);
        setStep("password");
      }else enter();
    }catch(reason){
      setError(reason instanceof Error?reason.message:"No pudimos iniciar sesión.");
    }finally{setBusy(false)}
  }

  const problem=touched?passwordChangeProblem(current,next,repeat):"";
  async function savePassword(event:FormEvent){
    event.preventDefault();
    setTouched(true);
    if(busy||passwordChangeProblem(current,next,repeat))return;
    setBusy(true);
    setError("");
    try{
      await changePartnerPassword(current,next);
      enter();
    }catch(reason){
      setError(reason instanceof Error?reason.message:"No pudimos cambiar la contraseña.");
      setBusy(false);
    }
  }

  if(step==="checking")return <FullScreenLoader label="Validando sesión"/>;
  if(step==="entering")return <FullScreenLoader label="Abriendo el panel de reparto"/>;

  return <main className="admin-auth-page partner-login-page">
    <section className="admin-auth-shell" aria-labelledby="partner-login-title">
      <div className="admin-auth-card">
        <div className="admin-auth-accent" aria-hidden="true"/>
        <div className="admin-auth-body">
          <div className="admin-auth-brand"><Image src="/assets/images/login-brand.png" alt="fudIA" width={360} height={240} sizes="120px" priority/></div>
          {step==="login"?<form className="admin-auth-form" onSubmit={signIn} aria-busy={busy} noValidate>
            <header className="admin-auth-heading"><h1 id="partner-login-title">Panel de reparto</h1><p>Ingresa con la cuenta de tu central.</p></header>
            {error&&<div className="admin-auth-error" role="alert"><Icon name="alert" size={18}/><span>{error}</span></div>}
            <FormField label="Correo electrónico" error={emailError||undefined}>
              <div className={"admin-auth-field"+(emailError?" has-error":"")}>
                <Icon name="mail" size={18}/>
                <Input className="admin-auth-input" id="partner-email" type="email" inputMode="email" autoComplete="username" placeholder="nombre@central.com" aria-invalid={Boolean(emailError)} disabled={busy} value={email} onChange={event=>setEmail(event.target.value)}/>
              </div>
            </FormField>
            <PasswordInput id="partner-password" label="Contraseña" value={password} onChange={setPassword} autoComplete="current-password" error={passwordError||undefined} disabled={busy}/>
            <button className="admin-auth-submit" type="submit" disabled={busy} aria-live="polite">{busy?<><i aria-hidden="true"/>Verificando acceso…</>:<>Ingresar<Icon name="key" size={20}/></>}</button>
          </form>:<form className="admin-auth-form" onSubmit={savePassword} aria-busy={busy} noValidate>
            <header className="admin-auth-heading"><h1 id="partner-login-title">Crea tu contraseña</h1><p>{operator?`${operator}, reemplaza`:"Reemplaza"} la contraseña temporal para entrar al panel.</p></header>
            {error&&<div className="admin-auth-error" role="alert"><Icon name="alert" size={18}/><span>{error}</span></div>}
            <input type="email" name="username" autoComplete="username" value={email} readOnly hidden/>
            {askCurrent&&<PasswordInput id="partner-current" label="Contraseña temporal" value={current} onChange={setCurrent} autoComplete="current-password" disabled={busy} autoFocus/>}
            <PasswordInput id="partner-new" label="Contraseña nueva" value={next} onChange={setNext} autoComplete="new-password" disabled={busy} autoFocus={!askCurrent}/>
            <PasswordInput id="partner-repeat" label="Repite la contraseña" value={repeat} onChange={setRepeat} autoComplete="new-password" error={problem||undefined} disabled={busy}/>
            <p className="partner-login-rule">Mínimo 8 caracteres, distinta de la temporal.</p>
            <button className="admin-auth-submit" type="submit" disabled={busy} aria-live="polite">{busy?<><i aria-hidden="true"/>Guardando…</>:<>Guardar y entrar<Icon name="check" size={20}/></>}</button>
          </form>}
          <p className="admin-auth-help"><Icon name="motorbike" size={16}/><span>¿Olvidaste tu contraseña? Pide a Fudia una temporal.</span></p>
        </div>
      </div>
      <footer className="admin-auth-footer">© 2026 fudIA · Todos los derechos reservados</footer>
    </section>
  </main>;
}
