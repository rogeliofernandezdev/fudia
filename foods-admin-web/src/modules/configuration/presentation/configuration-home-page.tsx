"use client";
import "./configuration.css";
import Link from "next/link";
import {Icon,type IconName} from "@/design-system/icons";
import {PageHeader} from "@/design-system/page-header";
import {useSession} from "@/providers";

const sections:Array<{icon:IconName;title:string;description:string;href:string;permission?:string;platformAdminOnly?:boolean}>=[
  {icon:"store",title:"Empresa",description:"Razón social, nombre comercial e identidad general",href:"/configuracion/empresa",permission:"organizations.read"},
  {icon:"receipt",title:"País y configuración fiscal",description:"Perfiles, monedas, impuestos y tasas",href:"/configuracion/fiscal",permission:"organizations.read"},
  {icon:"users",title:"Usuarios y permisos",description:"Equipo, roles y accesos por local",href:"/configuracion/usuarios",permission:"users.read"},
  {icon:"settings",title:"Módulos",description:"Control de módulos contratado por empresa",href:"/configuracion/modulos",platformAdminOnly:true},
];

export function ConfigurationHomePage(){
  const{user,can}=useSession();
  const visible=sections.filter(section=>section.platformAdminOnly?Boolean(user?.platformAdmin):section.permission?can(section.permission):true);
  return <><PageHeader eyebrow="CONFIGURACIÓN" title="Configuración" description="Administra la estructura y los accesos necesarios para operar el restaurante."/>
    <section className="settings-grid">
      {visible.map(section=><Link key={section.title} href={section.href}>
        <span><Icon name={section.icon}/></span>
        <b>{section.title}<small>{section.description}</small></b>
        <Icon name="chevron"/>
      </Link>)}
    </section>
  </>;
}
