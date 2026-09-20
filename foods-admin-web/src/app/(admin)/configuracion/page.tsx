"use client";
import Link from "next/link";
import {Icon,IconName} from "@/design-system/icons";
import {PageHeader} from "@/design-system/page-header";

const modules:Array<{i:IconName;t:string;d:string;href:string}>=[
  {i:"receipt",t:"País y configuración fiscal",d:"Perfiles, monedas, impuestos y tasas",href:"/configuracion/fiscal"},
  {i:"users",t:"Usuarios y permisos",d:"Equipo, roles y accesos",href:"/configuracion/usuarios"},
  {i:"receipt",t:"Facturación electrónica",d:"Series y comprobantes",href:"/configuracion/facturacion"},
  {i:"settings",t:"Módulos",d:"Activa o desactiva módulos por empresa",href:"/configuracion/modulos"},
  {i:"settings",t:"Integraciones",d:"WhatsApp, pagos e impresión",href:"/configuracion/integraciones"},
];

export default function Page(){
  return <><PageHeader eyebrow="CONFIGURACIÓN" title="Configuración" description="Centraliza la estructura, seguridad e integraciones de tu empresa."/>
  <section className="settings-grid">{modules.map(x=><Link key={x.t} href={x.href}><span><Icon name={x.i}/></span><b>{x.t}<small>{x.d}</small></b><Icon name="chevron"/></Link>)}</section>
  </>;
}
