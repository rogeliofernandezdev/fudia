"use client";
import "./styles/tabs.css";
import {useRef,type KeyboardEvent} from "react";
import {Icon,type IconName} from "./icons";

export type TabItem<T extends string>={value:T;label:string;count?:number;icon?:IconName};

/**
 * Pestañas homologadas (control segmentado): `role="tablist"`, foco itinerante y
 * flechas/Inicio/Fin para moverse. El contador opcional acompaña cada pestaña.
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
        {item.icon&&<Icon name={item.icon} size={16}/>}
        <span>{item.label}</span>
        {item.count!==undefined&&<b>{item.count}</b>}
      </button>;
    })}
  </div>;
}
