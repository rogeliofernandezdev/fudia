"use client";

import {useEffect,useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,EmptyState,ErrorState,FormField,Icon,InlineNotice,Input,Skeleton,Status,Tag} from "@/design-system";
import {useFeedback} from "@/providers";
import {canRelease,courierLink,courierNameValue,failCountdown,formatCountdown,groupOrders,needsPickupDecision,outcomeFeedback,pickupAlertMeta,readyForPickup,remainingSeconds,stageMeta,waitLabel} from "../domain/panel";
import type {FailedDelivery,PanelAction,PanelOrder} from "../domain/types";
import {assignCourier,loadPartnerPanel,sendPanelAction} from "../infrastructure/panel-api";
import {requestAlertPermission,useOfferAlert} from "./use-offer-alert";
import {useStoredFlag} from "./use-stored-flag";
import {OrderDetails,hour} from "./order-details";
import "./partner-panel.css";

const POLL_MS=5_000;

type Confirming={order:PanelOrder;kind:"reject"|"release"|"fail"|"change"};

/** Avance de la recogida, dicho por el backend: preparación, pago en caja o listo. */
function pickupProgress(order:PanelOrder){
  if(!["listo","en_camino","entregado"].includes(order.orderStatus))return "El local aún prepara el pedido.";
  return readyForPickup(order)?"Listo para recoger y pagado en caja.":"Listo; el motorizado paga en caja al recoger.";
}

/** Motorizado del reparto: sin nombre no hay enlace; con nombre, el enlace se copia o se comparte. */
function CourierBlock({order,busy,onAssign,onChange}:{order:PanelOrder;busy:boolean;onAssign:(name:string)=>void;onChange?:()=>void}){
  const {notify}=useFeedback();
  const [name,setName]=useState("");
  const link=typeof window==="undefined"?"":courierLink(window.location.origin,order.courierToken);
  const canShare=typeof navigator!=="undefined"&&typeof navigator.share==="function";
  if(!order.courierName||!link){
    const value=courierNameValue(name);
    return <form className="partner-panel-courier" onSubmit={event=>{event.preventDefault();if(value)onAssign(value)}}>
      <FormField label="Nombre del motorizado" icon="user">
        <Input value={name} maxLength={60} autoComplete="off" onChange={event=>setName(event.target.value)}/>
      </FormField>
      <Button type="submit" icon="link" disabled={busy||!value} loading={busy}>{busy?"Generando…":"Generar enlace"}</Button>
    </form>;
  }
  const copy=()=>{void navigator.clipboard?.writeText(link).then(()=>notify({tone:"success",title:"Enlace copiado",message:`Envíaselo a ${order.courierName}.`}),()=>notify({tone:"danger",title:"No se pudo copiar",message:"Mantén presionado el enlace para copiarlo."}))};
  const share=()=>{void navigator.share({title:`Pedido ${order.orderCode}`,text:`Reparto del pedido ${order.orderCode}`,url:link}).catch(()=>undefined)};
  return <div className="partner-panel-courier">
    <p className="partner-panel-courier-name"><Icon name="motorbike" size={18}/><span>Motorizado</span><b>{order.courierName}</b></p>
    <Input readOnly value={link} aria-label={`Enlace de ${order.courierName}`} onFocus={event=>event.currentTarget.select()}/>
    <div className={`partner-panel-actions${canShare?"":" is-single"}`}>
      <Button kind="secondary" icon="link" onClick={copy}>Copiar enlace</Button>
      {canShare&&<Button kind="secondary" icon="share" onClick={share}>Compartir</Button>}
    </div>
    {/* Ante una alerta, «Cambiar motorizado» está en la decisión: no se repite aquí. */}
    {onChange&&<button type="button" className="partner-panel-link" disabled={busy} onClick={onChange}><Icon name="userCheck" size={16}/>Cambiar motorizado</button>}
  </div>;
}

