import Link from "next/link";
import type {ReactNode} from "react";
import {IconTile,type IconTileTone} from "./icon-tile";
import {Icon,type IconName} from "./icons";

/**
 * Indicador homologado: icono tintado y etiqueta arriba, flecha ↗ cuando abre
 * el detalle, cifra protagonista y una línea de contexto. `attention` solo
 * para lo que pide acción (demoras, vencidos): tiñe la cifra en ámbar.
 */
export function StatCard({icon,tone="primary",label,value,meta,href,attention=false,className=""}:{icon:IconName;tone?:IconTileTone;label:string;value:ReactNode;meta?:ReactNode;href?:string;attention?:boolean;className?:string}){
  return <article className={["panel stat-card",attention?"attention":"",className].filter(Boolean).join(" ")}>
    <div className="stat-card-head">
      <IconTile name={icon} tone={attention?"warning":tone}/>
      <small>{label}</small>
      {href&&<Link href={href} className="stat-card-link" aria-label={`Ver detalle: ${label}`}><Icon name="arrowUpRight" size={16}/></Link>}
    </div>
    <strong>{value}</strong>
    {meta&&<em>{meta}</em>}
  </article>;
}
