import type {Product} from "./types";
import {reservationLimit} from "./option-quantity";

export type ComboPickOption={value:string;label:string;category:string;limit:number|null};
export type ComboPickGroup={label:string;options:ComboPickOption[]};

const NO_CATEGORY="Sin categoría";
export const normalize=(value:string)=>value.normalize("NFD").replace(/[̀-ͯ]/g,"").toLowerCase().trim();

/** Una categoría pertenece a una parte si coincide con su nombre: «Entrada» ↔ «Entradas», «Segundo» ↔ «Segundos». */
export function belongsToGroup(product:Product,groupName:string){
  const category=normalize(product.categoryName??"");
  const group=normalize(groupName);
  return Boolean(group)&&(category===group||category===`${group}s`||category.startsWith(`${group} `)||category.includes(group));
}

const capitalize=(value:string)=>value.charAt(0).toLocaleUpperCase("es")+value.slice(1);

/**
 * Secciones del buscador «Agregar opción»: una por categoría de la carta (nombre con mayúscula inicial), sin los
 * productos ya agregados a la parte. Primero las categorías de la parte, luego el resto en orden alfabético y
 * «Sin categoría» al final; una sección sin productos disponibles no se muestra.
 */
export function buildOptionGroups(products:Product[],excluded:string[],partName:string):ComboPickGroup[]{
  const sections=new Map<string,ComboPickGroup&{matches:boolean}>();
  for(const product of products){
    if(excluded.includes(product.id))continue;
    const category=product.categoryName?.trim()||NO_CATEGORY;
    const key=normalize(category);
    const section=sections.get(key)??{label:capitalize(category),options:[],matches:belongsToGroup(product,partName)};
    section.options.push({value:product.id,label:product.name,category:capitalize(category),limit:reservationLimit(product)});
    sections.set(key,section);
  }
  return Array.from(sections.values())
    .sort((a,b)=>Number(b.matches)-Number(a.matches)||Number(a.label===NO_CATEGORY)-Number(b.label===NO_CATEGORY)||a.label.localeCompare(b.label,"es"))
    .map(({label,options})=>({label,options:options.sort((a,b)=>a.label.localeCompare(b.label,"es"))}));
}
