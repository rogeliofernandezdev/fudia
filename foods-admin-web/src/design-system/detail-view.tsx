import type {ReactNode} from "react";
import {Icon,type IconName} from "./icons";
import {IconTile,type IconTileTone} from "./icon-tile";

/**
 * Piezas de detalle con la organización de la referencia aprobada: identidad
 * con su estado, tarjeta de contacto, datos en dos columnas y progreso con
 * círculos. Se combinan con `Tabs` dentro del cuerpo de los modales de detalle.
 */
export function DetailHero({title,status,meta}:{title:ReactNode;status?:ReactNode;meta?:ReactNode}){
  return <section className="detail-hero">
    <div className="detail-hero-head"><h3>{title}</h3>{status}</div>
    {meta&&<p className="detail-hero-meta">{meta}</p>}
  </section>;
}

export function DetailContact({icon="user",tone="teal",name,detail,actions}:{icon?:IconName;tone?:IconTileTone;name:ReactNode;detail?:ReactNode;actions?:ReactNode}){
  return <section className="detail-contact">
    <IconTile name={icon} tone={tone}/>
    <div><b>{name}</b>{detail&&<small>{detail}</small>}</div>
    {actions&&<div className="detail-contact-actions">{actions}</div>}
  </section>;
}

export type DetailFact={label:string;value:ReactNode;wide?:boolean};

export function DetailFacts({items,label}:{items:DetailFact[];label?:string}){
  return <dl className="detail-facts" aria-label={label}>
    {items.map(item=><div key={item.label} className={item.wide?"wide":undefined}><dt>{item.label}</dt><dd>{item.value}</dd></div>)}
  </dl>;
}

export type DetailStep={key:string;icon:IconName;title:ReactNode;time?:ReactNode;description?:ReactNode;aside?:ReactNode;state?:"done"|"current"|"pending"};

/** Lista de hitos o movimientos con círculo: cumplido oscuro, actual índigo, pendiente gris. */
export function DetailTimeline({steps,label}:{steps:DetailStep[];label:string}){
  return <ol className="detail-timeline" aria-label={label}>
    {steps.map(step=><li key={step.key} className={`is-${step.state??"done"}`}>
      <span className="detail-timeline-dot" aria-hidden="true"><Icon name={step.icon} size={16}/></span>
      <div className="detail-timeline-body">
        <p><b>{step.title}</b>{step.time&&<span>{step.time}</span>}</p>
        {step.description&&<small>{step.description}</small>}
      </div>
      {step.aside&&<div className="detail-timeline-aside">{step.aside}</div>}
    </li>)}
  </ol>;
}
