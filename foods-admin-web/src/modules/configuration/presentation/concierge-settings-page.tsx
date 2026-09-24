"use client";
import "./concierge-settings.css";
import {useEffect,useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,Input,PageHeader} from "@/design-system/page-header";
import {Icon} from "@/design-system/icons";
import {useFeedback} from "@/providers/feedback-provider";
import {useSession} from "@/providers/session-context";
import type {ConciergeSettingsDraft} from "../domain/concierge-settings";
import {getConciergeSettings,saveConciergeSettings} from "../infrastructure/concierge-settings-api";

export function ConciergeSettingsPage(){
 const{can}=useSession();
 const{notify}=useFeedback();
 const client=useQueryClient();
 const canManage=can("organizations.manage");
 const query=useQuery({queryKey:["concierge-settings"],queryFn:getConciergeSettings});
 const[draft,setDraft]=useState<ConciergeSettingsDraft>({active:false,whatsappPhone:""});
 useEffect(()=>{if(query.data)setDraft({active:query.data.active,whatsappPhone:query.data.whatsappPhone})},[query.data]);
 const mutation=useMutation({
  mutationFn:saveConciergeSettings,
  onSuccess:data=>{
   setDraft({active:data.active,whatsappPhone:data.whatsappPhone});
   client.setQueryData(["concierge-settings"],data);
   notify({tone:"success",title:"Fudia Concierge actualizado",message:data.active?"Los QR del local ya pueden iniciar pedidos por WhatsApp.":"Concierge quedó desactivado para este local."});
  },
  onError:error=>notify({tone:"danger",title:"No se pudo guardar",message:error.message}),
 });
 const phoneValid=draft.whatsappPhone===""||/^\+[1-9][0-9]{7,14}$/.test(draft.whatsappPhone);
 const canSave=canManage&&phoneValid&&(!draft.active||draft.whatsappPhone!=="")&&!mutation.isPending;
 return <><PageHeader eyebrow="CONFIGURACIÓN" title="Fudia Concierge" description="Configura los pedidos conversacionales por WhatsApp iniciados desde el QR de cada mesa."/>
  <div className="concierge-settings">
   {query.isLoading?<section className="panel concierge-loading">Cargando configuración de Fudia Concierge…</section>:query.isError?<section className="panel concierge-error"><Icon name="alert" size={26}/><b>No pudimos cargar Fudia Concierge</b><p>{query.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>query.refetch()}>Reintentar</Button></section>:query.data&&<>
    <section className="panel concierge-overview">
     <div><span><Icon name="mail" size={22}/></span><section><h2>{query.data.locationName}</h2><p>{query.data.organizationName} · La configuración se aplica solo al local activo.</p></section></div>
     <span className={`concierge-state ${draft.active?"active":""}`}><i/>{draft.active?"Activo":"Inactivo"}</span>
    </section>
    <div className="concierge-config">
     <form className="panel concierge-form" onSubmit={event=>{event.preventDefault();if(canSave)mutation.mutate({...draft,whatsappPhone:draft.whatsappPhone.trim()})}}>
      <header><h2>Canal de WhatsApp</h2><p>El número es público porque se utiliza para abrir la conversación desde el QR. Las credenciales de Meta permanecen como secretos del servicio.</p></header>
      <label className="concierge-field">Número de WhatsApp
       <Input disabled={!canManage} value={draft.whatsappPhone} onChange={e=>setDraft({...draft,whatsappPhone:e.target.value.replace(/\s/g,"")})} placeholder="+51987654321" inputMode="tel"/>
       <small>Formato internacional E.164: empieza con +, código de país y número. Ejemplo: +51987654321.</small>
      </label>
      <label className="concierge-toggle">
       <input disabled={!canManage} type="checkbox" checked={draft.active} onChange={e=>setDraft({...draft,active:e.target.checked})}/>
       <span><b>Activar Fudia Concierge en este local</b><small>Cuando esté activo, el QR de las mesas mostrará la opción para iniciar el pedido por WhatsApp.</small></span>
      </label>
      {!phoneValid&&<div className="concierge-warning">El número debe usar formato internacional, por ejemplo +51987654321.</div>}
      {draft.active&&!draft.whatsappPhone&&<div className="concierge-warning">Configura el número de WhatsApp antes de activar Concierge.</div>}
      <footer>{canManage?<Button type="submit" icon="save" disabled={!canSave}>{mutation.isPending?"Guardando…":"Guardar"}</Button>:<small>Tu rol puede consultar esta configuración, pero no modificarla.</small>}</footer>
     </form>
     <section className="panel concierge-preview">
      <header><h2>Flujo del comensal</h2><p>Concierge conserva la conversación; Fudia conserva el control del pedido.</p></header>
      <div className="concierge-flow">
       <span><i>1</i><b>Escanea el QR<small>Fudia identifica la mesa y el local.</small></b></span>
       <span><i>2</i><b>Abre WhatsApp<small>El enlace envía un token opaco, no datos internos.</small></b></span>
       <span><i>3</i><b>Conversa con Concierge<small>GPT usa herramientas para consultar la carta real.</small></b></span>
       <span><i>4</i><b>Confirma el pedido<small>foods-backend recalcula precios y disponibilidad.</small></b></span>
       <span><i>5</i><b>Pedido a Cocina<small>La comanda confirmada aparece en KDS con canal WhatsApp.</small></b></span>
      </div>
      <div className="concierge-note">El modelo no puede definir precios ni crear productos. La fuente de verdad sigue siendo foods-backend.</div>
     </section>
    </div>
   </>}
  </div>
 </>;
}
