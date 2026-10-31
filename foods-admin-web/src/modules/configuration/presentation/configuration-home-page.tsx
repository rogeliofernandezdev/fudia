"use client";
import "./configuration.css";
import Link from "next/link";
import {Icon} from "@/design-system/icons";
import {PageHeader} from "@/design-system/page-header";
import {useSession} from "@/providers";
import {navigationItemForPath} from "@/shell/navigation";
import {pageRoutes} from "@/shared/routing/page-routes";

const sections:Array<{title:string;description:string;href:string;permission?:string;platformAdminOnly?:boolean;module?:string;access?:string}>=[
  {title:"Mi restaurante",description:"Razón social, nombre comercial e identidad general",href:pageRoutes.organizationSettings,permission:"organizations.manage"},
  {title:"Fiscal y moneda",description:"Perfiles, monedas, impuestos y tasas",href:pageRoutes.taxSettings,permission:"organizations.read"},
  {title:"Medios de pago",description:"Catálogo único para ventas, caja y gastos",href:pageRoutes.paymentMethods,permission:"organizations.read"},
  {title:"Usuarios y roles",description:"Equipo, roles y accesos por local",href:pageRoutes.usersSettings,permission:"users.read"},
  {title:"Módulos",description:"Control de módulos contratado por empresa",href:pageRoutes.modulesSettings,platformAdminOnly:true},
  {title:"Empresas y planes",description:"Plan contratado y situación de pago de cada empresa cliente",href:pageRoutes.companiesSettings,platformAdminOnly:true},
  {title:"Fudia Concierge",description:"Pedidos conversacionales por WhatsApp desde el QR",href:pageRoutes.concierge,permission:"organizations.read",module:"whatsapp_bot"},
];

export function ConfigurationHomePage(){
  const{user,can,canAccess,modules}=useSession();
  const visible=sections.filter(section=>{
    if(section.platformAdminOnly)return Boolean(user?.platformAdmin);
    if(section.module&&!user?.platformAdmin&&!modules?.[section.module])return false;
    if(section.access&&!user?.platformAdmin&&!canAccess(section.access))return false;
    return section.permission?can(section.permission):true;
  });
  return <><PageHeader title="Configuración" description="Administra la estructura y los accesos necesarios para operar el restaurante."/>
    <section className="settings-grid">
      {visible.map(section=><Link key={section.title} href={section.href}><span><Icon name={navigationItemForPath(section.href)?.icon??"settings"}/></span><b>{section.title}<small>{section.description}</small></b><Icon name="chevron"/></Link>)}
    </section>
  </>;
}
