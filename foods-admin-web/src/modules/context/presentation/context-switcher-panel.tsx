import type {Ref} from "react";
import {Button,FormField,Icon,IconButton,Select} from "@/design-system";
import type {LocationSummary,OrgSummary} from "../domain/types";

type Props={
  panelRef?:Ref<HTMLElement>;id:string;platformAdmin:boolean;organizationId:string;locationId:string;
  organizations:OrgSummary[];locations:LocationSummary[];loading:boolean;error?:string;busy:boolean;canApply:boolean;
  onOrganizationChange:(id:string)=>void;onLocationChange:(id:string)=>void;onApply:()=>void;onClose:()=>void;onRetry:()=>void;
};

export function ContextSwitcherPanel({panelRef,id,platformAdmin,organizationId,locationId,organizations,locations,loading,error,busy,canApply,onOrganizationChange,onLocationChange,onApply,onClose,onRetry}:Props){
  const noOrganizations=platformAdmin&&!organizations.length;
  return <section ref={panelRef} id={id} className="context-popover" role="dialog" aria-modal="false" aria-labelledby={`${id}-title`} aria-busy={loading||busy}>
    <header className="context-popover-head">
      <span className="context-popover-icon"><Icon name={platformAdmin?"building":"store"} size={18}/></span>
      <h2 id={`${id}-title`}>{platformAdmin?"Empresa y local":"Cambiar local"}</h2>
      <IconButton icon="close" label="Cerrar selector" className="context-close" disabled={busy} onClick={onClose}/>
    </header>
    {loading?<div className="context-fields context-loading" aria-label="Cargando opciones" aria-busy="true">{Array.from({length:platformAdmin?2:1},(_,index)=><div aria-hidden="true" key={index}><i className="context-skeleton-bar"/><span className="context-skeleton-bar"/></div>)}<b className="context-skeleton-bar" aria-hidden="true"/></div>
    :error?<div className="context-state" role="alert"><Icon name="alert" size={22}/><b>No pudimos cargar las opciones</b><p>{error}</p><Button kind="secondary" icon="refresh" onClick={onRetry}>Reintentar</Button></div>
    :noOrganizations?<div className="context-state"><Icon name="building" size={22}/><p>No hay empresas disponibles.</p></div>
    :<form onSubmit={event=>{event.preventDefault();if(canApply&&!busy)onApply()}}>
      <div className="context-fields">
        {platformAdmin&&<FormField label="Empresa"><Select value={organizationId} disabled={busy} onChange={event=>onOrganizationChange(event.target.value)}><option value="">Selecciona una empresa</option>{organizations.map(item=><option key={item.id} value={item.id}>{item.name}</option>)}</Select></FormField>}
        <FormField label="Local"><Select value={locationId} disabled={busy||(platformAdmin&&!organizationId)||!locations.length} onChange={event=>onLocationChange(event.target.value)}><option value="">Selecciona un local</option>{locations.map(item=><option key={item.id} value={item.id}>{item.name}</option>)}</Select></FormField>
        {!locations.length&&(!platformAdmin||organizationId)&&<p className="context-empty" role="status">No hay locales disponibles.</p>}
      </div>
      <footer className="context-popover-actions"><Button type="submit" className="context-apply" icon={busy?"refresh":"check"} disabled={busy||!canApply} aria-busy={busy}>{busy?"Cambiando…":"Aplicar"}</Button></footer>
    </form>}
  </section>;
}
