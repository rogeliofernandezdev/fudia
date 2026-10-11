import type {Category} from "./types";

function normalize(value:string){return value.normalize("NFD").replace(/[\u0300-\u036f]/g,"").toLocaleLowerCase("es").trim()}

/** Filtra el catálogo completo antes de paginar; no solo la página visible. */
export function filterCategories(items:Category[],query:{search:string;scope:string;status:string}){
 const search=normalize(query.search);
 return items.filter(category=>(!search||normalize(category.name).includes(search))&&(!query.scope||category.productScope===query.scope)&&(!query.status||category.active===(query.status==="active")));
}
