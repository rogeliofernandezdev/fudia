import type {ReactNode} from "react";
import {IconTile,type IconName,type IconTileTone} from "@/design-system";

/** Panel de Mi restaurante: icono tintado y título; sin descripciones, los campos se explican solos. */
export function OrganizationPanel({icon,tone="primary",title,aside,className="",children}:{icon:IconName;tone?:IconTileTone;title:string;aside?:ReactNode;className?:string;children:ReactNode}){
 return <section className={`panel organization-panel ${className}`.trim()}>
  <header className="organization-panel-header"><div><IconTile name={icon} tone={tone} size="sm"/><h2>{title}</h2></div>{aside}</header>
  <div className="organization-panel-body">{children}</div>
 </section>;
}
