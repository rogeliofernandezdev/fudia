"use client";

import Link from "next/link";
import {useId,useRef,useState} from "react";
import {Icon} from "@/design-system/icons";
import type {NavGroup} from "./navigation";
import "./styles/navigation-search.css";

export function NavigationSearch({groups}:{groups:NavGroup[]}){
  const [query,setQuery]=useState("");
  const [open,setOpen]=useState(false);
  const root=useRef<HTMLDivElement>(null);
  const id=useId();
  const normalize=(value:string)=>value.normalize("NFD").replace(/[\u0300-\u036f]/g,"").toLowerCase();
  const results=groups.flatMap(group=>group.items.map(item=>({...item,group:group.label}))).filter(item=>normalize(item.name).includes(normalize(query.trim())));
  return <div className="global-search" ref={root} role="search" onBlur={event=>{if(!event.currentTarget.contains(event.relatedTarget))setOpen(false);}}>
    <Icon name="search" size={18}/>
    <input aria-label="Buscar opción del sistema" aria-controls={open&&query.trim()?id:undefined} placeholder="Buscar opción del sistema…" value={query} onFocus={()=>setOpen(true)} onChange={event=>{setQuery(event.target.value);setOpen(true);}} onKeyDown={event=>{if(event.key==="ArrowDown"){event.preventDefault();root.current?.querySelector<HTMLAnchorElement>("nav a")?.focus();}}}/>
    {open&&query.trim()&&<nav id={id} className="navigation-search-results" aria-label="Resultados de navegación">
      {results.length?results.map(item=><Link key={item.href} href={item.href} onClick={()=>{setOpen(false);setQuery("");}}><Icon name={item.icon} size={18}/><span><b>{item.name}</b><small>{item.group}</small></span><Icon name="chevron" size={14}/></Link>):<p role="status">No encontramos una opción con ese nombre.</p>}
    </nav>}
  </div>;
}
