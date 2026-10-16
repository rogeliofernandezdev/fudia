import type {IconName} from "@/design-system/icons";
import {pageRoutes} from "@/shared/routing/page-routes";
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
 {label:"CONTROL",items:[{href:pageRoutes.dashboard,name:"Dashboard",icon:"gauge",module:"reportes",access:"dashboard",permission:"dashboard.read"},{href:pageRoutes.sales,name:"Ventas",icon:"sales",module:"reportes",access:"dashboard",permission:"orders.read"}]},
 {label:"OPERACIÓN",items:[{href:pageRoutes.pos,name:"Punto de venta",icon:"register",module:"pos",permission:"orders.read"},{href:pageRoutes.orders,name:"Pedidos",icon:"orders",module:"pedidos",permission:"orders.read"},{href:pageRoutes.diningRoom,name:"Salón",icon:"tables",module:"pedidos",permission:"orders.read"},{href:pageRoutes.kitchen,name:"Cocina",icon:"kitchen",module:"cocina",permission:"orders.read"},{href:pageRoutes.tables,name:"Mesas y zonas",icon:"layout",module:"mesas",permission:"tables.read"},{href:pageRoutes.cash,name:"Caja y turnos",icon:"cash",module:"caja",permission:"cash.read"},{href:pageRoutes.reservations,name:"Reservas",icon:"calendar",module:"reservas",permission:"reservations.read"},{href:pageRoutes.callCenter,name:"Call center",icon:"phone",module:"call_center"},{href:pageRoutes.qrMenu,name:"Carta digital QR",icon:"qr",module:"carta_qr"},{href:pageRoutes.kiosk,name:"Kiosco de autoservicio",icon:"kiosk",module:"kiosco"}]},
 {label:"CARTA Y PRODUCCIÓN",items:[{href:pageRoutes.products,name:"Carta y productos",icon:"utensils",module:"productos",permission:"menu.read"},{href:pageRoutes.availability,name:"Disponibilidad",icon:"availability",module:"productos",access:"disponibilidad",permission:"menu.read"},{href:pageRoutes.combos,name:"Menús y combos",icon:"combo",module:"combos",permission:"menu.read"},{href:pageRoutes.recipes,name:"Recetas",icon:"cookingPot",module:"recetas",permission:"menu.read"}]},
 {label:"ABASTECIMIENTO",items:[{href:pageRoutes.inventory,name:"Inventario",icon:"stock",module:"inventario",permission:"inventory.read"},{href:pageRoutes.stockLedger,name:"Kardex",icon:"ledger",module:"kardex",permission:"inventory.read"},{href:pageRoutes.purchases,name:"Compras",icon:"cart",module:"compras",permission:"purchases.read"},{href:pageRoutes.logistics,name:"Logística",icon:"route",module:"logistica"}]},
 {label:"DELIVERY",items:[{href:pageRoutes.delivery,name:"Delivery propio",icon:"truck",module:"delivery"},{href:pageRoutes.deliveryApps,name:"Apps de delivery",icon:"link",module:"delivery_apps"},{href:pageRoutes.couriers,name:"App repartidores",icon:"bike",module:"repartidores"}]},
 {label:"NEGOCIO",items:[{href:pageRoutes.customers,name:"Clientes",icon:"user",module:"clientes",permission:"customers.read"},{href:pageRoutes.crm,name:"CRM y fidelización",icon:"heart",module:"crm"},{href:pageRoutes.loyaltyPoints,name:"Plaza puntos",icon:"star",module:"puntos"},{href:pageRoutes.offers,name:"Ofertas y descuentos",icon:"tag",module:"ofertas"},{href:pageRoutes.staff,name:"Personal y asistencias",icon:"badge",module:"personal"},{href:pageRoutes.locations,name:"Locales",icon:"store",module:"locales",permission:"organizations.read"}]},
 {label:"INTELIGENCIA",items:[{href:pageRoutes.expenses,name:"Costos y gastos",icon:"wallet",module:"costos",permission:"expenses.read"},{href:pageRoutes.businessIntelligence,name:"Restaurant BI",icon:"pie",module:"bi"},{href:pageRoutes.appManager,name:"App manager",icon:"phoneApp",module:"app_manager"}]},
 {label:"CONFIGURACIÓN",items:[{href:pageRoutes.gettingStarted,name:"Puesta en marcha",icon:"check",module:"locales",access:"locales",permission:"organizations.read"},{href:pageRoutes.organizationSettings,name:"Empresa",icon:"building",module:"locales",access:"locales",permission:"organizations.read"},{href:pageRoutes.taxSettings,name:"Fiscal y moneda",icon:"percent",module:"fiscal",permission:"organizations.read"},{href:pageRoutes.paymentMethods,name:"Medios de pago",icon:"payment",module:"fiscal",access:"fiscal",permission:"organizations.read"},{href:pageRoutes.usersSettings,name:"Usuarios y roles",icon:"users",module:"usuarios",permission:"users.read"},{href:pageRoutes.billingSettings,name:"Facturación",icon:"receipt",module:"facturacion"},{href:pageRoutes.modulesSettings,name:"Módulos",icon:"layers",module:"fiscal",platformAdminOnly:true},{href:pageRoutes.companiesSettings,name:"Empresas y planes",icon:"contract",module:"locales",platformAdminOnly:true},{href:pageRoutes.platformGlobalSettings,name:"Países y WhatsApp",icon:"chat",module:"integraciones",platformAdminOnly:true},{href:pageRoutes.integrationsSettings,name:"Integraciones",icon:"plug",module:"integraciones"},{href:pageRoutes.concierge,name:"Fudia Concierge",icon:"chat",module:"whatsapp_bot",permission:"organizations.read"}]}
];

type AccessContext=Pick<SessionContextResponse,"user"|"modules"|"menuAccess"|"permissions"|"setupRequired">;

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
  if(!context.user.platformAdmin&&context.setupRequired&&(context.permissions.includes("*")||context.permissions.includes("organizations.manage")))return pageRoutes.gettingStarted;
  return visibleNavigation(context)[0]?.items[0]?.href??pageRoutes.noAccess;
}

export function navigationItemForPath(path:string){
  return navigationGroups
    .flatMap(group=>group.items)
    .sort((a,b)=>b.href.length-a.href.length)
    .find(item=>path===item.href||path.startsWith(item.href+"/"));
}
