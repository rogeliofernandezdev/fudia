"use client";
import "./concierge-settings.css";
import {useQuery} from "@tanstack/react-query";
import {ErrorState,Icon,PageHeader,Skeleton,Status} from "@/design-system";
import {getConciergeSettings} from "../infrastructure/concierge-settings-api";

export function ConciergeSettingsPage(){
 const query=useQuery({queryKey:["concierge-settings"],queryFn:getConciergeSettings});
 const data=query.data;
 return <><PageHeader title="Fudia Concierge" description="Consulta el estado del módulo de pedidos por WhatsApp con IA para esta empresa."/>
  <div className="concierge-settings">
   {query.isLoading?<ConciergeSettingsSkeleton/>:query.isError?<section className="panel"><ErrorState title="No pudimos cargar Fudia Concierge" text={query.error.message} onRetry={()=>void query.refetch()} retrying={query.isFetching}/></section>:data&&<>
    <section className="panel concierge-overview">
     <div><span><Icon name="chat" size={24}/></span><section><h2>{data.organizationName}</h2><p>{data.locationName} · El módulo se administra desde la plataforma global de Fudia.</p></section></div>
     <Status tone={data.active?"green":"gray"}>{data.active?"Habilitado":"No habilitado"}</Status>
    </section>

    <div className="concierge-config">
     <section className="panel concierge-form">
      <header><h2>Canal oficial de Fudia</h2><p>La empresa no configura números, credenciales de Meta ni activaciones locales. El bot y su número se administran exclusivamente en Meta / WhatsApp Business.</p></header>

      {!data.active?<div className="concierge-status-card locked">
       <span><Icon name="lock" size={20}/></span>
       <div><b>Fudia Concierge no está habilitado para esta empresa</b><small>El administrador global de Fudia puede activar el módulo <code>whatsapp_bot</code> cuando forme parte del contrato o plan de la empresa.</small></div>
      </div>:<div className="concierge-status-card ready">
       <span><Icon name="check" size={20}/></span>
       <div><b>Módulo habilitado por Fudia</b><small>La empresa está autorizada a usar Concierge. La identidad y operación del canal oficial se administran en Meta / WhatsApp Business.</small></div>
      </div>}

      <div className="concierge-readonly-field">
       <small>ADMINISTRACIÓN DEL CANAL</small>
       <b>Meta / WhatsApp Business</b>
       <p>El número oficial no se almacena ni se configura en esta aplicación. Fudia utiliza la integración técnica autorizada con Meta para operar el bot.</p>
      </div>

      <div className="concierge-note">La empresa obtiene acceso únicamente mediante el módulo contratado. La activación y desactivación pertenecen al administrador global de Fudia.</div>
     </section>

     <section className="panel concierge-preview">
      <header><h2>Flujo del comensal</h2><p>El QR identifica la empresa, el local y la mesa; Meta mantiene la identidad del canal oficial.</p></header>
      <div className="concierge-flow">
       <span><i>1</i><b>Escanea el QR<small>Fudia identifica empresa, local y mesa mediante un token opaco.</small></b></span>
       <span><i>2</i><b>Abre WhatsApp<small>Fudia Concierge obtiene desde Meta el canal oficial y abre la conversación.</small></b></span>
       <span><i>3</i><b>Conversa con Concierge<small>La IA consulta únicamente la carta real de la empresa identificada por el QR.</small></b></span>
       <span><i>4</i><b>Confirma el pedido<small>foods-backend revalida precios, disponibilidad y mesa.</small></b></span>
       <span><i>5</i><b>Pedido a Cocina<small>La ronda confirmada aparece en KDS con canal WhatsApp.</small></b></span>
      </div>
      <div className="concierge-note">Si el módulo <code>whatsapp_bot</code> no está activo para la empresa, el QR no ofrece pedidos por WhatsApp y el backend de Concierge rechaza el acceso.</div>
     </section>
    </div>
   </>}
  </div>
 </>;
}

function ConciergeSettingsSkeleton(){
 return <div className="concierge-skeleton" role="status" aria-busy="true" aria-label="Cargando estado de Fudia Concierge">
  <section className="panel concierge-overview" aria-hidden="true">
   <div><Skeleton width="var(--size-44)" height="var(--size-44)" radius="var(--radius-card)"/><section className="concierge-skeleton-lines"><Skeleton width="var(--size-240)" height="var(--size-14)"/><Skeleton width="var(--size-330)" height="var(--size-12)"/></section></div>
   <Skeleton width="var(--size-120)" height="var(--size-28)" radius="var(--radius-sheet)"/>
  </section>
  <div className="concierge-config" aria-hidden="true">
   <section className="panel concierge-form">
    <div className="concierge-skeleton-lines"><Skeleton width="var(--size-180)" height="var(--size-14)"/><Skeleton height="var(--size-12)"/><Skeleton width="72%" height="var(--size-12)"/></div>
    {Array.from({length:2},(_,i)=><Skeleton key={i} height="var(--size-84)" radius="var(--radius-card)"/>)}
    <Skeleton height="var(--size-48)" radius="var(--radius-tile)"/>
   </section>
   <section className="panel concierge-preview">
    <div className="concierge-skeleton-lines"><Skeleton width="var(--size-160)" height="var(--size-14)"/><Skeleton height="var(--size-12)"/></div>
    <div className="concierge-flow">{Array.from({length:5},(_,i)=><Skeleton key={i} height="var(--size-48)" radius="var(--radius-tile)"/>)}</div>
   </section>
  </div>
 </div>;
}
