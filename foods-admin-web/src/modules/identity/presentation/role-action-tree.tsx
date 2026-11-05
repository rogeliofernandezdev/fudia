"use client";
import {Icon} from "@/design-system/icons";
import {IconTile,type IconTileTone} from "@/design-system/icon-tile";
import {filterPermissionSections,type PermissionSection} from "./permission-navigation";

export function RoleActionTree({sections,selected,required,search="",show="all",openGroups={},onOpen,onToggle,onToggleModule,readOnly=false}:{
 sections:PermissionSection[];selected:readonly string[];required?:Map<string,string[]>;search?:string;show?:"all"|"assigned"|"unassigned";
 openGroups?:Record<string,boolean>;onOpen?:(key:string)=>void;onToggle?:(permission:string)=>void;onToggleModule?:(values:string[])=>void;readOnly?:boolean;
}){
 const visible=filterPermissionSections(sections,search,show,selected);
 const sectionTone=(name:string):IconTileTone=>name==="Abastecimiento"?"brand":name==="Carta y producción"||name==="Configuración"?"digital":name==="Negocio"?"teal":"primary";
 return <>{visible.map(section=><section className="re-action-section" data-tone={sectionTone(section.name)} key={section.name} aria-label={section.name}>
  <h3 className="re-section-title">{section.name}</h3>
  {section.modules.map(module=>{
   const tone=module.icon==="chefHat"||module.icon==="cookingPot"?"brand":sectionTone(section.name);
   const key=`actions-${module.id}`,isOpen=readOnly||openGroups[key]!==false;
   const count=module.items.filter(item=>selected.includes(item.value)).length;
   const optional=module.visibleItems.filter(item=>!required?.has(item.value));
   const complete=optional.length>0&&optional.every(item=>selected.includes(item.value));
   return <div className={`re-acc-group re-action-module${isOpen?" open":""}`} key={module.id}>
    <div className="re-acc-header">
     {readOnly?<h4 className="re-module-title"><IconTile name={module.icon} tone={tone} size="sm"/>{module.name}</h4>:<button type="button" className="re-acc-heading" aria-expanded={isOpen} aria-controls={`actions-${module.id.replaceAll("/","-")}`} onClick={()=>onOpen?.(key)}>
      <span className="re-acc-chevron"><Icon name="chevron" size={14}/></span><IconTile name={module.icon} tone={tone} size="sm"/><span className="re-acc-name">{module.name}</span>
      <span className={`re-acc-count${count?" has-selection":""}`}><b>{count}</b> de {module.items.length}</span>
     </button>}
     {!readOnly&&<button type="button" className={`re-acc-toggle${complete?" remove":""}`} disabled={!optional.length} aria-label={`${complete?"Quitar":"Marcar"} acciones opcionales ${search.trim()||show!=="all"?"visibles ":""}de ${module.name}`} onClick={()=>onToggleModule?.(optional.map(item=>item.value))}>{complete?"Quitar":"Todos"}</button>}
    </div>
    {isOpen&&<div className="re-acc-items" id={`actions-${module.id.replaceAll("/","-")}`}>
     {module.visibleItems.map(item=>{
      const checked=selected.includes(item.value),reasons=required?.get(item.value),reasonId=`req-${module.id.replaceAll("/","-")}-${item.value}`;
      return readOnly?<span className="re-action-read" key={item.value}><Icon name="check" size={16}/>{item.label}</span>:<label key={item.value} className={`re-acc-item${checked?" checked":""}${reasons?" required":""}`}>
       <input type="checkbox" checked={checked} disabled={Boolean(reasons)} aria-describedby={reasons?reasonId:undefined} onChange={()=>{if(!reasons)onToggle?.(item.value)}}/><i><Icon name="check" size={14}/></i>
       <span className="re-acc-item-copy">{item.label}{reasons&&<small id={reasonId}>Necesario para {reasons.join(", ")}</small>}</span>
      </label>;
     })}
    </div>}
   </div>;
  })}
 </section>)}</>;
}
