"use client";
import {Dialog} from "./dialog";
import "./styles/remote-modal-skeleton.css";
import {ModalHeader,ModalFooter} from "./modal";
import type {ReactNode} from "react";

type Props={close:()=>void;className?:string;label?:string;rows?:number;title?:ReactNode};

/**
 * Modal en carga: la X permanece operable con teclado; solo los bloques de carga se ocultan a tecnologías de asistencia.
 * La cabecera no espera al API: con `title` se muestra la misma del modal final desde el primer cuadro;
 * solo sin esos datos se dibuja el título en carga.
 */
export function RemoteModalSkeleton({close,className="",label="Cargando formulario",rows=6,title}:Props){
  return <div className="modal-backdrop modal-overlay-in">
    <Dialog onResponseClose={close} className={`crud-modal remote-modal-skeleton modal-panel-in ${className}`.trim()} role="dialog" aria-modal="true" aria-busy="true" aria-label={label}>

      {title?<ModalHeader title={title} onClose={close} initialFocusClose/>
      :<ModalHeader title={<span className="remote-modal-skeleton-copy" aria-hidden="true"><b/></span>} onClose={close} initialFocusClose/>}
      <div className="remote-modal-skeleton-body" aria-hidden="true">
        <div className="remote-modal-skeleton-grid">
          {Array.from({length:rows},(_,index)=><div className={index===0||index===rows-1?"remote-modal-skeleton-field wide":"remote-modal-skeleton-field"} key={index}><i/><b/></div>)}
        </div>
      </div>
      <ModalFooter><div className="remote-modal-skeleton-footer" aria-hidden="true"><span/><b/></div></ModalFooter>
    </Dialog>
  </div>;
}
