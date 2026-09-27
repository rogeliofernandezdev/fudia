"use client";
import "./login.css";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { useRouter } from "next/navigation";
import Image from "next/image";
import { useQueryClient } from "@tanstack/react-query";
import { Icon, type IconName } from "@/design-system/icons";
import { FormField, Input } from "@/design-system";
import { login } from "../infrastructure/auth-api";
import { loginResolver, type LoginDraft } from "../domain/login-schema";
import {loadSessionContext} from "@/shared/session/session-api";
import {firstAccessibleRoute} from "@/shell/navigation";

const trustItems: Array<{ icon: IconName; label: string }> = [
  { icon: "lock", label: "Conexión segura" },
  { icon: "store", label: "Acceso por empresa" },
  { icon: "check", label: "Usuario validado" },
];

export function LoginForm() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [showPassword, setShowPassword] = useState(false);

  const { register, handleSubmit, formState: { errors } } = useForm<LoginDraft>({ defaultValues: { email: "", password: "" }, resolver: loginResolver, mode: "onSubmit", reValidateMode: "onChange" });

  const submit = handleSubmit(async (values) => {
    if (loading) return;
    setLoading(true);
    setError("");
    try {
      await login(values);
      queryClient.clear();
      const context=await loadSessionContext();
      queryClient.setQueryData(["session-context"],context);
      router.replace(firstAccessibleRoute(context));
      router.refresh();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "No pudimos iniciar sesión.");
    } finally {
      setLoading(false);
    }
  });

  return <main className="admin-auth-page">
    <section className="admin-auth-shell" aria-labelledby="login-title">
      <div className="admin-auth-brand">
        <Image src="/assets/images/login.png" alt="fudIA" width={1536} height={1024} priority />
        <span className="admin-auth-context">Administración</span>
      </div>
      <div className="admin-auth-card">
        <div className="admin-auth-accent" aria-hidden="true" />
        <div className="admin-auth-body">
          <div className="admin-auth-intro">
            <h1 id="login-title">Bienvenido de nuevo</h1>
            <p>Ingresa con el correo y la contraseña de tu empresa.</p>
          </div>
          <form className="admin-auth-form" onSubmit={submit} noValidate>
            {error && <div className="admin-auth-error" role="alert"><Icon name="alert" size={17}/><span>{error}</span></div>}
            <FormField label="Correo electrónico" error={errors.email?.message}><div className={"admin-auth-field"+(errors.email?" has-error":"")}><Icon name="mail" size={17}/><Input className="admin-auth-input" id="admin-email" type="email" inputMode="email" autoComplete="username" placeholder="nombre@restaurante.com" aria-invalid={Boolean(errors.email)} {...register("email")} /></div></FormField>
            <FormField label="Contraseña" error={errors.password?.message}><div className={"admin-auth-field"+(errors.password?" has-error":"")}><Icon name="lock" size={17}/><Input className="admin-auth-input admin-auth-input-password" id="admin-password" type={showPassword ? "text" : "password"} autoComplete="current-password" placeholder="Tu contraseña" aria-invalid={Boolean(errors.password)} {...register("password")}/><button type="button" onClick={() => setShowPassword(value => !value)} aria-label={showPassword ? "Ocultar contraseña" : "Mostrar contraseña"} aria-pressed={showPassword}><Icon name="eye" size={17}/></button></div></FormField>
            <button className="admin-auth-submit" type="submit" disabled={loading} aria-live="polite">{loading ? <><i aria-hidden="true"/>Verificando acceso…</> : <>Ingresar al sistema<Icon name="login" size={17}/></>}</button>
          </form>
          <ul className="admin-auth-trust">{trustItems.map(item => <li key={item.label}><Icon name={item.icon} size={14}/><span>{item.label}</span></li>)}</ul>
        </div>
      </div>
      <footer className="admin-auth-footer">© 2026 fudIA · Todos los derechos reservados</footer>
    </section>
  </main>;
}
