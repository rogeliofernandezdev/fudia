import {accessKey,navigationGroups} from "@/shell/navigation";
import type {IconName} from "@/design-system/icons";
import type {PermissionGroup,ScreenActions} from "../domain/types";

export type PermissionItem={value:string;label:string;sharedWith?:string[]};
export type PermissionModule={id:string;name:string;icon:IconName;items:PermissionItem[]};
export type PermissionSection={name:string;modules:PermissionModule[]};

// Cada pantalla del menú lateral muestra las acciones que el API declara para ella
// (screenActions), con la etiqueta de esa pantalla. Una acción compartida conserva
// un solo valor y avisa en qué otras pantallas aplica.
export function permissionSections(groups:PermissionGroup[],menuGroups?:PermissionGroup[],screenActions:ScreenActions={}):PermissionSection[]{
 const allowedMenus=menuGroups?new Set(menuGroups.flatMap(group=>group.items.map(item=>item.value))):undefined;
 const actions=[...new Map(groups.flatMap(group=>group.items).map(item=>[item.value,item])).values()];
 const available=new Set(actions.map(item=>item.value));
 const sections=navigationGroups.flatMap(section=>{
  const modules=section.items.flatMap(module=>{
   const key=accessKey(module);
   if(allowedMenus&&!allowedMenus.has(key))return [];
   const items=(screenActions[key]??[]).filter(item=>available.has(item.value)).map((item):PermissionItem=>({value:item.value,label:item.label}));
   return items.length?[{id:module.href,name:module.name,icon:module.icon,items} as PermissionModule]:[];
  });
  return modules.length?[{name:section.label.charAt(0)+section.label.slice(1).toLocaleLowerCase("es"),modules}]:[];
 });
 const modules=sections.flatMap(section=>section.modules);
 const placed=new Set(modules.flatMap(module=>module.items.map(item=>item.value)));
 for(const screen of modules)for(const item of screen.items){
  const others=modules.filter(other=>other!==screen&&other.items.some(entry=>entry.value===item.value)).map(other=>other.name);
  if(others.length)item.sharedWith=others;
 }
 // Una acción nueva del API sin pantalla asignada sigue siendo editable.
 const remaining=actions.filter(item=>!placed.has(item.value));
 if(remaining.length)sections.push({name:"Otros permisos",modules:[{id:"other",name:"Acciones adicionales",icon:"lock",items:remaining}]});
 return sections;
}

/** «También en Pedidos, Salón y 3 más»: dónde más aplica una acción compartida. */
export function sharedNote(names:readonly string[]){
 const shown=names.slice(0,3),rest=names.length-shown.length;
 return `También en ${rest>0?`${shown.join(", ")} y ${rest} más`:shown.length>1?`${shown.slice(0,-1).join(", ")} y ${shown.at(-1)}`:shown[0]}`;
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
