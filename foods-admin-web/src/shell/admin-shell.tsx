"use client";
import "./styles/account-menu.css";
import "./styles/navigation-state.css";
import "./styles/shell.css";
import Link from "next/link";
import {usePathname,useRouter} from "next/navigation";
import {useEffect,useRef,useState} from "react";
import {useQueryClient} from "@tanstack/react-query";
import {FullScreenLoader} from "@/design-system";
import {Icon} from "@/design-system/icons";
import {Logo} from "@/design-system/logo";
import {useSession} from "@/providers/session-context";
import {ContextSwitcher} from "@/modules/context";
import {RestaurantSetupTour} from "@/modules/setup";
import {pageRoutes} from "@/shared/routing/page-routes";
import {deleteSession} from "@/shared/session/session-api";
import {broadcastSessionChange} from "@/shared/session/session-events";
import {accessKey,firstAccessibleRoute,moduleIsActive,navigationItemForPath,visibleNavigation} from "./navigation";
import {AdminSessionError} from "./admin-session-error";
import {NavigationSearch} from "./navigation-search";

export function AdminShell({children}:{children:React.ReactNode}){
 const path=usePathname();const router=useRouter();const queryClient=useQueryClient();const{user,organization,location,modules,menuAccess,permissions,setupRequired,canAccess,can,isLoading,isError,isUnauthorized}=useSession();const[menuOpen,setMenuOpen]=useState(false);const[collapsed,setCollapsed]=useState(false);const[accountOpen,setAccountOpen]=useState(false);const[signingOut,setSigningOut]=useState(false);const accountRef=useRef<HTMLDivElement>(null);
 const isPlatformAdmin=Boolean(user?.platformAdmin);
 const navigationContext=user&&modules?{user,modules,menuAccess,permissions,setupRequired}:null;
 const visibleGroups=navigationContext?visibleNavigation(navigationContext):[];
 const homeHref=navigationContext?firstAccessibleRoute(navigationContext):pageRoutes.noAccess;
 const currentItem=navigationItemForPath(path);
 const blockReason=isPlatformAdmin?null:!currentItem?null:currentItem.platformAdminOnly?"platform":!modules||!moduleIsActive(modules,currentItem.module)?"module":!canAccess(accessKey(currentItem))||Boolean(currentItem.permission&&!can(currentItem.permission))?"role":null;
 async function logout(){if(signingOut)return;setSigningOut(true);try{await deleteSession()}finally{broadcastSessionChange("signed-out");queryClient.clear();router.replace(pageRoutes.login);router.refresh()}}
 useEffect(()=>{if(!menuOpen)return;const previous=document.body.style.overflow;document.body.style.overflow="hidden";const close=(event:KeyboardEvent)=>{if(event.key==="Escape")setMenuOpen(false)};document.addEventListener("keydown",close);return()=>{document.body.style.overflow=previous;document.removeEventListener("keydown",close)}},[menuOpen]);
 useEffect(()=>{if(!accountOpen)return;const close=(event:MouseEvent)=>{if(!accountRef.current?.contains(event.target as Node))setAccountOpen(false)};const escape=(event:KeyboardEvent)=>{if(event.key==="Escape")setAccountOpen(false)};document.addEventListener("mousedown",close);document.addEventListener("keydown",escape);return()=>{document.removeEventListener("mousedown",close);document.removeEventListener("keydown",escape)}},[accountOpen]);
 if(isUnauthorized)return <FullScreenLoader label="Tu sesión finalizó"/>;
 if(isError)return <AdminSessionError fullPage icon="alert" eyebrow="SESIÓN NO DISPONIBLE" title="No pudimos cargar tu sesión" description="Comprueba tu conexión e inténtalo nuevamente. Si el problema continúa, vuelve al inicio de sesión."><button className="admin-session-error-primary" type="button" onClick={()=>window.location.reload()}><Icon name="refresh" size={17}/>Reintentar</button><button className="admin-session-error-secondary" type="button" disabled={signingOut} onClick={()=>void logout()}><Icon name="logout" size={17}/>{signingOut?"Cerrando sesión…":"Volver al inicio de sesión"}</button></AdminSessionError>;
 if(isLoading||!user||!location||!modules||!menuAccess)return <FullScreenLoader label="Preparando tu espacio"/>;
 return <div className="admin-shell" data-sidebar={collapsed?"collapsed":"expanded"}>
  {menuOpen&&<button className="sidebar-scrim" aria-label="Cerrar navegación" onClick={()=>setMenuOpen(false)}/>} 
  <aside className={`sidebar${menuOpen?" open":""}`} aria-label="Navegación principal"><div className="sidebar-brand"><Link href={homeHref} onClick={()=>setMenuOpen(false)}><Logo inverse/></Link><button className="sidebar-close" onClick={()=>setMenuOpen(false)} aria-label="Cerrar navegación"><Icon name="close"/></button></div><nav>{visibleGroups.map(group=><section key={group.label}><small>{group.label}</small>{group.items.map(item=><Link key={item.href} href={item.href} onClick={()=>setMenuOpen(false)} className={path.startsWith(item.href)?"active":""} aria-current={path.startsWith(item.href)?"page":undefined}><span className="nav-icon"><Icon name={item.icon}/></span><span>{item.name}</span></Link>)}</section>)}</nav></aside>
  <main><header className="topbar"><button className="mobile-nav" onClick={()=>setMenuOpen(true)} aria-label="Abrir navegación" aria-expanded={menuOpen}><Icon name="menu"/></button><button className="sidebar-toggle" type="button" onClick={()=>setCollapsed(value=>!value)} aria-label={collapsed?"Expandir navegación":"Contraer navegación"} aria-expanded={!collapsed}><Icon name="menu"/></button><NavigationSearch groups={visibleGroups}/><div className="context"><ContextSwitcher/>{user?.platformAdmin&&<Link href={pageRoutes.platformOnboarding} className="platform-link"><Icon name="plus" size={16}/>Nueva empresa</Link>}<div className="account-menu" ref={accountRef}><button className="account-trigger" type="button" aria-label={`Abrir opciones de ${user?.name??"usuario"}`} aria-haspopup="menu" aria-expanded={accountOpen} onClick={()=>{setAccountOpen(value=>!value)}}><span className="account-avatar">{(user?.name??"U").slice(0,2).toUpperCase()}</span><span className="account-trigger-copy"><b>{user?.name??"Usuario"}</b><small>{user?.platformAdmin?"Administrador de plataforma":organization?.name??"Usuario"}</small></span><Icon name="chevron" size={15}/></button>{accountOpen&&<section className="account-popover" role="menu"><header><span className="account-avatar">{(user?.name??"U").slice(0,2).toUpperCase()}</span><span><b>{user?.name??"Usuario"}</b><small>{user?.platformAdmin?"Administrador de plataforma":organization?.name??"Usuario"}</small></span></header><nav><Link href={pageRoutes.profileSettings} role="menuitem" onClick={()=>setAccountOpen(false)}><Icon name="users" size={18}/><span><b>Mi perfil y seguridad</b><small>Datos personales y contraseña</small></span><Icon name="chevron" size={14}/></Link></nav><button className="account-logout" type="button" role="menuitem" disabled={signingOut} onClick={logout}><Icon name="logout" size={18}/><span>{signingOut?"Cerrando sesión…":"Cerrar sesión"}</span>{signingOut&&<i aria-hidden="true"/>}</button></section>}</div></div></header><div className="content">{setupRequired&&!isPlatformAdmin&&path!==pageRoutes.gettingStarted&&can("organizations.manage")&&<Link className="setup-reminder" href={pageRoutes.gettingStarted}><Icon name="arrowRightCircle" size={20}/><span><b>Configuración inicial pendiente</b><small>Continúa la guía para preparar tu restaurante.</small></span><Icon name="chevron" size={16}/></Link>}{!blockReason?children:<AdminSessionError icon="lock" title={blockReason==="module"?"Módulo no contratado":blockReason==="platform"?"Acceso de plataforma":"Acceso no habilitado"} description={blockReason==="module"?`Este módulo no está habilitado para ${organization?.name??"tu empresa"}. Contacta a tu proveedor para activarlo.`:blockReason==="platform"?"Esta opción es exclusiva del administrador de la plataforma.":"Tu rol no incluye esta opción del sistema."}><button className="admin-session-error-primary" type="button" onClick={()=>router.push(homeHref)}><Icon name="grid" size={16}/>Ir al inicio</button></AdminSessionError>}</div></main>
  {!blockReason&&<RestaurantSetupTour/>}
 </div>;
}
