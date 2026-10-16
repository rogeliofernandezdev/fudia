"use client";
import "./platform-shell.css";
import Link from "next/link";
import {usePathname} from "next/navigation";
import {FullScreenLoader} from "@/design-system";
import {Icon} from "@/design-system/icons";
import {Logo} from "@/design-system/logo";
import {useSession} from "@/providers";
import {pageRoutes} from "@/shared/routing/page-routes";

export function PlatformShell({children}:{children:React.ReactNode}){
  const path=usePathname();
  const{user,isLoading,isError}=useSession();
  if(isLoading)return <FullScreenLoader label="Validando acceso de plataforma"/>;
  if(isError)return <main className="platform-content"><div className="catalog-state"><b>No pudimos validar tu sesión.</b><Link href="/dashboard">Volver al admin</Link></div></main>;
  if(!user?.platformAdmin)return <main className="platform-content"><div className="catalog-state"><b>Acceso restringido</b><p>Solo un administrador de plataforma puede registrar empresas.</p><Link href="/dashboard">Volver al admin</Link></div></main>;
  return <div className="platform-shell">
    <header className="platform-topbar">
      <div className="platform-brand">
        <Link href="/dashboard" aria-label="Ir al admin"><Logo/></Link>
        <span className="platform-brand-divider" aria-hidden="true"/>
        <span className="platform-context"><small>Plataforma</small><b>Administración SaaS</b></span>
      </div>
      <nav className="platform-nav" aria-label="Administración de plataforma">
        <Link className={path===pageRoutes.platformOnboarding?"active":""} href={pageRoutes.platformOnboarding} aria-current={path===pageRoutes.platformOnboarding?"page":undefined}><Icon name="building" size={16}/><span>Registrar empresa</span></Link>
        <Link className={path===pageRoutes.platformPlans?"active":""} href={pageRoutes.platformPlans} aria-current={path===pageRoutes.platformPlans?"page":undefined}><Icon name="payment" size={16}/><span>Planes SaaS</span></Link>
        <Link className={path===pageRoutes.platformSubscription?"active":""} href={pageRoutes.platformSubscription} aria-current={path===pageRoutes.platformSubscription?"page":undefined}><Icon name="receipt" size={16}/><span>Suscripción actual</span></Link>
        <Link className={path===pageRoutes.platformGlobalSettings?"active":""} href={pageRoutes.platformGlobalSettings} aria-current={path===pageRoutes.platformGlobalSettings?"page":undefined}><Icon name="chat" size={16}/><span>Países y WhatsApp</span></Link>
      </nav>
      <Link href="/dashboard" className="button secondary platform-exit" aria-label="Volver al admin"><Icon name="chevronLeft" size={16}/>Volver al admin</Link>
    </header>
    <main className="platform-content">{children}</main>
  </div>;
}
