"use client";
import {Dialog} from "./dialog";
import "./styles/remote-modal-skeleton.css";
import {ModalHeader,ModalFooter} from "./modal";

/** Modal en carga: la X permanece operable con teclado; solo los bloques de carga se ocultan a tecnologías de asistencia. */
export function RemoteModalSkeleton({close,className="",label="Cargando formulario",rows=6}:{close:()=>void;className?:string;label?:string;rows?:number}){
  return <div className="modal-backdrop modal-overlay-in">
    <Dialog onResponseClose={close} className={`crud-modal remote-modal-skeleton modal-panel-in ${className}`.trim()} role="dialog" aria-modal="true" aria-busy="true" aria-label={label}>

      <ModalHeader title={<span className="remote-modal-skeleton-copy" aria-hidden="true"><b/></span>} icon="clock" onClose={close} initialFocusClose/>
      <div className="remote-modal-skeleton-body" aria-hidden="true">
        <div className="remote-modal-skeleton-grid">
          {Array.from({length:rows},(_,index)=><div className={index===0||index===rows-1?"remote-modal-skeleton-field wide":"remote-modal-skeleton-field"} key={index}><i/><b/></div>)}
        </div>
      </div>
      <ModalFooter><div className="remote-modal-skeleton-footer" aria-hidden="true"><span/><b/></div></ModalFooter>
    </Dialog>
  </div>;
}