function OrderCard({order,remaining,wait,failIn,busy,onAct,onAssign,onConfirm}:{order:PanelOrder;remaining:number;wait:number;failIn:number|null;busy:boolean;onAct:(action:PanelAction)=>void;onAssign:(name:string)=>void;onConfirm:(kind:Confirming["kind"])=>void}){
  const meta=stageMeta[order.stage];
  const active=order.stage==="accepted"||order.stage==="picked_up";
  const alert=order.stage==="accepted"&&order.pickupAlert?pickupAlertMeta[order.pickupAlert]:null;
  const deciding=needsPickupDecision(order);
  const reported=order.stage==="picked_up"&&Boolean(order.issueReportedAt);
  return <article className={`partner-panel-card is-${order.stage}`} aria-label={`Pedido ${order.orderCode}`}>
    <header className="partner-panel-card-head">
      <div><b>{order.orderCode}</b><span>{order.organizationName}</span></div>
      <Status tone={meta.tone} icon={meta.icon}>{meta.label}</Status>
    </header>
    {order.stage==="offer"&&<p className="partner-panel-timer" role="timer" aria-live="off"><Icon name="clock" size={16}/>Responde en <b>{formatCountdown(remaining)}</b></p>}
    {alert&&(deciding
      ?<InlineNotice tone={alert.tone} title={`${alert.title} · ${waitLabel(wait)}`}>{alert.message}</InlineNotice>
      :<InlineNotice tone="info" title={`Esperando la recogida · ${waitLabel(wait)}`}>Mantienes al motorizado actual.</InlineNotice>)}
    {deciding&&<div className="partner-panel-decision" role="group" aria-label="Decisión sobre la recogida">
      <Button kind="secondary" icon="clock" disabled={busy} onClick={()=>onAct("keep")}>Mantener</Button>
      <Button kind="secondary" icon="userCheck" disabled={busy} onClick={()=>onConfirm("change")}>Cambiar motorizado</Button>
      <Button kind="danger-outline" icon="undo" disabled={busy} onClick={()=>onConfirm("release")}>Liberar</Button>
    </div>}
    {reported&&<InlineNotice tone="warning" title="El motorizado no pudo entregar">Fudia pidió al cliente que se comunique. Si lo entrega, se cierra solo.</InlineNotice>}
    <OrderDetails order={order} courier={null}/>
    {active&&<section className="partner-panel-progress" aria-label="Avance del reparto">
      {order.stage==="accepted"&&<p className="partner-panel-note"><Icon name="store" size={16}/>{pickupProgress(order)}</p>}
      {order.stage==="picked_up"&&order.pickedUpAt&&<p className="partner-panel-note"><Icon name="motorbike" size={16}/>Recogido a las {hour(order.pickedUpAt)}</p>}
      <CourierBlock order={order} busy={busy} onAssign={onAssign} onChange={deciding?undefined:()=>onConfirm("change")}/>
    </section>}
    {order.stage==="offer"&&<footer className="partner-panel-actions">
      <Button kind="secondary" disabled={busy} onClick={()=>onConfirm("reject")}>Rechazar</Button>
      <Button icon="check" disabled={busy} onClick={()=>onAct("accept")}>Aceptar</Button>
    </footer>}
    {canRelease(order)&&!deciding&&<footer className="partner-panel-actions is-single">
      <Button kind="danger-outline" icon="undo" disabled={busy} onClick={()=>onConfirm("release")}>Liberar pedido</Button>
    </footer>}
    {reported&&<footer className="partner-panel-actions is-single">
      {failIn!==null&&failIn>0&&<p>Podrás registrarla en {formatCountdown(failIn)} si el cliente no responde.</p>}
      <Button kind="danger-outline" icon="alert" disabled={busy||failIn===null||failIn>0} onClick={()=>onConfirm("fail")}>Registrar entrega fallida</Button>
    </footer>}
  </article>;
}

