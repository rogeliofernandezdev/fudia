"use client";
import "./styles/tabs.css";
import {useRef,type KeyboardEvent,type ReactNode} from "react";
import {Icon,type IconName} from "./icons";

export type TabItem<T extends string>={value:T;label:string;count?:number;icon:IconName};

/** Alinea navegación y acciones de la vista sin llevarlas a la cabecera. */
export function TabsBar({children,action}:{children:ReactNode;action?:ReactNode}){
  return <div className="ds-tabs-bar">{children}{action&&<div className="ds-tabs-actions">{action}</div>}</div>;
}

/**
 * Pestañas homologadas (control segmentado): `role="tablist"`, foco itinerante y
 * flechas/Inicio/Fin para moverse. Icono obligatorio y contador opcional.
 */
export function Tabs<T extends string>({items,value,onChange,label,className=""}:{items:TabItem<T>[];value:T;onChange:(value:T)=>void;label:string;className?:string}){
  const list=useRef<HTMLDivElement>(null);
  const move=(event:KeyboardEvent<HTMLButtonElement>,index:number)=>{
    const last=items.length-1;
    const next=event.key==="ArrowRight"?(index===last?0:index+1):event.key==="ArrowLeft"?(index===0?last:index-1):event.key==="Home"?0:event.key==="End"?last:null;
    if(next===null)return;
    event.preventDefault();
    onChange(items[next].value);
    list.current?.querySelectorAll<HTMLButtonElement>("[role=tab]")[next]?.focus();
  };
  return <div ref={list} className={`ds-tabs ${className}`.trim()} role="tablist" aria-label={label}>
    {items.map((item,index)=>{
      const selected=item.value===value;
      return <button key={item.value} type="button" role="tab" aria-selected={selected} tabIndex={selected?0:-1} className={selected?"active":""} onClick={()=>onChange(item.value)} onKeyDown={event=>move(event,index)}>
        <Icon name={item.icon} size={16}/>
        <span>{item.label}</span>
        {item.count!==undefined&&<b>{item.count}</b>}
      </button>;
    })}
  </div>;
}
