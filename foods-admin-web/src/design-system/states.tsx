import type {ReactNode} from "react";
import {Icon,type IconName} from "./icons";
import {Button} from "./page-header";

type StateProps={
  /** Título breve que explica el estado. */
  title:string;
  /** Detalle opcional: qué pasó o qué puede hacer el usuario. */
  text?:ReactNode;
  className?:string;
};

/**
 * Estado vacío homologado: icono en una superficie suave, título, detalle y una
 * acción opcional (por ejemplo «Nuevo producto»). Distingue «aún no hay datos» de
 * «sin coincidencias» mediante el texto que recibe.
 */
export function EmptyState({icon="info",title,text,action,className=""}:StateProps&{icon?:IconName;action?:ReactNode}){
  return <div className={`catalog-state ${className}`.trim()} role="status">
    <span><Icon name={icon}/></span>
    <b>{title}</b>
    {text&&<p>{text}</p>}
    {action}
  </div>;
}

/**
 * Estado de error recuperable: nunca reemplaza datos por ceros y, cuando hay
 * `onRetry`, ofrece «Reintentar» como acción secundaria con icono de recarga.
 */
export function ErrorState({title="No pudimos cargar la información",text,onRetry,retrying=false,className=""}:StateProps&{onRetry?:()=>void;retrying?:boolean}){
  return <div className={`catalog-state error ${className}`.trim()} role="alert">
    <span><Icon name="alert"/></span>
    <b>{title}</b>
    {text&&<p>{text}</p>}
    {onRetry&&<Button kind="secondary" icon="refresh" onClick={onRetry} disabled={retrying} loading={retrying}>{retrying?"Reintentando…":"Reintentar"}</Button>}
  </div>;
}