function FailedCard({item}:{item:FailedDelivery}){
  const money=(amount:string)=>`${item.currencySymbol} ${amount}`.trim();
  return <article className="partner-panel-card is-failed" aria-label={`Entrega fallida ${item.orderCode}`}>
    <header className="partner-panel-card-head">
      <div><b>{item.orderCode}</b><span>{item.customerName||"Sin nombre"} · {hour(item.registeredAt)}</span></div>
      <Tag icon={item.fudiaShareEligible?"circleCheck":"circleMinus"}>{item.fudiaShareEligible?"Con aporte Fudia":"Sin aporte Fudia"}</Tag>
    </header>
    <p className="partner-panel-note"><Icon name="mapPin" size={16}/>{item.address||"Dirección por confirmar"}</p>
    {item.courierName&&<p className="partner-panel-note"><Icon name="motorbike" size={16}/>{item.courierName}</p>}
    <dl className="partner-panel-money">
      <div><dt>Pagó el motorizado</dt><dd>{money(item.foodAmount)}</dd></div>
      <div><dt>Le reembolsas</dt><dd>{money(item.courierReimbursement)}</dd></div>
      <div><dt>Asume tu central</dt><dd>{money(item.partnerShare)}</dd></div>
      <div><dt>Asume Fudia</dt><dd>{money(item.fudiaShare)}</dd></div>
    </dl>
    {item.validationNotes.length>0&&<ul className="partner-panel-validation" aria-label="Validación de la incidencia">{item.validationNotes.map(note=><li key={note}>{note}</li>)}</ul>}
  </article>;
}

/**
 * Panel del proveedor de Delivery Fudia. La central lo abre con su enlace
 * privado: recibe los pedidos, los acepta, registra al motorizado (que recoge y
 * entrega desde su propio enlace), decide ante las alertas de recogida y
 * registra las entregas fallidas.
 */
