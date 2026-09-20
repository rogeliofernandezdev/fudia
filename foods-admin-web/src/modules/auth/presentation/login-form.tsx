"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Image from "next/image";
import { useQueryClient } from "@tanstack/react-query";
import { Icon, type IconName } from "@/design-system/icons";
import { Input } from "@/design-system/page-header";
import { login } from "../infrastructure/auth-api";

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

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (loading) return;
    setLoading(true);
    setError("");
    const data = new FormData(event.currentTarget);
    try {
      await login({
        email: String(data.get("email") ?? ""),
        password: String(data.get("password") ?? ""),
      });
      queryClient.clear();
      router.replace("/dashboard");
      router.refresh();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "No pudimos iniciar sesión.");
    } finally {
      setLoading(false);
    }
  }

  return <main className="admin-auth-page">
    <section className="admin-auth-shell" aria-labelledby="login-title">
      <div className="admin-auth-context">ADMINISTRACIÓN</div>
      <div className="admin-auth-card">
        <div className="admin-auth-accent" aria-hidden="true" />
        <div className="admin-auth-body">
          <h1 className="admin-auth-sr-only" id="login-title">Iniciar sesión</h1>
          <form className="admin-auth-form" onSubmit={submit}>
            <div className="admin-auth-form-logo"><Image src="/assets/images/login.png" alt="fudIA" width={1536} height={1024} priority /></div>
            {error && <div className="admin-auth-error" role="alert"><Icon name="alert" size={17}/><span>{error}</span></div>}
            <label htmlFor="admin-email">Correo electrónico<div className="admin-auth-field"><Icon name="mail" size={17}/><Input className="admin-auth-input" id="admin-email" name="email" type="email" autoComplete="username" required placeholder="nombre@restaurante.com" /></div></label>
            <label htmlFor="admin-password">Contraseña<div className="admin-auth-field"><Icon name="lock" size={17}/><Input className="admin-auth-input admin-auth-input-password" id="admin-password" name="password" type={showPassword ? "text" : "password"} autoComplete="current-password" required placeholder="Ingresa tu contraseña"/><button type="button" onClick={() => setShowPassword(value => !value)} aria-label={showPassword ? "Ocultar contraseña" : "Mostrar contraseña"} aria-pressed={showPassword}><Icon name="eye" size={17}/></button></div></label>
            <button className="admin-auth-submit" type="submit" disabled={loading} aria-live="polite">{loading ? <><i aria-hidden="true"/>Verificando acceso…</> : <>Ingresar al sistema <Icon name="lock" size={17}/></>}</button>
          </form>
          <div className="admin-auth-trust">{trustItems.map(item => <div key={item.label}><Icon name={item.icon} size={16}/><span>{item.label}</span></div>)}</div>
        </div>
      </div>
      <footer className="admin-auth-footer"><Icon name="lock" size={13}/> © 2026 fudIA. Todos los derechos reservados.</footer>
    </section>
  </main>;
}
