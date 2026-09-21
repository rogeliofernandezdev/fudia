"use client";
import "./platform-shell.css";
import Link from "next/link";
import {Icon} from "@/design-system/icons";
import {Logo} from "@/design-system/logo";
import {useSession} from "@/providers";

export function PlatformShell({children}:{children:React.ReactNode}){
  const{user,isLoading,isError}=useSession();
  if(isLoading)return <main className="platform-content"><div className="catalog-state"><b>Cargando acceso de plataforma…</b></div></main>;
  if(isError)return <main className="platform-content"><div className="catalog-state"><b>No pudimos validar tu sesión.</b><Link href="/dashboard">Volver al admin</Link></div></main>;
  if(!user?.platformAdmin)return <main className="platform-content"><div className="catalog-state"><b>Acceso restringido</b><p>Solo un administrador de plataforma puede registrar empresas.</p><Link href="/dashboard">Volver al admin</Link></div></main>;
  return <div className="platform-shell">
    <header className="platform-topbar">
      <Link href="/dashboard"><Logo/></Link>
      <span className="platform-badge"><Icon name="power" size={14}/>PLATAFORMA</span>
      <Link href="/dashboard" className="platform-exit"><Icon name="chevronLeft" size={16}/>Volver al admin</Link>
    </header>
    <main className="platform-content">{children}</main>
  </div>;
}