export function PartnerPanel({token}:{token:string}){
  const {notify}=useFeedback();
  const client=useQueryClient();
  const queryKey=["delivery-partner-panel",token];
  const panel=useQuery({queryKey,queryFn:()=>loadPartnerPanel(token),retry:false,staleTime:0,refetchIntervalInBackground:true,refetchInterval:query=>query.state.error?POLL_MS*6:POLL_MS});
  const [alerts,setAlerts]=useStoredFlag("fudia-panel-alerts");
  const [confirming,setConfirming]=useState<Confirming|null>(null);
  const [nextCourier,setNextCourier]=useState("");
  const [showDelivered,setShowDelivered]=useState(false);
  const [now,setNow]=useState(0);
  const orders=panel.data?.orders??[];
  const groups=groupOrders(orders);
  const failed=panel.data?.failedDeliveries??[];
  useOfferAlert(panel.data?.orders,alerts);
  // Los relojes (oferta, espera de recogida y entrega fallida) avanzan entre consultas.
  const ticking=groups.offers.length>0||groups.active.some(order=>order.pickupWaitSeconds>0||(order.issueReportedAt&&order.failAvailableInSeconds>0));
  useEffect(()=>{
    if(!ticking)return;
    const timer=window.setInterval(()=>setNow(Date.now()),1000);
    return ()=>window.clearInterval(timer);
  },[ticking]);
  const clock=now||panel.dataUpdatedAt;
  const elapsed=Math.max(0,Math.floor((clock-panel.dataUpdatedAt)/1000));
  const done=(outcome:Parameters<typeof outcomeFeedback>[0],order:PanelOrder,action?:PanelAction)=>{
    setConfirming(null);
    setNextCourier("");
    const feedback=outcomeFeedback(outcome,order.orderCode,action);
    notify({tone:feedback.tone==="warning"?"info":feedback.tone,title:feedback.title,message:feedback.message});
    void client.invalidateQueries({queryKey});
  };
  const onError=(error:Error)=>{setConfirming(null);notify({tone:"danger",title:"No se pudo registrar",message:error.message})};
  const act=useMutation({
    mutationFn:({order,action}:{order:PanelOrder;action:PanelAction})=>sendPanelAction(token,action,order.token),
    onSuccess:(result,{order,action})=>done(result.outcome,order,action),
    onError,
  });
  const assign=useMutation({
    mutationFn:({order,name}:{order:PanelOrder;name:string})=>assignCourier(token,order.token,name),
    onSuccess:(result,{order})=>done(result.outcome,order),
    onError,
  });
  const busy=act.isPending||assign.isPending;
  const card=(order:PanelOrder)=><OrderCard key={order.token} order={order} busy={busy}
    remaining={remainingSeconds(order,panel.dataUpdatedAt,clock)}
    wait={order.pickupWaitSeconds>0?order.pickupWaitSeconds+elapsed:0}
    failIn={failCountdown(order,panel.dataUpdatedAt,clock)}
    onAct={action=>act.mutate({order,action})}
    onAssign={name=>assign.mutate({order,name})}
    onConfirm={kind=>{setNextCourier("");setConfirming({order,kind})}}/>;

  if(panel.isPending)return <main className="partner-panel-stage"><section className="partner-panel" aria-busy="true">
    <header className="partner-panel-header"><div><h1>Panel de reparto</h1><p>Cargando tus pedidos…</p></div></header>
    <div className="partner-panel-body"><Skeleton height="var(--size-56)" radius="var(--radius-tile)"/><Skeleton height="var(--size-280)" radius="var(--radius-card)"/><Skeleton height="var(--size-280)" radius="var(--radius-card)"/><span className="sr-only">Cargando pedidos</span></div>
  </section></main>;
  if(!panel.data)return <main className="partner-panel-stage"><section className="partner-panel">
    <header className="partner-panel-header"><div><h1>Panel de reparto</h1><p>Delivery Fudia</p></div></header>
    <div className="partner-panel-body"><ErrorState title="No pudimos abrir el panel" text={panel.error?.message} retrying={panel.isFetching} onRetry={()=>void panel.refetch()}/></div>
  </section></main>;

  const data=panel.data;
  const empty=!groups.offers.length&&!groups.active.length;
  const target=confirming?.order;
  const changeName=courierNameValue(nextCourier);
  return <main className="partner-panel-stage"><section className="partner-panel" aria-label={`Panel de reparto de ${data.partnerName}`}>
    <header className="partner-panel-header">
      <span className="partner-panel-badge"><Icon name="motorbike" size={24}/></span>
      <div><h1>{data.partnerName}</h1><p>Panel de reparto · Delivery Fudia</p></div>
    </header>
    <div className="partner-panel-body">
      <div className="partner-panel-controls">
        <button type="button" className="partner-panel-toggle" aria-pressed={alerts} onClick={()=>{if(!alerts)requestAlertPermission();setAlerts(!alerts)}}>
          <Icon name="bell" size={18}/><span><b>Avisos con sonido</b><small>{alerts?"Sonarán los pedidos nuevos y las alertas.":"Actívalos para no perder pedidos ni alertas."}</small></span><i aria-hidden="true"/>
        </button>
      </div>
      {!data.active&&<p className="partner-panel-banner" role="status"><Icon name="info" size={18}/>Tu cuenta está pausada: no recibirás pedidos nuevos. Puedes terminar los que ya tienes.</p>}
      {panel.isError&&<p className="partner-panel-banner is-danger" role="alert"><Icon name="alert" size={18}/>Sin conexión con Fudia. Los pedidos pueden estar desactualizados; reintentando…</p>}
      {groups.offers.length>0&&<section aria-labelledby="partner-panel-offers"><h2 id="partner-panel-offers">Pedidos nuevos<span className="partner-panel-count">{groups.offers.length}</span></h2>{groups.offers.map(card)}</section>}
      {groups.active.length>0&&<section aria-labelledby="partner-panel-active"><h2 id="partner-panel-active">En curso<span className="partner-panel-count">{groups.active.length}</span></h2>{groups.active.map(card)}</section>}
      {empty&&<EmptyState icon="motorbike" title="Sin pedidos por ahora" text="Deja esta pantalla abierta: los pedidos nuevos aparecen aquí al instante." className="partner-panel-empty"/>}
      {groups.delivered.length>0&&<section aria-labelledby="partner-panel-delivered">
        <h2 id="partner-panel-delivered"><button type="button" className="partner-panel-disclosure" aria-expanded={showDelivered} onClick={()=>setShowDelivered(!showDelivered)}>Entregados hoy<span className="partner-panel-count">{groups.delivered.length}</span><Icon name="chevron" size={18}/></button></h2>
        {showDelivered&&groups.delivered.map(card)}
      </section>}
      {failed.length>0&&<section aria-labelledby="partner-panel-failed">
        <h2 id="partner-panel-failed">Entregas fallidas<span className="partner-panel-count">{failed.length}</span></h2>
        {failed.map(item=><FailedCard key={`${item.orderCode}-${item.registeredAt}`} item={item}/>)}
      </section>}
    </div>
  </section>
  <ConfirmDialog open={confirming?.kind==="reject"} tone="danger" title="Rechazar este pedido" description="Ya no podrás tomarlo; seguirá disponible para otras centrales." subject={target?{label:"Pedido",name:target.orderCode}:undefined}
    confirmLabel="Rechazar pedido" pending={act.isPending} onCancel={()=>setConfirming(null)} onConfirm={()=>target&&act.mutate({order:target,action:"reject"})}/>
  <ConfirmDialog open={confirming?.kind==="release"} tone="danger" title="Liberar el pedido" description="Se ofrecerá a otras centrales y el enlace del motorizado dejará de funcionar." subject={target?{label:"Pedido",name:target.orderCode}:undefined}
    confirmLabel="Liberar pedido" pending={act.isPending} onCancel={()=>setConfirming(null)} onConfirm={()=>target&&act.mutate({order:target,action:"release"})}/>
  <ConfirmDialog open={confirming?.kind==="change"} tone="danger" title="Cambiar motorizado" description="El enlace actual dejará de funcionar y generaremos uno nuevo. El avance del pedido se mantiene." subject={target?.courierName?{label:"Motorizado actual",name:target.courierName}:undefined}
    confirmLabel="Generar enlace nuevo" pending={assign.isPending} confirmDisabled={!changeName||changeName===target?.courierName} onCancel={()=>setConfirming(null)} onConfirm={()=>target&&changeName&&assign.mutate({order:target,name:changeName})}>
    <FormField label="Nombre del nuevo motorizado" icon="user"><Input value={nextCourier} maxLength={60} autoComplete="off" onChange={event=>setNextCourier(event.target.value)}/></FormField>
  </ConfirmDialog>
  <ConfirmDialog open={confirming?.kind==="fail"} tone="danger" title="Registrar entrega fallida" description="Fudia conservará los datos y montos del pedido para gestionar el reclamo." subject={target?{label:"Pedido",name:target.orderCode}:undefined}
    confirmLabel="Registrar entrega fallida" pending={act.isPending} onCancel={()=>setConfirming(null)} onConfirm={()=>target&&act.mutate({order:target,action:"fail"})}>
    <ul className="partner-panel-validation">
      <li>Reembolsas al motorizado el 100 % de lo que pagó por la comida{target?` (${`${target.currencySymbol} ${target.subtotal}`.trim()})`:""}.</li>
      <li>La pérdida se reparte 50 % Fudia y 50 % tu central, sujeta a la validación de la incidencia y a los límites de Fudia.</li>
    </ul>
  </ConfirmDialog>
  </main>;
}
