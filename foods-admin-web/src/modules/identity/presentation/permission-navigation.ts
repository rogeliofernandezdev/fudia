import {accessKey,navigationGroups} from "@/shell/navigation";
import {pageRoutes as routes} from "@/shared/routing/page-routes";
import type {IconName} from "@/design-system/icons";
import type {PermissionGroup} from "../domain/types";

export type PermissionModule={id:string;name:string;icon:IconName;items:PermissionGroup["items"]};
export type PermissionSection={name:string;modules:PermissionModule[]};

// Presentation bindings only: the API remains the source of actions and labels.
// Shared permissions are shown at their entry points and keep a single value.
const actionRoutes:Record<string,readonly string[]>={
 dashboard:[routes.dashboard],reports:[routes.dashboard],audit:[routes.dashboard],
 orders:[routes.pos,routes.orders,routes.diningRoom],kitchen:[routes.kitchen],bar:[routes.kitchen],
 tables:[routes.tables],reservations:[routes.reservations],cash:[routes.cash],
 "cash.manage":[routes.pos,routes.cash],receipts:[routes.receipts],
 menu:[routes.products,routes.availability,routes.combos],recipes:[routes.recipes],
 inventory:[routes.inventory],"inventory.read":[routes.inventory,routes.stockLedger],
 purchases:[routes.purchases],delivery:[routes.delivery,routes.couriers],
 customers:[routes.customers],expenses:[routes.expenses],users:[routes.usersSettings],
 organizations:[routes.locations,routes.organizationSettings,routes.concierge],
 fiscal:[routes.taxSettings],subscription:[routes.billingSettings],
};

export function permissionSections(groups:PermissionGroup[],menuGroups?:PermissionGroup[]):PermissionSection[]{
 const allowedMenus=menuGroups?new Set(menuGroups.flatMap(group=>group.items.map(item=>item.value))):undefined;
 const actions=[...new Map(groups.flatMap(group=>group.items).map(item=>[item.value,item])).values()];
 const placed=new Set<string>();
 const sections=navigationGroups.flatMap(section=>{
  const modules=section.items.flatMap(module=>{
   if(allowedMenus&&!allowedMenus.has(accessKey(module)))return [];
   const items=actions.filter(item=>item.value===module.permission||(actionRoutes[item.value]??actionRoutes[item.value.split(".")[0]]??[]).includes(module.href));
   if(!items.length)return [];
   items.forEach(item=>placed.add(item.value));
   return [{id:module.href,name:module.name,icon:module.icon,items}];
  });
  return modules.length?[{name:section.label.charAt(0)+section.label.slice(1).toLocaleLowerCase("es"),modules}]:[];
 });
 // Future server actions must stay editable even before a navigation binding exists.
 const remaining=actions.filter(item=>!placed.has(item.value));
 if(remaining.length)sections.push({name:"Otros permisos",modules:[{id:"other",name:"Acciones adicionales",icon:"lock",items:remaining}]});
 return sections;
}

const normalized=(value:string)=>value.normalize("NFD").replace(/[\u0300-\u036f]/g,"").toLocaleLowerCase("es");
export function filterPermissionSections(sections:PermissionSection[],search:string,show:"all"|"assigned"|"unassigned",selected:readonly string[]){
 const query=normalized(search.trim());
 return sections.flatMap(section=>{
  const modules=section.modules.flatMap(module=>{
   const items=module.items.filter(item=>(!query||normalized(`${section.name} ${module.name} ${item.label}`).includes(query))&&(show==="all"||(show==="assigned")===selected.includes(item.value)));
   return items.length?[{...module,visibleItems:items}]:[];
  });
  return modules.length?[{...section,modules}]:[];
 });
}
