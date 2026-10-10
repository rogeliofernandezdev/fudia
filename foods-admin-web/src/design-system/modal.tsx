"use client";

import {createContext,useCallback,useContext,useId,useState,type FormHTMLAttributes,type ReactNode} from "react";
import {Dialog} from "./dialog";
import {ConfirmDialog} from "./confirm-dialog";
import {Icon,type IconName} from "./icons";
import {Button} from "./page-header";
import {ErrorState} from "./states";
import {Skeleton} from "./skeleton";

export type ModalSize="sm"|"md"|"lg"|"xl";
type ModalTone="primary"|"danger"|"success"|"digital";

type ModalContextValue={requestClose:()=>void;busy:boolean};
const ModalContext=createContext<ModalContextValue>({requestClose:()=>{},busy:false});
/** Acceso al cierre del modal (respeta `dirty` y `busy`) desde el pie o el cuerpo. */
export const useModal=()=>useContext(ModalContext);

type ModalProps={
  /** Acción concreta del modal: «Nuevo gasto», «Editar zona», «Pedido 0042». */
  title:ReactNode;
  /** Dato de contexto bajo el título (cliente, fecha, estado). Nunca repite el título. */
  subtitle?:ReactNode;
  icon:IconName;
  tone?:ModalTone;
  /** sm 480 · md 640 · lg 880 · xl 1040. En móvil sm/md son hoja inferior; lg/xl, pantalla completa. */
  size?:ModalSize;
  onClose:()=>void;
  /** Con cambios sin guardar, X y Cancelar piden confirmar el descarte. */
  dirty?:boolean;
  /** Durante una petición se bloquean los cierres. */
  busy?:boolean;
  closeLabel?:string;
  /** Cierre por respuesta global (FeedbackProvider). Por defecto, `onClose`. */
  onResponseClose?:()=>void;
  className?:string;
  children:ReactNode;
};

/**
 * Modal homologado sobre `Dialog`: fondo, panel por tamaño, cabecera (icono, título, subtítulo, X),
 * cuerpo desplazable y pie fijo. No cierra con Escape ni con el fondo.
 */
export function Modal({title,subtitle,icon,tone="primary",size="md",onClose,dirty=false,busy=false,closeLabel="Cerrar",onResponseClose,className="",children}:ModalProps){
  const id=useId();
  const [confirmDiscard,setConfirmDiscard]=useState(false);
  const requestClose=useCallback(()=>{if(busy)return;if(dirty){setConfirmDiscard(true);return}onClose()},[busy,dirty,onClose]);
  return <ModalContext.Provider value={{requestClose,busy}}>
    <div className="modal-backdrop modal-overlay-in">
      <Dialog onResponseClose={onResponseClose??onClose} className={`crud-modal ds-modal size-${size} ${className}`.trim()} aria-labelledby={`${id}-title`} aria-describedby={subtitle?`${id}-subtitle`:undefined} aria-busy={busy||undefined}>
        <div className="modal-accent" aria-hidden="true"/>
        <header className="modal-header">
          <span className={`modal-title-icon ${tone==="primary"?"":tone}`.trim()}><Icon name={icon} size={18}/></span>
          <div><h2 id={`${id}-title`}>{title}</h2>{subtitle&&<p id={`${id}-subtitle`}>{subtitle}</p>}</div>
          <button type="button" className="modal-close" aria-label={closeLabel} onClick={requestClose} disabled={busy}><Icon name="close" size={18}/></button>
        </header>
        {children}
      </Dialog>
    </div>
    <ConfirmDialog open={confirmDiscard} title="¿Descartar los cambios?" description="Los datos que ingresaste no se guardarán." confirmLabel="Descartar" onCancel={()=>setConfirmDiscard(false)} onConfirm={()=>{setConfirmDiscard(false);onClose()}}/>
  </ModalContext.Provider>;
}

/** Formulario del modal: ocupa el panel para que cuerpo y pie se repartan el alto. */
export function ModalForm({className="",...props}:FormHTMLAttributes<HTMLFormElement>){return <form className={`modal-form ${className}`.trim()} {...props}/>}
/** Cuerpo desplazable del modal. */
export function ModalBody({children,className=""}:{children:ReactNode;className?:string}){return <div className={`modal-body ${className}`.trim()}>{children}</div>}
/** Pie fijo: acción segura a la izquierda de la principal. `note` admite una restricción real, en texto neutro. */
export function ModalFooter({children,note,className=""}:{children:ReactNode;note?:ReactNode;className?:string}){return <footer className={`modal-footer ${className}`.trim()}>{note&&<p className="modal-footer-note">{note}</p>}{children}</footer>}
/** «Cancelar» homologado: usa el cierre del modal, que confirma el descarte si hay cambios. */
export function ModalCancel({label="Cancelar"}:{label?:string}){const {requestClose,busy}=useModal();return <Button kind="ghost" onClick={requestClose} disabled={busy}>{label}</Button>}

/** Carga remota dentro del modal: conserva cabecera y X operables; el cuerpo reproduce campos o datos. */
export function ModalLoading({kind="form",rows=6,label="Cargando"}:{kind?:"form"|"detail";rows?:number;label?:string}){
  return <div className={`modal-body modal-loading ${kind}`} role="status" aria-busy="true" aria-label={label}>
    {Array.from({length:rows},(_,index)=><div className="modal-loading-row" key={index}><Skeleton className="modal-loading-label"/><Skeleton className={kind==="form"?"modal-loading-control":"modal-loading-value"}/></div>)}
  </div>;
}

/** Error de carga dentro del modal: conserva el título del modal y ofrece Reintentar. */
export function ModalError({title="No pudimos cargar la información",text,onRetry,retrying}:{title?:string;text?:ReactNode;onRetry?:()=>void;retrying?:boolean}){
  return <div className="modal-body"><ErrorState title={title} text={text} onRetry={onRetry} retrying={retrying}/></div>;
}
