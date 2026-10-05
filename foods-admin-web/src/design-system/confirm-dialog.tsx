"use client";

import {useId} from "react";
import "./styles/confirm-dialog.css";
import {Icon} from "@/design-system/icons";

type ConfirmDialogProps={
  open:boolean;
  title:string;
  description:string;
  subject?:{label:string;name:string};
  note?:string;
  tone?:"danger"|"success";
  confirmLabel:string;
  pending?:boolean;
  onCancel:()=>void;
  onConfirm:()=>void;
};

export function ConfirmDialog({open,title,description,subject,note,tone="danger",confirmLabel,pending=false,onCancel,onConfirm}:ConfirmDialogProps){
  const id=useId();
  if(!open)return null;

  return <div className="confirm-overlay">
    <section className="confirm-card" role="alertdialog" aria-modal="true" aria-labelledby={`${id}-title`} aria-describedby={`${id}-description`} aria-busy={pending}>
      <header className="confirm-header">
        <span className={`confirm-icon ${tone}`}><Icon name={tone==="danger"?"alert":"check"} size={20}/></span>
        <h2 id={`${id}-title`}>{title}</h2>
      </header>
      <div className="confirm-body" id={`${id}-description`}>
        {subject&&<div className="confirm-subject"><span>{subject.label}</span><strong>{subject.name}</strong></div>}
        <p className="confirm-description">{description}</p>
        {note&&<p className="confirm-note"><Icon name="ledger" size={16}/><span>{note}</span></p>}
      </div>
      <footer className="confirm-actions">
        <button type="button" className="button ghost" autoFocus disabled={pending} onClick={onCancel}>Cancelar</button>
        <button type="button" className={`button ${tone}`} disabled={pending} onClick={onConfirm}>{pending?"Procesando…":confirmLabel}</button>
      </footer>
    </section>
  </div>;
}
