"use client";

import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,EmptyState,ErrorState,Icon,IconTile,InlineNotice,PageHeader,Skeleton,Status} from "@/design-system";
import {useFeedback} from "@/providers";
import {courierSteps,dialablePhone,outcomeFeedback,stageMeta} from "../domain/panel";
import type {CourierAction} from "../domain/types";
import {loadCourierView,sendCourierAction} from "../infrastructure/panel-api";
import {useCourierLocation,type LocationStatus} from "./use-courier-location";
import {useStoredFlag} from "./use-stored-flag";
import {OrderDetails} from "./order-details";
import "./partner-panel.css";

const POLL_MS=5_000;

const locationCopy:Record<LocationStatus,string>={
  off:"Actívala para que el cliente siga su pedido.",
  locating:"Buscando tu ubicación…",
  active:"El cliente ve por dónde vas.",
  denied:"Permite la ubicación en el navegador para compartirla.",
  unsupported:"Este navegador no permite compartir la ubicación.",
  error:"No pudimos obtener tu ubicación. Revisa el GPS.",
};

/** Barra de contexto: interruptor de ubicación del dispositivo. */
function LocationToggle({enabled,onToggle,disabled,status}:{enabled:boolean;onToggle:()=>void;disabled:boolean;status:LocationStatus}){
  return <button type="button" className="partner-panel-toggle" aria-pressed={enabled} disabled={disabled} onClick={onToggle}>
    <IconTile name="mapPin" tone={enabled?"primary":"neutral"}/>
    <span><b>Compartir ubicación</b><small>{enabled?locationCopy[status]:locationCopy.off}</small></span>
    <i aria-hidden="true"/>
  </button>;
}

function Frame({title,subtitle,busy,children}:{title:string;subtitle:string;busy?:boolean;children:React.ReactNode}){
  return <main className="partner-panel-stage"><div className="partner-panel-content" aria-busy={busy||undefined}>
    <PageHeader title={title} description={subtitle}/>
    <div className="partner-panel-body">{children}</div>
  </div></main>;
}

/**
 * Enlace privado del motorizado: lo genera su central con su nombre y vale para
 * un solo reparto hasta entregarlo o hasta que la central lo reasigne o libere.
 * Recoge, entrega o reporta «No pude entregar» y comparte su ubicación.
 */
export function CourierPanel({token}:{token:string}){
  const {notify}=useFeedback();
  const client=useQueryClient();
  const queryKey=["delivery-courier",token];
  const [finished,setFinished]=useState(false);
  const view=useQuery({queryKey,queryFn:()=>loadCourierView(token),enabled:!finished,retry:false,staleTime:0,refetchIntervalInBackground:true,refetchInterval:query=>query.state.error?POLL_MS*6:POLL_MS});
  const [sharing,setSharing]=useStoredFlag("fudia-courier-location");
  const [confirming,setConfirming]=useState<"deliver"|"not_delivered"|null>(null);
  const order=view.data?.order;
  const inProgress=!finished&&(order?.stage==="accepted"||order?.stage==="picked_up");
  const location=useCourierLocation(token,sharing&&inProgress);
  const act=useMutation({
    mutationFn:(action:CourierAction)=>sendCourierAction(token,action),
    onSuccess:(result,action)=>{
      setConfirming(null);
      const feedback=outcomeFeedback(result.outcome,order?.orderCode??"",action);
      notify({tone:feedback.tone==="warning"?"info":feedback.tone,title:feedback.title,message:feedback.message});
      if(result.outcome==="delivered"){setFinished(true);return}
      void client.invalidateQueries({queryKey});
    },
    onError:error=>{setConfirming(null);notify({tone:"danger",title:"No se pudo registrar",message:error.message})},
  });
  const toggle=<LocationToggle enabled={sharing} disabled={!inProgress} status={location.status} onToggle={()=>setSharing(!sharing)}/>;

  if(finished)return <Frame title={view.data?.courierName||"Reparto"} subtitle="Delivery Fudia">
    {toggle}
    <section className="panel"><EmptyState icon="circleCheck" title="Entrega registrada" text="Este enlace ya cumplió su reparto. Tu central te enviará el siguiente."/></section>
  </Frame>;
  if(view.isPending)return <Frame title="Reparto" subtitle="Delivery Fudia" busy>
    {toggle}
    <Skeleton height="var(--size-56)" radius="var(--radius-tile)"/><Skeleton height="var(--size-280)" radius="var(--radius-card)"/><span className="sr-only">Cargando pedido</span>
  </Frame>;
  if(!view.data||!order)return <Frame title="Reparto" subtitle="Delivery Fudia">
    {toggle}
    <section className="panel"><ErrorState title="No pudimos abrir el reparto" text={view.error?.message} retrying={view.isFetching} onRetry={()=>void view.refetch()}/></section>
  </Frame>;

  const data=view.data;
  const meta=stageMeta[order.stage];
  const steps=courierSteps(order);
  const phone=dialablePhone(order.customerPhone);
  return <Frame title={data.courierName} subtitle={`Motorizado · ${data.partnerName}`}>
    {toggle}
    {view.isError&&<p className="partner-panel-banner is-danger" role="alert"><Icon name="alert" size={18}/>Sin conexión con Fudia. El pedido puede estar desactualizado; reintentando…</p>}
    <article className="panel" aria-label={`Pedido ${order.orderCode}`}>
      <header><div><h2 className="partner-panel-code">{order.orderCode}</h2><small>{order.organizationName}</small></div><Status tone={meta.tone} icon={meta.icon}>{meta.label}</Status></header>
      <div className="partner-panel-sheet-body">
        {order.issueReportedAt&&<InlineNotice tone="warning" title="Avisamos a tu central y al cliente"/>}
        <OrderDetails order={order} courier={location.position}/>
        {steps&&<footer className="partner-panel-actions is-single">
          <p>{steps.primary.hint}</p>
          <Button icon={steps.primary.action==="pickup"?"shoppingBag":"circleCheck"} disabled={act.isPending||!steps.primary.enabled}
            onClick={()=>steps.primary.action==="deliver"?setConfirming("deliver"):act.mutate(steps.primary.action)}>{steps.primary.label}</Button>
          {steps.secondary&&<Button kind="danger-outline" icon="alert" disabled={act.isPending} onClick={()=>setConfirming("not_delivered")}>{steps.secondary.label}</Button>}
        </footer>}
      </div>
    </article>
    <ConfirmDialog open={confirming==="deliver"} tone="success" title="Confirmar la entrega" description="Confirma solo cuando el cliente ya recibió su pedido." subject={{label:"Pedido",name:order.orderCode}}
      confirmLabel="Confirmar entrega" pending={act.isPending} onCancel={()=>setConfirming(null)} onConfirm={()=>act.mutate("deliver")}/>
    <ConfirmDialog open={confirming==="not_delivered"} tone="danger" title="No pude entregar" description="Antes de reportarlo, llama al cliente al número del pedido, espera unos minutos y vuelve a intentarlo." subject={{label:"Pedido",name:order.orderCode}}
      note="Fudia avisará a tu central y pedirá al cliente que se comunique." confirmLabel="Reportar" pending={act.isPending} onCancel={()=>setConfirming(null)} onConfirm={()=>act.mutate("not_delivered")}>
      {phone&&<a className="partner-panel-link" href={`tel:${phone}`}><Icon name="phone" size={16}/>Llamar al {order.customerPhone}</a>}
    </ConfirmDialog>
  </Frame>;
}
