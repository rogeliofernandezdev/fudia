"use client";
import "./configuration.css";

import Link from "next/link";
import {Icon,type IconName} from "@/design-system/icons";
import {PageHeader} from "@/design-system/page-header";
import {useSession} from "@/providers";

const sections:Array<{icon:IconName;title:string;description:string;href:string;access?:string;platformAdminOnly?:boolean}>=[
  {icon:"receipt",title:"País y configuración fiscal",description:"Perfiles, monedas, impuestos y tasas",href:"/configuracion/fiscal"},
  {icon:"users",title:"Usuarios y permisos",description:"Equipo, roles y accesos",href:"/configuracion/usuarios",access:"usuarios"},
  {icon:"receipt",title:"Facturación electrónica",description:"Series y comprobantes",href:"/configuracion/facturacion"},
  {icon:"settings",title:"Módulos",description:"Activa o desactiva módulos por empresa",href:"/configuracion/modulos",platformAdminOnly:true},
  {icon:"settings",title:"Integraciones",description:"WhatsApp, pagos e impresión",href:"/configuracion/integraciones"},
];

export function ConfigurationHomePage(){
  const{user,canAccess}=useSession();
  const visible=sections.filter(section=>section.platformAdminOnly?Boolean(user?.platformAdmin):section.access?canAccess(section.access):true);
  return <><PageHeader eyebrow="CONFIGURACIÓN" title="Configuración" description="Centraliza la estructura, seguridad e integraciones de tu empresa."/>
    <section className="settings-grid">
      {visible.map(section=><Link key={section.title} href={section.href}>
        <span><Icon name={section.icon}/></span>
        <b>{section.title}<small>{section.description}</small></b>
        <Icon name="chevron"/>
      </Link>)}
    </section>
  </>;
}
