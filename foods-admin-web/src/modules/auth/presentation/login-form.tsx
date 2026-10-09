"use client";
import "./login.css";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { useRouter } from "next/navigation";
import Image from "next/image";
import { useQueryClient } from "@tanstack/react-query";
import { Icon, type IconName } from "@/design-system/icons";
import { Dialog, FormField, FullScreenLoader, Input } from "@/design-system";
import { ActiveSessionError, login, logout } from "../infrastructure/auth-api";
import { loginResolver, type LoginDraft } from "../domain/login-schema";
import {loadSessionContext} from "@/shared/session/session-api";
import {firstAccessibleRoute} from "@/shell/navigation";
import {broadcastSessionChange} from "@/shared/session/session-events";

const trustItems: Array<{ icon: IconName; label: string }> = [
  { icon: "lock", label: "Conexión segura" },
  { icon: "store", label: "Acceso por empresa" },
  { icon: "check", label: "Usuario validado" },
];

export function LoginForm() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [loading, setLoading] = useState(false);
  const [checkingSession,setCheckingSession]=useState(true);
  const [error, setError] = useState("");
  const [activeSession,setActiveSession]=useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [showRecoveryHelp, setShowRecoveryHelp] = useState(false);
  // Tras un login correcto la pantalla pasa al loader hasta que la ruta destino reemplace al login.
  const [redirecting,setRedirecting]=useState(false);

  const { register, handleSubmit, formState: { errors } } = useForm<LoginDraft>({ defaultValues: { email: "", password: "" }, resolver: loginResolver, mode: "onSubmit", reValidateMode: "onChange" });

  useEffect(()=>{
    let cancelled=false;
    void loadSessionContext().then(context=>{
      if(cancelled)return;
      queryClient.clear();
      queryClient.setQueryData(["session-context"],context);
      router.replace(firstAccessibleRoute(context));
      router.refresh();
    }).catch(()=>{if(!cancelled)setCheckingSession(false)});
    return()=>{cancelled=true};
  },[queryClient,router]);

  /** Entra a la primera ruta permitida por el rol sin volver a mostrar el formulario. */
  async function enterWorkspace(){
    setRedirecting(true);
    const context=await loadSessionContext();
    queryClient.setQueryData(["session-context"],context);
    const route=firstAccessibleRoute(context);
    router.prefetch(route);
    router.replace(route);
    router.refresh();
  }

  async function authenticate(values:LoginDraft,replaceExisting=false){
    if (loading) return;
    setLoading(true);
    setError("");
    try {
      if(replaceExisting){
        setActiveSession("");
        await logout();
        broadcastSessionChange("signed-out");
      }
      await login(values);
      queryClient.clear();
      broadcastSessionChange("signed-in");
      // El botón queda bloqueado: la pantalla cambia al loader hasta que cargue el destino.
      await enterWorkspace();
    } catch (reason) {
      setRedirecting(false);
      setLoading(false);
      if(reason instanceof ActiveSessionError)setActiveSession(reason.userName);
      else setError(reason instanceof Error ? reason.message : "No pudimos iniciar sesión.");
    }
  }

  const submit=handleSubmit(values=>authenticate(values));
  const switchAccount=handleSubmit(values=>authenticate(values,true));

  async function continueSession(){
    setLoading(true);
    setError("");
    try{
      queryClient.clear();
      await enterWorkspace();
    }catch{
      setRedirecting(false);
      setLoading(false);
      setActiveSession("");
      setError("La sesión anterior ya no está disponible. Ingresa nuevamente.");
    }
  }

  if(checkingSession)return <FullScreenLoader label="Validando sesión"/>;
  if(redirecting)return <FullScreenLoader label="Ingresando a tu espacio"/>;

  return <main className="admin-auth-page">
    <section className="admin-auth-shell" aria-labelledby="login-title">
      <div className="admin-auth-card">
        <div className="admin-auth-accent" aria-hidden="true" />
        <div className="admin-auth-body">
          <div className="admin-auth-brand">
            <Image src="/assets/images/login.png" alt="fudIA" width={1536} height={1024} sizes="(max-width: 480px) 160px, 180px" priority />
          </div>
          <h1 className="sr-only" id="login-title">Iniciar sesión</h1>
          <form className="admin-auth-form" onSubmit={submit} aria-busy={loading} noValidate>
            {error && <div className="admin-auth-error" role="alert"><Icon name="alert" size={17}/><span>{error}</span></div>}
            {activeSession ? <section className="admin-auth-session-conflict" role="alert">
              <header><Icon name="users" size={20}/><div><b>Sesión activa</b><span>Ya ingresaste como {activeSession} en este navegador.</span></div></header>
              <div>
                <button type="button" className="admin-auth-session-secondary" disabled={loading} onClick={()=>void continueSession()}>Continuar sesión</button>
                <button type="button" className="admin-auth-session-primary" disabled={loading} onClick={()=>void switchAccount()}>Cerrar sesión y cambiar de cuenta</button>
              </div>
            </section> : <>
              <FormField label="Correo electrónico" error={errors.email?.message}>
                <div className={"admin-auth-field"+(errors.email?" has-error":"")}>
                  <Icon name="mail" size={17}/>
                  <Input className="admin-auth-input" id="admin-email" type="email" inputMode="email" autoComplete="username" placeholder="nombre@restaurante.com" aria-invalid={Boolean(errors.email)} disabled={loading} {...register("email")} />
                </div>
              </FormField>
              <FormField label="Contraseña" error={errors.password?.message}>
                <div className={"admin-auth-field"+(errors.password?" has-error":"")}>
                  <Icon name="lock" size={17}/>
                  <Input className="admin-auth-input admin-auth-input-password" id="admin-password" type={showPassword ? "text" : "password"} autoComplete="current-password" placeholder="Tu contraseña" aria-invalid={Boolean(errors.password)} disabled={loading} {...register("password")}/>
                  <button type="button" disabled={loading} onClick={() => setShowPassword(value => !value)} aria-label={showPassword ? "Ocultar contraseña" : "Mostrar contraseña"} aria-pressed={showPassword}><Icon name="eye" size={17}/></button>
                </div>
              </FormField>
              <div className="admin-auth-actions">
                <button className="admin-auth-recovery" type="button" disabled={loading} onClick={() => setShowRecoveryHelp(true)}>Recuperar contraseña</button>
                <button className="admin-auth-submit" type="submit" disabled={loading} aria-live="polite">
                  {loading ? <><i aria-hidden="true"/>Verificando acceso…</> : <>Ingresar<Icon name="lock" size={17}/></>}
                </button>
              </div>
            </>}
          </form>
          <ul className="admin-auth-trust">{trustItems.map(item => <li key={item.label}><Icon name={item.icon} size={14}/><span>{item.label}</span></li>)}</ul>
        </div>
      </div>
      <footer className="admin-auth-footer">© 2026 fudIA · Todos los derechos reservados</footer>
    </section>
    {showRecoveryHelp && <div className="modal-backdrop modal-overlay-in">
      <Dialog className="crud-modal compact modal-panel-in" aria-labelledby="login-recovery-title" onResponseClose={() => setShowRecoveryHelp(false)}>
        <div className="modal-accent"/>
        <header>
          <span className="modal-title-icon"><Icon name="key"/></span>
          <div><h2 id="login-recovery-title">Próximamente</h2></div>
          <button type="button" aria-label="Cerrar ayuda" onClick={() => setShowRecoveryHelp(false)}><Icon name="close"/></button>
        </header>
      </Dialog>
    </div>}
  </main>;
}
