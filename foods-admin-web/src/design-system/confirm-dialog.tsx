"use client";

import {Dialog} from "./dialog";
import {Children,useId,type ReactNode} from "react";
import "./styles/confirm-dialog.css";
import {Icon} from "@/design-system/icons";

type ConfirmDialogProps={
  open:boolean;
  title:string;
  description?:ReactNode;
  subject?:{label?:string;name:string};
  children?:ReactNode;
  note?:string;
  tone?:"danger"|"success";
  confirmLabel:string;
  pending?:boolean;
  onCancel:()=>void;
  onConfirm:()=>void;
};

export function ConfirmDialog({open,title,description,subject,children,note,tone="danger",confirmLabel,pending=false,onCancel,onConfirm}:ConfirmDialogProps){
  const id=useId();
  if(!open)return null;
  const hasMessage=Children.toArray(description).length>0;
  const hasDescription=Boolean(subject||note)||hasMessage||Children.toArray(children).length>0;

  return <div className="confirm-overlay modal-overlay-in">
    <Dialog onResponseClose={onCancel} className="confirm-card" role="alertdialog" aria-modal="true" aria-labelledby={`${id}-title`} aria-describedby={hasDescription?`${id}-description`:undefined} aria-busy={pending}>
      <header className="confirm-header">
        <span className={`confirm-icon ${tone}`}><Icon name={tone==="danger"?"alert":"check"} size={20}/></span>
        <h2 id={`${id}-title`}>{title}</h2>
      </header>
      {hasDescription&&<div className="confirm-body" id={`${id}-description`}>
        {subject&&<p className="confirm-description">{subject.label&&`${subject.label}: `}<strong>{subject.name}</strong></p>}
        {hasMessage&&<p className="confirm-description">{description}</p>}
        {children}
        {note&&<p className="confirm-note">{note}</p>}
      </div>}
      <footer className="confirm-actions">
        <button type="button" className="button ghost" data-dialog-initial-focus autoFocus disabled={pending} onClick={onCancel}>Cancelar</button>
        <button type="button" className={`button ${tone}`} disabled={pending} onClick={onConfirm}>{pending?"Procesando…":confirmLabel}</button>
      </footer>
    </Dialog>
  </div>;
}
