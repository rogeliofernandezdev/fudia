"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useState } from "react";
import { Icon } from "./icon";
import { Logo } from "./logo";
import { Select } from "./ui/controls";

const navigation = [
  { href: "/inicio", label: "Inicio", icon: "home" as const },
  { href: "/mesas", label: "Mesas", icon: "tables" as const },
  { href: "/pos", label: "Punto de venta", icon: "pos" as const },
  { href: "/cocina", label: "Comandas", icon: "kitchen" as const, badge: 3 },
  { href: "/pedidos", label: "Pedidos", icon: "orders" as const },
  { href: "/caja", label: "Caja", icon: "cash" as const },
];

const managementNav = [
  { href: "/reservas", label: "Reservas", icon: "tables" as const },
  { href: "/menu", label: "Disponibilidad", icon: "pos" as const },
  { href: "/inventario", label: "Inventario", icon: "store" as const },
  { href: "/clientes", label: "Clientes", icon: "user" as const },
  { href: "/personal", label: "Personal", icon: "user" as const },
  { href: "/reportes", label: "Reportes", icon: "receipt" as const },
];

const bottomNavigation = [
  { href: "/inicio", label: "Inicio", icon: "home" as const },
  { href: "/mesas", label: "Mesas", icon: "tables" as const },
  { href: "/pos", label: "Venta", icon: "pos" as const, floating: true },
  { href: "/cocina", label: "Comandas", icon: "kitchen" as const, badge: 3 },
  { href: "/pedidos", label: "Pedidos", icon: "orders" as const },
];

const isActivePath = (href: string, pathname: string) =>
  href === "/pos" ? pathname === "/pos" || pathname.startsWith("/pos/") : pathname === href;

