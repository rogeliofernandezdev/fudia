import "./styles/session-error.css";
import type {ReactNode} from "react";
import {Icon,type IconName} from "@/design-system/icons";
import {Logo} from "@/design-system/logo";

type AdminSessionErrorProps={
  icon:IconName;
  eyebrow?:string;
  title:string;
  description:string;
  fullPage?:boolean;
  children?:ReactNode;
};

export function AdminSessionError({icon,eyebrow,title,description,fullPage=false,children}:AdminSessionErrorProps){
  const content=<section className="admin-session-error" role="alert">
    <span className="admin-session-error-icon"><Icon name={icon} size={25}/></span>
    {eyebrow&&<span className="admin-session-error-eyebrow">{eyebrow}</span>}
    <h1>{title}</h1>
    <p>{description}</p>
    {children&&<div className="admin-session-error-actions">{children}</div>}
  </section>;

  if(!fullPage)return content;
  return <main className="admin-session-error-page">
    <div className="admin-session-error-brand"><Logo/></div>
    {content}
    <small>fudIA protege tu sesión y los accesos de cada usuario.</small>
  </main>;
}
