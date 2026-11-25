"use client";

import {useState} from "react";

/**
 * Paginación en el navegador para listas que el API entrega completas (Recetas, Entregas
 * fallidas, historiales de Plataforma). Devuelve las filas de la página y las props de la
 * primitiva compartida `Pagination`, así toda tabla de gestión pagina igual. `resetKey`
 * (búsqueda, filtro) vuelve a la primera página cuando cambia.
 */
export function useClientPagination<T>(items:readonly T[],resetKey?:unknown,initialSize=10){
  const[page,setPage]=useState(1);
  const[size,setSize]=useState(initialSize);
  const[key,setKey]=useState(resetKey);
  if(key!==resetKey){setKey(resetKey);setPage(1);}
  const last=Math.max(1,Math.ceil(items.length/size));
  const current=Math.min(page,last);
  return{
    rows:items.slice((current-1)*size,current*size),
    pagination:{page:current,size,total:items.length,onPage:setPage,onSize:(value:number)=>{setSize(value);setPage(1);}},
  };
}
