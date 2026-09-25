"use client";
import "./concierge-settings.css";
import {useQuery} from "@tanstack/react-query";
import {Button,PageHeader} from "@/design-system/page-header";
import {Icon} from "@/design-system/icons";
import {getConciergeSettings} from "../infrastructure/concierge-settings-api";

export function ConciergeSettingsPage(){
 const query=useQuery({queryKey:["concierge-settings"],queryFn:getConciergeSettings});
 const data=query.data;
 return <><PageHeader eyebrow="CONFIGURACIÓN" title="Fudia Concierge" description="Consulta el estado del módulo de pedidos por WhatsApp con IA para esta empresa."/>
  <div className="concierge-settings">
   {query.isLoading?<section className="panel concierge-loading">Cargando estado de Fudia Concierge…</section>:query.isError?<section className="panel concierge-error"><Icon name="alert" size={26}/><b>No pudimos cargar Fudia Concierge</b><p>{query.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>query.refetch()}>Reintentar</Button></section>:data&&<>
    <section className="panel concierge-overview">
     <div><span><Icon name="mail" size={22}/></span><section><h2>{data.organizationName}</h2><p>{data.locationName} · El módulo se administra desde la plataforma global de Fudia.</p></section></div>
     <span className={`concierge-state ${data.available?"active":""}`}><i/>{data.available?"Disponible":data.active?"Activado":"No contratado"}</span>
    </section>

    <div className="concierge-config">
     <section className="panel concierge-form">
      <header><h2>Canal oficial de Fudia</h2><p>La empresa no configura números, credenciales de Meta ni activaciones locales. Fudia opera un único bot y un único número de WhatsApp para todas las empresas habilitadas.</p></header>

      {!data.active?<div className="concierge-status-card locked">
       <span><Icon name="lock" size={20}/></span>
       <div><b>Fudia Concierge no está habilitado para esta empresa</b><small>El administrador global de Fudia puede activar el módulo <code>whatsapp_bot</code> cuando forme parte del contrato o plan de la empresa.</small></div>
      </div>:data.available?<div className="concierge-status-card ready">
       <span><Icon name="check" size={20}/></span>
       <div><b>Módulo habilitado por Fudia</b><small>Los QR de las mesas pueden iniciar pedidos en el WhatsApp oficial de Fudia.</small></div>
      </div>:<div className="concierge-status-card warning">
       <span><Icon name="alert" size={20}/></span>
       <div><b>Módulo activado, canal global no disponible</b><small>La activación comercial está correcta, pero el número global de WhatsApp de Fudia todavía no está disponible en la configuración de plataforma.</small></div>
      </div>}

      <div className="concierge-readonly-field">
       <small>NÚMERO OFICIAL</small>
       <b>{data.whatsappPhone||"Gestionado por Fudia"}</b>
       <p>Este número es único para toda la plataforma y no puede ser modificado por la empresa ni por el local.</p>
      </div>

      <div className="concierge-note">La empresa obtiene acceso únicamente mediante el módulo contratado. La activación y desactivación pertenecen al administrador global de Fudia.</div>
     </section>

     <section className="panel concierge-preview">
      <header><h2>Flujo del comensal</h2><p>El QR identifica la empresa, el local y la mesa; el número de WhatsApp es siempre el mismo.</p></header>
      <div className="concierge-flow">
       <span><i>1</i><b>Escanea el QR<small>Fudia identifica empresa, local y mesa mediante un token opaco.</small></b></span>
       <span><i>2</i><b>Abre WhatsApp<small>Todos los restaurantes habilitados usan el número oficial de Fudia.</small></b></span>
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
