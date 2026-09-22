import type {IconName} from "@/design-system/icons";
import type {SessionContextResponse} from "@/shared/session/session-api";

export type NavItem={
  href:string;
  name:string;
  icon:IconName;
  module:string;
  access?:string;
  platformAdminOnly?:boolean;
};
export type NavGroup={label:string;items:NavItem[]};

export const navigationGroups:NavGroup[]=[
 {label:"CONTROL",items:[{href:"/dashboard",name:"Reportes",icon:"grid",module:"reportes",access:"dashboard"},{href:"/ventas",name:"Ventas",icon:"receipt",module:"reportes",access:"dashboard"}]},
 {label:"OPERACIÓN",items:[{href:"/pos",name:"Punto de venta",icon:"sales",module:"pos"},{href:"/pedidos",name:"Pedidos",icon:"receipt",module:"pedidos"},{href:"/salon",name:"Salón",icon:"utensils",module:"pedidos"},{href:"/cocina",name:"Cocina",icon:"kitchen",module:"cocina"},{href:"/mesas",name:"Mesas y zonas",icon:"grid",module:"mesas"},{href:"/caja",name:"Caja y turnos",icon:"sales",module:"caja"},{href:"/reservas",name:"Reservas",icon:"clock",module:"reservas"},{href:"/call-center",name:"Call center",icon:"receipt",module:"call_center"},{href:"/carta-qr",name:"Carta digital QR",icon:"qr",module:"carta_qr"},{href:"/kiosco",name:"Kiosco de autoservicio",icon:"grid",module:"kiosco"}]},
 {label:"CARTA Y PRODUCCIÓN",items:[{href:"/productos",name:"Carta y productos",icon:"utensils",module:"productos"},{href:"/disponibilidad",name:"Disponibilidad",icon:"availability",module:"productos",access:"disponibilidad"},{href:"/combos",name:"Menús y combos",icon:"combo",module:"combos"},{href:"/recetas",name:"Recetas",icon:"cookingPot",module:"recetas"}]},
 {label:"ABASTECIMIENTO",items:[{href:"/inventario",name:"Inventario",icon:"stock",module:"inventario"},{href:"/kardex",name:"Kardex",icon:"ledger",module:"kardex"},{href:"/compras",name:"Compras",icon:"truck",module:"compras"},{href:"/logistica",name:"Logística",icon:"truck",module:"logistica"}]},
 {label:"DELIVERY",items:[{href:"/delivery",name:"Delivery propio",icon:"truck",module:"delivery"},{href:"/delivery-apps",name:"Apps de delivery",icon:"settings",module:"delivery_apps"},{href:"/repartidores",name:"App repartidores",icon:"truck",module:"repartidores"}]},
 {label:"NEGOCIO",items:[{href:"/clientes",name:"Clientes",icon:"users",module:"clientes"},{href:"/crm",name:"CRM y fidelización",icon:"users",module:"crm"},{href:"/puntos",name:"Plaza puntos",icon:"users",module:"puntos"},{href:"/ofertas",name:"Ofertas y descuentos",icon:"sales",module:"ofertas"},{href:"/personal",name:"Personal y asistencias",icon:"users",module:"personal"},{href:"/locales",name:"Locales",icon:"store",module:"locales"}]},
 {label:"INTELIGENCIA",items:[{href:"/costos",name:"Costos y gastos",icon:"stock",module:"costos"},{href:"/bi",name:"Restaurant BI",icon:"grid",module:"bi"},{href:"/app-manager",name:"App manager",icon:"settings",module:"app_manager"}]},
 {label:"CONFIGURACIÓN",items:[{href:"/configuracion/empresa",name:"Empresa",icon:"store",module:"locales",access:"locales"},{href:"/configuracion/fiscal",name:"Fiscal y moneda",icon:"receipt",module:"fiscal"},{href:"/configuracion/usuarios",name:"Usuarios y roles",icon:"users",module:"usuarios"},{href:"/configuracion/facturacion",name:"Facturación",icon:"receipt",module:"facturacion"},{href:"/configuracion/modulos",name:"Módulos",icon:"settings",module:"fiscal",platformAdminOnly:true},{href:"/configuracion/integraciones",name:"Integraciones",icon:"settings",module:"integraciones"},{href:"/whatsapp-bot",name:"WhatsApp IA para pedidos",icon:"settings",module:"whatsapp_bot"}]}
];

type AccessContext=Pick<SessionContextResponse,"user"|"modules"|"menuAccess">;

export function accessKey(item:NavItem){return item.access??item.module}

export function moduleIsActive(modules:Record<string,boolean>,key:string){
  if(!modules[key])return false;
  if(key==="recetas")return modules.inventario!==false;
  return true;
}

export function canOpenNavigationItem(item:NavItem,context:AccessContext){
  if(context.user.platformAdmin)return true;
  if(item.platformAdminOnly)return false;
  if(!moduleIsActive(context.modules,item.module))return false;
  return context.menuAccess.includes("*")||context.menuAccess.includes(accessKey(item));
}

export function visibleNavigation(context:AccessContext){
  return navigationGroups
    .map(group=>({...group,items:group.items.filter(item=>canOpenNavigationItem(item,context))}))
    .filter(group=>group.items.length>0);
}

export function firstAccessibleRoute(context:AccessContext){
  return visibleNavigation(context)[0]?.items[0]?.href??"/sin-acceso";
}

export function navigationItemForPath(path:string){
  return navigationGroups
    .flatMap(group=>group.items)
    .sort((a,b)=>b.href.length-a.href.length)
    .find(item=>path===item.href||path.startsWith(item.href+"/"));
}