export function AppShell({ children, title }: { children: React.ReactNode; title: string }) {
  const pathname = usePathname();
  const router = useRouter();
  const [collapsed, setCollapsed] = useState(false);
  const [profileOpen, setProfileOpen] = useState(false);
  const [isSigningOut, setIsSigningOut] = useState(false);
  const signOut = async () => {
    if (isSigningOut) return;
    setIsSigningOut(true);
    try {
      await fetch("/api/session", { method: "DELETE" });
    } finally {
      router.replace("/login");
      router.refresh();
    }
  };
  return <div className="shell" data-sidebar={collapsed ? "collapsed" : "expanded"}>
    <aside className="side">
      <div className="side-brand"><Link href="/inicio" aria-label="Ir al inicio"><Logo inverse /></Link><button className="side-toggle" type="button" aria-label={collapsed ? "Expandir navegación" : "Contraer navegación"} aria-expanded={!collapsed} onClick={() => setCollapsed(value => !value)}><Icon name="menu" size={20}/></button></div>
      <div className="side-scroll" tabIndex={0} aria-label="Secciones de la aplicación">
      <span className="side-label">OPERACIÓN</span>
      <nav aria-label="Operación">{navigation.map(item => <Link key={item.href} aria-current={isActivePath(item.href, pathname) ? "page" : undefined} className={isActivePath(item.href, pathname) ? "active" : ""} href={item.href}>
        <Icon name={item.icon} /><span>{item.label}</span>{item.badge && <em>{item.badge}</em>}
      </Link>)}</nav>
      <span className="side-label">GESTIÓN</span>
      <nav aria-label="Gestión">{managementNav.map(item => <Link key={item.href} aria-current={isActivePath(item.href, pathname) ? "page" : undefined} className={isActivePath(item.href, pathname) ? "active" : ""} href={item.href}>
        <Icon name={item.icon} /><span>{item.label}</span>
      </Link>)}</nav>
      </div>
      <div className="side-status"><Icon name="wifi" size={17}/><span><b>Todo conectado</b><small>Sincronizado ahora</small></span></div>
    </aside>
    <div className="main-area">
      <header className="app-top">
        <details key={pathname} className="mobile-menu" onKeyDown={event => {
          if (event.key === "Escape") {
            event.currentTarget.open = false;
            event.currentTarget.querySelector("summary")?.focus();
          }
        }}>
          <summary aria-label="Abrir menú principal" title="Menú principal"><Icon name="menu" size={22}/></summary>
          <div className="mobile-menu-panel">
            <nav>{navigation.map(item => <Link key={item.href} className={isActivePath(item.href, pathname) ? "active" : ""} href={item.href}>
              <Icon name={item.icon}/><span>{item.label}</span>{item.badge && <em>{item.badge}</em>}<Icon name="chevron" size={16}/>
            </Link>)}</nav>
            <div className="mobile-menu-section">GESTIÓN</div>
            <nav>{managementNav.map(item => <Link key={item.href} className={isActivePath(item.href, pathname) ? "active" : ""} href={item.href}>
              <Icon name={item.icon}/><span>{item.label}</span><Icon name="chevron" size={16}/>
            </Link>)}</nav>
          </div>
        </details>
        <div className="mobile-brand"><Logo /></div>
        <div className="top-context"><span className="context-icon"><Icon name="store" size={18}/></span><div><small>LOCAL ACTIVO</small><Select className="context-select" aria-label="Local activo" defaultValue="miraflores"><option value="miraflores">Sabor Criollo · Miraflores</option></Select></div></div>
        <strong className="mobile-title">{title}</strong>
        <div className="top-tools">
          <button className="quick-search" aria-label="Búsqueda rápida"><Icon name="search"/><span>Buscar</span><kbd>⌘ K</kbd></button>
          <div className="sync-state"><i><Icon name="wifi" size={14}/></i><span><b>En línea</b><small>Sincronizado ahora</small></span></div>
          <div className="shift-state"><Icon name="clock" size={16}/><span><small>TURNO ACTIVO</small><b>04 h 32 min</b></span></div>
          <button className="notification-button" aria-label="Notificaciones"><Icon name="bell"/><i/><em>3</em></button>
          <div className="profile-menu">
            <button className="profile-control" type="button" aria-label="Abrir opciones de Rogelio" aria-haspopup="menu" aria-expanded={profileOpen} onClick={() => setProfileOpen(value => !value)}><span className="avatar">RF</span><span><b>Rogelio</b><small>Administrador</small></span><Icon name="chevron" size={15}/></button>
            {profileOpen && <div className="profile-popover" role="menu">
              <header><span className="avatar">RF</span><span><b>Rogelio Fernández</b><small>Administrador</small></span></header>
              <button type="button" role="menuitem" className="profile-signout" disabled={isSigningOut} onClick={signOut}><Icon name="logout" size={19}/><span>{isSigningOut ? "Cerrando sesión…" : "Cerrar sesión"}</span>{isSigningOut && <i aria-hidden="true"/>}</button>
            </div>}
          </div>
        </div>
      </header>
      <main className="app-content">{pathname.startsWith("/pos") && <SaleFlow pathname={pathname}/>} {children}</main>
      <nav className="bottom-nav" aria-label="Navegación principal">
        {bottomNavigation.map(item => {
          const active = isActivePath(item.href, pathname);
          return <Link key={item.href} href={item.href} className={`${active ? "active " : ""}${item.floating ? "floating-pos" : ""}`} aria-current={active ? "page" : undefined}>
            <span className="bottom-icon"><Icon name={item.icon} size={item.floating ? 24 : 21}/>{item.badge && <i>{item.badge}</i>}</span>
            <small>{item.label}</small>
          </Link>;
        })}
      </nav>
    </div>
  </div>;
}

function SaleFlow({pathname}:{pathname:string}) {
  const steps=[
    {href:"/pos",number:"01",label:"Pedido",detail:"Productos y cantidades",icon:"orders" as const},
    {href:"/pos/personalizar",number:"02",label:"Personalizar",detail:"Opciones de cocina",icon:"kitchen" as const},
    {href:"/pos/pago",number:"03",label:"Pago",detail:"Medio y vuelto",icon:"card" as const},
    {href:"/pos/comprobante",number:"04",label:"Comprobante",detail:"Boleta o factura",icon:"receipt" as const},
  ];
  const active=Math.max(0,steps.findIndex(step=>step.href===pathname));
  return <nav className="premium-wizard" aria-label="Flujo de venta">
    <header><span>ESPACIO DE VENTA</span><strong>{steps[active].label}</strong><small>Pedido · Cobro · Emisión</small></header>
    <div className="wizard-track">{steps.map((step,index)=><Link href={step.href} aria-current={index===active?"step":undefined} className={index===active?"active":""} key={step.href}>
      <span className="wizard-icon"><Icon name={step.icon} size={20}/></span>
      <span className="wizard-copy"><small>PASO {step.number}</small><strong>{step.label}</strong><em>{step.detail}</em></span>
    </Link>)}</div>
  </nav>;
}

export function PageHeading({ eyebrow, title, description, action }: { eyebrow?: string; title: string; description: string; action?: React.ReactNode }) {
  return <div className="page-heading"><div>{eyebrow && <span>{eyebrow}</span>}<h1>{title}</h1><p>{description}</p></div>{action}</div>;
}
