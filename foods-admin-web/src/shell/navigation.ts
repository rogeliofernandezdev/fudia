import type {IconName} from "@/design-system/icons";
import type {SessionContextResponse} from "@/shared/session/session-api";

export type NavItem={
  href:string;
  name:string;
  icon:IconName;
  module:string;
  access?:string;
  platformAdminOnly?:boolean;
  permission?:string;
};
export type NavGroup={label:string;items:NavItem[]};

export const navigationGroups:NavGroup[]=[
 {label:"CONTROL",items:[{href:"/dashboard",name:"Reportes",icon:"grid",module:"reportes",access:"dashboard",permission:"dashboard.read"},{href:"/ventas",name:"Ventas",icon:"receipt",module:"reportes",access:"dashboard",permission:"orders.read"}]},
 {label:"OPERACIÓN",items:[{href:"/pos",name:"Punto de venta",icon:"sales",module:"pos",permission:"orders.read"},{href:"/pedidos",name:"Pedidos",icon:"receipt",module:"pedidos",permission:"orders.read"},{href:"/salon",name:"Salón",icon:"utensils",module:"pedidos",permission:"orders.read"},{href:"/cocina",name:"Cocina",icon:"kitchen",module:"cocina",permission:"orders.read"},{href:"/mesas",name:"Mesas y zonas",icon:"grid",module:"mesas",permission:"tables.read"},{href:"/caja",name:"Caja y turnos",icon:"sales",module:"caja",permission:"cash.read"},{href:"/reservas",name:"Reservas",icon:"clock",module:"reservas",permission:"reservations.read"},{href:"/call-center",name:"Call center",icon:"receipt",module:"call_center"},{href:"/carta-qr",name:"Carta digital QR",icon:"qr",module:"carta_qr"},{href:"/kiosco",name:"Kiosco de autoservicio",icon:"grid",module:"kiosco"}]},
 {label:"CARTA Y PRODUCCIÓN",items:[{href:"/productos",name:"Carta y productos",icon:"utensils",module:"productos",permission:"menu.read"},{href:"/disponibilidad",name:"Disponibilidad",icon:"availability",module:"productos",access:"disponibilidad",permission:"menu.read"},{href:"/combos",name:"Menús y combos",icon:"combo",module:"combos",permission:"menu.read"},{href:"/recetas",name:"Recetas",icon:"cookingPot",module:"recetas",permission:"menu.read"}]},
 {label:"ABASTECIMIENTO",items:[{href:"/inventario",name:"Inventario",icon:"stock",module:"inventario",permission:"inventory.read"},{href:"/kardex",name:"Kardex",icon:"ledger",module:"kardex",permission:"inventory.read"},{href:"/compras",name:"Compras",icon:"truck",module:"compras",permission:"purchases.read"},{href:"/logistica",name:"Logística",icon:"truck",module:"logistica"}]},
 {label:"DELIVERY",items:[{href:"/delivery",name:"Delivery propio",icon:"truck",module:"delivery"},{href:"/delivery-apps",name:"Apps de delivery",icon:"settings",module:"delivery_apps"},{href:"/repartidores",name:"App repartidores",icon:"truck",module:"repartidores"}]},
 {label:"NEGOCIO",items:[{href:"/clientes",name:"Clientes",icon:"users",module:"clientes",permission:"customers.read"},{href:"/crm",name:"CRM y fidelización",icon:"users",module:"crm"},{href:"/puntos",name:"Plaza puntos",icon:"users",module:"puntos"},{href:"/ofertas",name:"Ofertas y descuentos",icon:"sales",module:"ofertas"},{href:"/personal",name:"Personal y asistencias",icon:"users",module:"personal"},{href:"/locales",name:"Locales",icon:"store",module:"locales",permission:"organizations.read"}]},
 {label:"INTELIGENCIA",items:[{href:"/costos",name:"Costos y gastos",icon:"stock",module:"costos",permission:"expenses.read"},{href:"/bi",name:"Restaurant BI",icon:"grid",module:"bi"},{href:"/app-manager",name:"App manager",icon:"settings",module:"app_manager"}]},
 {label:"CONFIGURACIÓN",items:[{href:"/configuracion/empresa",name:"Empresa",icon:"store",module:"locales",access:"locales",permission:"organizations.read"},{href:"/configuracion/fiscal",name:"Fiscal y moneda",icon:"receipt",module:"fiscal",permission:"organizations.read"},{href:"/configuracion/medios-pago",name:"Medios de pago",icon:"payment",module:"fiscal",access:"fiscal",permission:"organizations.read"},{href:"/configuracion/usuarios",name:"Usuarios y roles",icon:"users",module:"usuarios",permission:"users.read"},{href:"/configuracion/facturacion",name:"Facturación",icon:"receipt",module:"facturacion"},{href:"/configuracion/modulos",name:"Módulos",icon:"settings",module:"fiscal",platformAdminOnly:true},{href:"/configuracion/integraciones",name:"Integraciones",icon:"settings",module:"integraciones"},{href:"/whatsapp-bot",name:"WhatsApp IA para pedidos",icon:"settings",module:"whatsapp_bot"}]}
];

type AccessContext=Pick<SessionContextResponse,"user"|"modules"|"menuAccess"|"permissions">;

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
  if(!(context.menuAccess.includes("*")||context.menuAccess.includes(accessKey(item))))return false;
  if(item.permission&&!(context.permissions.includes("*")||context.permissions.includes(item.permission)))return false;
  return true;
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
