"use client";
import "./styles/remote-modal-skeleton.css";
import {Icon} from "./icons";

export function RemoteModalSkeleton({close,className="",label="Cargando formulario",rows=6}:{close:()=>void;className?:string;label?:string;rows?:number}){
  return <div className="modal-backdrop modal-overlay-in">
    <section className={`crud-modal remote-modal-skeleton modal-panel-in ${className}`.trim()} role="dialog" aria-modal="true" aria-busy="true" aria-label={label}>
      <div className="modal-accent"/>
      <header className="remote-modal-skeleton-head" aria-hidden="true">
        <span className="remote-modal-skeleton-block remote-modal-skeleton-icon"/>
        <div className="remote-modal-skeleton-copy"><span/><b/></div>
        <button aria-label="Cerrar" onClick={close}><Icon name="close"/></button>
      </header>
      <div className="remote-modal-skeleton-body" aria-hidden="true">
        <div className="remote-modal-skeleton-grid">
          {Array.from({length:rows},(_,index)=><div className={index===0||index===rows-1?"remote-modal-skeleton-field wide":"remote-modal-skeleton-field"} key={index}><i/><b/></div>)}
        </div>
        <div className="remote-modal-skeleton-section"><span/><i/><i/></div>
      </div>
      <footer className="remote-modal-skeleton-footer" aria-hidden="true"><span/><b/></footer>
    </section>
  </div>;
}
