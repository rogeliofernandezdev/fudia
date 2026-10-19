import type {Product} from "./types";
import {reservationLimit} from "./option-quantity";

export type ComboPickOption={value:string;label:string;category:string;limit:number|null};
export type ComboPickGroup={label:string;options:ComboPickOption[]};

const NO_CATEGORY="Sin categoría";
export const normalize=(value:string)=>value.normalize("NFD").replace(/[̀-ͯ]/g,"").toLowerCase().trim();

/** Sinónimos de las partes del menú del día: en el Perú la entrada suele ser una sopa, un caldo o una ensalada. */
const PART_ALIASES:Record<string,string[]>={
  entrada:["entrada","sopa","caldo","ensalada","piqueo","aperitivo"],
  segundo:["segundo","principal","fondo"],
  postre:["postre","dulce"],
  bebida:["bebida","refresco","jugo","gaseosa","infusion"],
};

/**
 * Una categoría pertenece a una parte si contiene su nombre en singular («Entrada» ↔ «Entradas») o, en las partes
 * de la plantilla, uno de sus sinónimos («Entrada» ↔ «Sopas»). Partes con otro nombre solo usan su propio nombre.
 */
export function belongsToGroup(product:Product,groupName:string){
  const category=normalize(product.categoryName??"");
  const group=normalize(groupName).replace(/s$/,"");
  if(!group||!category)return false;
  return (PART_ALIASES[group]??[group]).some(term=>category.includes(term));
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
