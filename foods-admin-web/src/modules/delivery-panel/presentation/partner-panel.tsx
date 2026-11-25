"use client";

import {useEffect,useId,useState,type ReactNode} from "react";
import {useRouter} from "next/navigation";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,ErrorState,FormField,Icon,IconTile,InlineNotice,Input,PageHeader,Skeleton,Status,Tag} from "@/design-system";
import {useFeedback} from "@/providers";
import {useTickingClock} from "./use-ticking-clock";
import {canRelease,courierLink,courierNameValue,failCountdown,formatCountdown,needsPickupDecision,outcomeFeedback,pickupAlertMeta,readyForPickup,remainingSeconds,stageMeta,waitLabel} from "../domain/panel";
import type {FailedDelivery,PanelAction,PanelOrder} from "../domain/types";
import {assignCourier,loadPartnerPanel,loadPartnerSession,logoutPartner,needsLogin,needsPasswordChange,sendPanelAction} from "../infrastructure/panel-api";
import {requestAlertPermission,useOfferAlert} from "./use-offer-alert";
import {useStoredFlag} from "@/shared/hooks/use-stored-flag";
import {OrderDetails,hour} from "./order-details";
import {DispatchWorkspace} from "./dispatch-workspace";
import "./partner-panel.css";

const POLL_MS=5_000;

type Confirming={order:PanelOrder;kind:"reject"|"release"|"fail"|"change"};

function DispatchAppHeader({partnerName,operatorName,status,sound,onLogout,leaving}:{partnerName?:string;operatorName?:string;status?:ReactNode;sound:ReactNode;onLogout:()=>void;leaving:boolean}){
  const description=[partnerName||"Delivery Fudia",operatorName].filter(Boolean).join(" · ");
  return <div className="dispatch-app-header"><div className="dispatch-app-header-inner">
    <div className="dispatch-app-identity"><IconTile name="motorbike" tone="primary" appearance="vivid" size="lg"/>
      <PageHeader title="Panel de reparto" description={description}/>
    </div>
    <div className="dispatch-app-header-actions">{status}{sound}
      <button type="button" className="dispatch-app-logout" disabled={leaving} aria-label={operatorName?`Cerrar sesión de ${operatorName}`:"Cerrar sesión"} onClick={onLogout}><Icon name="logout" size={18}/><span>{leaving?"Saliendo…":"Salir"}</span></button>
    </div>
  </div></div>;
}

/** Avance de la recogida, dicho por el backend: preparación, pago en caja o listo. */
function pickupProgress(order:PanelOrder){
  if(!["listo","en_camino","entregado"].includes(order.orderStatus))return "El local aún prepara el pedido.";
  return readyForPickup(order)?"Listo para recoger y pagado en caja.":"Listo; el motorizado paga en caja al recoger.";
}

/** Barra de contexto: interruptor de avisos del dispositivo. */
function SoundToggle({enabled,onToggle}:{enabled:boolean;onToggle:()=>void}){
  return <button type="button" className="partner-panel-toggle" aria-pressed={enabled} onClick={onToggle}>
    <IconTile name="bell" tone={enabled?"primary":"neutral"}/>
    <span><b>Avisos con sonido</b><small>{enabled?"Sonarán los pedidos nuevos y las alertas.":"Actívalos para no perder pedidos ni alertas."}</small></span>
    <i aria-hidden="true"/>
  </button>;
}

/** Motorizado del reparto: sin nombre no hay enlace; con nombre, el enlace se copia o se comparte. */
function CourierBlock({order,busy,assigning,names,onAssign,onChange}:{order:PanelOrder;busy:boolean;assigning:boolean;names:string[];onAssign:(name:string)=>void;onChange?:()=>void}){
  const {notify}=useFeedback();
  const [name,setName]=useState("");
  const namesId=useId();
  const [touched,setTouched]=useState(false);
  const link=typeof window==="undefined"?"":courierLink(window.location.origin,order.courierToken);
  const canShare=typeof navigator!=="undefined"&&typeof navigator.share==="function";
  if(!order.courierName||!link){
    const value=courierNameValue(name);
    return <form className="partner-panel-courier" onSubmit={event=>{event.preventDefault();if(value)onAssign(value)}}>
      <FormField label="Nombre del motorizado" icon="user" error={touched&&name&&!value?"Escribe entre 2 y 60 caracteres.":undefined}>
        <Input value={name} required minLength={2} maxLength={60} list={namesId} disabled={busy} placeholder="Escribe o elige un nombre" autoComplete="off" onBlur={()=>setTouched(true)} onChange={event=>setName(event.target.value)}/>
      </FormField>
      <datalist id={namesId}>{names.map(value=><option key={value} value={value}/>)}</datalist>
      <Button type="submit" icon="userCheck" appearance="strong" disabled={busy||!value} loading={assigning}>{assigning?"Asignando…":"Asignar motorizado"}</Button>
      <p className="dispatch-assignment-hint">{!value?"Escribe un nombre de 2 a 60 caracteres para habilitar la asignación.":"Se generará un enlace privado para que registre la recogida y la entrega."}</p>
    </form>;
  }
  const copy=async()=>{
    try{if(!navigator.clipboard)throw new Error("Clipboard unavailable");await navigator.clipboard.writeText(link);notify({tone:"success",title:"Enlace copiado",message:`Envíaselo a ${order.courierName}.`})}
    catch{notify({tone:"danger",title:"No se pudo copiar",message:"Selecciona el enlace y cópialo desde el campo."})}
  };
  const share=()=>{void navigator.share({title:`Pedido ${order.orderCode}`,text:`Reparto del pedido ${order.orderCode}`,url:link}).catch((error:Error)=>{if(error.name!=="AbortError")notify({tone:"danger",title:"No se pudo compartir",message:"Copia el enlace y envíaselo al motorizado."})})};
  return <div className="partner-panel-courier">
    <p className="partner-panel-courier-name"><Icon name="motorbike" size={18}/><span>Motorizado</span><b>{order.courierName}</b></p>
    <FormField label="Enlace del reparto" icon="link" className="dispatch-share-field"><Input readOnly value={link} aria-label={`Enlace de ${order.courierName}`} onFocus={event=>event.currentTarget.select()}/></FormField>
    <div className={`partner-panel-actions${canShare?"":" is-single"}`}>
      <Button kind="secondary" appearance="strong" icon="link" onClick={copy}>Copiar enlace</Button>
      {canShare&&<Button kind="secondary" appearance="strong" icon="share" onClick={share}>Compartir</Button>}
    </div>
    {/* Ante una alerta, «Cambiar motorizado» está en la decisión: no se repite aquí. */}
    {onChange&&<button type="button" className="partner-panel-link" disabled={busy} onClick={onChange}><Icon name="userCheck" size={16}/>Cambiar motorizado</button>}
  </div>;
}

function OrderCard({order,receivedAt,busy,assigning,names,onAct,onAssign,onConfirm}:{order:PanelOrder;receivedAt:number;busy:boolean;assigning:boolean;names:string[];onAct:(action:PanelAction)=>void;onAssign:(name:string)=>void;onConfirm:(kind:Confirming["kind"])=>void}){
  // Los relojes (oferta, espera de recogida y entrega fallida) avanzan entre consultas
  // dentro de la tarjeta: solo ella se vuelve a renderizar cada segundo.
  const clock=useTickingClock(order.stage==="offer"||order.pickupWaitSeconds>0||Boolean(order.issueReportedAt&&order.failAvailableInSeconds>0),receivedAt);
  const elapsed=Math.max(0,Math.floor((clock-receivedAt)/1000));
  const remaining=remainingSeconds(order,receivedAt,clock);
  const wait=order.pickupWaitSeconds>0?order.pickupWaitSeconds+elapsed:0;
  const failIn=failCountdown(order,receivedAt,clock);
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
    {order.stage==="offer"&&<p className="partner-panel-timer" role="timer" aria-live="off"><IconTile name="clock" tone="warning" size="sm"/><span>Responde en</span><b>{formatCountdown(remaining)}</b></p>}
    {alert&&(deciding
      ?<InlineNotice tone={alert.tone} title={`${alert.title} · ${waitLabel(wait)}`}>{alert.message}</InlineNotice>
      :<InlineNotice tone="info" title={`Esperando la recogida · ${waitLabel(wait)}`}>Mantienes al motorizado actual.</InlineNotice>)}
    {deciding&&<div className="partner-panel-decision" role="group" aria-label="Decisión sobre la recogida">
      <Button kind="secondary" appearance="strong" icon="clock" disabled={busy} onClick={()=>onAct("keep")}>Mantener</Button>
      <Button kind="secondary" appearance="strong" icon="userCheck" disabled={busy} onClick={()=>onConfirm("change")}>Cambiar motorizado</Button>
      <Button kind="danger-outline" appearance="strong" icon="undo" disabled={busy} onClick={()=>onConfirm("release")}>Liberar</Button>
    </div>}
    {reported&&<InlineNotice tone="warning" title="El motorizado no pudo entregar">Fudia pidió al cliente que se comunique. Si lo entrega, se cierra solo.</InlineNotice>}
    {active&&<section className="partner-panel-progress" aria-label="Avance del reparto">
      <div className="dispatch-assignment-heading"><IconTile name="userCheck" tone="primary"/><div><h3>{order.courierName?"Motorizado asignado":"Asigna este reparto"}</h3>{!order.courierName&&<p>Elige al responsable antes de la recogida.</p>}</div></div>
      <CourierBlock order={order} names={names} busy={busy} assigning={assigning} onAssign={onAssign} onChange={deciding?undefined:()=>onConfirm("change")}/>
      {order.stage==="accepted"&&<p className="partner-panel-note"><Icon name="store" size={16}/>{pickupProgress(order)}</p>}
      {order.stage==="picked_up"&&order.pickedUpAt&&<p className="partner-panel-note"><Icon name="motorbike" size={16}/>Recogido a las {hour(order.pickedUpAt)}</p>}
    </section>}
    <OrderDetails order={order} courier={null}/>
    {order.stage==="offer"&&<footer className="partner-panel-actions">
      <Button kind="secondary" appearance="strong" disabled={busy} onClick={()=>onConfirm("reject")}>Rechazar</Button>
      <Button icon="check" appearance="strong" disabled={busy||remaining<=0} onClick={()=>onAct("accept")}>{remaining<=0?"Oferta vencida":"Aceptar pedido"}</Button>
    </footer>}
    {canRelease(order)&&!deciding&&<footer className="partner-panel-actions is-single">
      <Button kind="danger-outline" appearance="strong" icon="undo" disabled={busy} onClick={()=>onConfirm("release")}>Liberar pedido</Button>
    </footer>}
    {reported&&<footer className="partner-panel-actions is-single">
      {failIn!==null&&failIn>0&&<p>Podrás registrarla en {formatCountdown(failIn)} si el cliente no responde.</p>}
      <Button kind="danger-outline" appearance="strong" icon="alert" disabled={busy||failIn===null||failIn>0} onClick={()=>onConfirm("fail")}>Registrar entrega fallida</Button>
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
    <dl className="partner-panel-breakdown">
      <div><dt>Pagó el motorizado</dt><dd>{money(item.foodAmount)}</dd></div>
      <div><dt>Le reembolsas</dt><dd>{money(item.courierReimbursement)}</dd></div>
      <div><dt>Asume tu central</dt><dd>{money(item.partnerShare)}</dd></div>
      <div><dt>Asume Fudia</dt><dd>{money(item.fudiaShare)}</dd></div>
    </dl>
    {item.validationNotes.length>0&&<ul className="partner-panel-validation" aria-label="Validación de la incidencia">{item.validationNotes.map(note=><li key={note}>{note}</li>)}</ul>}
  </article>;
}

/**
 * Panel del proveedor de Delivery Fudia. El operador de la central entra con
 * su cuenta (la sesión vence a las 12 horas): recibe los pedidos, los acepta, registra al motorizado (que recoge y
 * entrega desde su propio enlace), decide ante las alertas de recogida y
 * registra las entregas fallidas. Sigue las plantillas de la aplicación:
 * `PageHeader`, barra de contexto con los avisos y paneles por sección.
 */
export function PartnerPanel(){
  const {notify}=useFeedback();
  const client=useQueryClient();
  const router=useRouter();
  const queryKey=["delivery-partner-panel"];
  const panel=useQuery({queryKey,queryFn:loadPartnerPanel,retry:false,staleTime:0,refetchIntervalInBackground:true,refetchInterval:query=>query.state.error?POLL_MS*6:POLL_MS});
  // Tras un corte de conexión se reintenta junto con el panel para recuperar el nombre del operador.
  const session=useQuery({queryKey:["delivery-partner-session"],queryFn:loadPartnerSession,retry:false,staleTime:60_000,refetchInterval:query=>query.state.error?POLL_MS*6:false});
  const [leaving,setLeaving]=useState(false);
  // Sesión vencida, cerrada en otro dispositivo o con contraseña temporal: vuelve al acceso.
  const signedOut=needsLogin(panel.error)||needsPasswordChange(panel.error)||needsLogin(session.error);
  useEffect(()=>{
    if(!signedOut)return;
    client.removeQueries({queryKey});
    router.replace("/delivery-panel/login");
  // eslint-disable-next-line react-hooks/exhaustive-deps
  },[signedOut]);
  const logout=async()=>{
    setLeaving(true);
    try{await logoutPartner()}catch{/* La cookie se borra igual al volver al acceso. */}
    client.clear();
    router.replace("/delivery-panel/login");
    router.refresh();
  };
  const [alerts,setAlerts]=useStoredFlag("fudia-panel-alerts");
  const [confirming,setConfirming]=useState<Confirming|null>(null);
  const [nextCourier,setNextCourier]=useState("");
  const orders=panel.data?.orders??[];
  const failed=panel.data?.failedDeliveries??[];
  useOfferAlert(panel.data?.orders,alerts);
  const done=(outcome:Parameters<typeof outcomeFeedback>[0],order:PanelOrder,action?:PanelAction)=>{
    setConfirming(null);
    setNextCourier("");
    const feedback=outcomeFeedback(outcome,order.orderCode,action);
    notify({tone:feedback.tone==="warning"?"info":feedback.tone,title:feedback.title,message:feedback.message});
    void client.invalidateQueries({queryKey});
  };
  const onError=(error:Error)=>{
    setConfirming(null);
    if(needsLogin(error)||needsPasswordChange(error)){void client.invalidateQueries({queryKey});return}
    notify({tone:"danger",title:"No se pudo registrar",message:error.message});
  };
  const act=useMutation({
    mutationFn:({order,action}:{order:PanelOrder;action:PanelAction})=>sendPanelAction(action,order.token),
    onSuccess:(result,{order,action})=>done(result.outcome,order,action),
    onError,
  });
  const assign=useMutation({
    mutationFn:({order,name}:{order:PanelOrder;name:string})=>assignCourier(order.token,name),
    onSuccess:(result,{order})=>done(result.outcome,order),
    onError,
  });
  const busy=act.isPending||assign.isPending||panel.isError;
  const names=[...new Set(orders.map(order=>order.courierName.trim()).filter(Boolean))];
  const card=(order:PanelOrder)=><OrderCard key={order.token} order={order} names={names} busy={busy} assigning={assign.isPending}
    receivedAt={panel.dataUpdatedAt}
    onAct={action=>act.mutate({order,action})}
    onAssign={name=>assign.mutate({order,name})}
    onConfirm={kind=>{setNextCourier("");setConfirming({order,kind})}}/>;
  const operatorName=session.data?.operator.name;
  const soundToggle=<SoundToggle enabled={alerts} onToggle={()=>{if(!alerts)requestAlertPermission();setAlerts(!alerts)}}/>;

  if(panel.isPending||signedOut)return <main className="partner-panel-stage is-dispatch-app"><DispatchAppHeader sound={soundToggle} operatorName={operatorName} onLogout={()=>void logout()} leaving={leaving}/><div className="partner-panel-content is-dispatch" aria-busy="true">
    <div className="partner-panel-body">
      <Skeleton height="var(--size-48)" radius="var(--radius-pill)"/><div className="dispatch-columns"><Skeleton height="var(--size-480)" radius="var(--radius-card)"/><Skeleton height="var(--size-480)" radius="var(--radius-card)"/></div><span className="sr-only">Cargando pedidos</span>
    </div>
  </div></main>;
  if(!panel.data)return <main className="partner-panel-stage is-dispatch-app"><DispatchAppHeader sound={soundToggle} operatorName={operatorName} onLogout={()=>void logout()} leaving={leaving}/><div className="partner-panel-content is-dispatch">
    <div className="partner-panel-body">
      <section className="panel"><ErrorState title="No pudimos abrir el panel" text={panel.error?.message} retrying={panel.isFetching} onRetry={()=>void panel.refetch()}/></section>
    </div>
  </div></main>;

  const data=panel.data;
  const live=!data.active?{label:"Pausado",tone:"paused"}:panel.isError?{label:"Reconectando…",tone:"offline"}:{label:"Conectado",tone:"active"};
  const target=confirming?.order;
  const changeName=courierNameValue(nextCourier);
  return <main className="partner-panel-stage is-dispatch-app"><DispatchAppHeader partnerName={data.partnerName} operatorName={operatorName} onLogout={()=>void logout()} leaving={leaving} sound={soundToggle}
    status={<span className={`partner-panel-live is-${live.tone}`}><i aria-hidden="true"/>{live.label}</span>}/><div className="partner-panel-content is-dispatch">
    <div className="partner-panel-body">
      {!data.active&&<p className="partner-panel-banner" role="status"><Icon name="info" size={18}/>Tu cuenta está pausada: no recibirás pedidos nuevos. Puedes terminar los que ya tienes.</p>}
      {panel.isError&&<p className="partner-panel-banner is-danger" role="alert"><Icon name="alert" size={18}/>Sin conexión con Fudia. Los pedidos pueden estar desactualizados; reintentando…</p>}
      <DispatchWorkspace orders={orders} receivedAt={panel.dataUpdatedAt} renderOrder={card}/>
      {failed.length>0&&<section className="panel partner-panel-section" aria-labelledby="partner-panel-failed">
        <header><h2 id="partner-panel-failed">Entregas fallidas</h2><span className="partner-panel-count" data-empty={failed.length===0||undefined}>{failed.length}</span></header>
        <div className="partner-panel-cards">{failed.map(item=><FailedCard key={`${item.orderCode}-${item.registeredAt}`} item={item}/>)}</div>
      </section>}
    </div>
  </div>
  <ConfirmDialog open={confirming?.kind==="reject"} tone="danger" title="Rechazar este pedido" description="Ya no podrás tomarlo; seguirá disponible para otras centrales." subject={target?{label:"Pedido",name:target.orderCode}:undefined}
    confirmLabel="Rechazar pedido" pending={act.isPending} confirmDisabled={panel.isError} onCancel={()=>setConfirming(null)} onConfirm={()=>target&&act.mutate({order:target,action:"reject"})}/>
  <ConfirmDialog open={confirming?.kind==="release"} tone="danger" title="Liberar el pedido" description="Se ofrecerá a otras centrales y el enlace del motorizado dejará de funcionar." subject={target?{label:"Pedido",name:target.orderCode}:undefined}
    confirmLabel="Liberar pedido" pending={act.isPending} confirmDisabled={panel.isError} onCancel={()=>setConfirming(null)} onConfirm={()=>target&&act.mutate({order:target,action:"release"})}/>
  <ConfirmDialog open={confirming?.kind==="change"} tone="danger" title="Cambiar motorizado" description="El enlace actual dejará de funcionar y generaremos uno nuevo. El avance del pedido se mantiene." subject={target?.courierName?{label:"Motorizado actual",name:target.courierName}:undefined}
    confirmLabel="Generar enlace nuevo" pending={assign.isPending} confirmDisabled={panel.isError||!changeName||changeName===target?.courierName} onCancel={()=>setConfirming(null)} onConfirm={()=>target&&changeName&&assign.mutate({order:target,name:changeName})}>
    <FormField label="Nombre del nuevo motorizado" icon="user"><Input value={nextCourier} maxLength={60} autoComplete="off" onChange={event=>setNextCourier(event.target.value)}/></FormField>
  </ConfirmDialog>
  <ConfirmDialog open={confirming?.kind==="fail"} tone="danger" title="Registrar entrega fallida" description="Fudia conservará los datos y montos del pedido para gestionar el reclamo." subject={target?{label:"Pedido",name:target.orderCode}:undefined}
    confirmLabel="Registrar entrega fallida" pending={act.isPending} confirmDisabled={panel.isError} onCancel={()=>setConfirming(null)} onConfirm={()=>target&&act.mutate({order:target,action:"fail"})}>
    <ul className="partner-panel-validation">
      <li>Reembolsas al motorizado el 100 % de lo que pagó por la comida{target?` (${`${target.currencySymbol} ${target.subtotal}`.trim()})`:""}.</li>
      <li>La pérdida se reparte 50 % Fudia y 50 % tu central, sujeta a la validación de la incidencia y a los límites de Fudia.</li>
    </ul>
  </ConfirmDialog>
  </main>;
}
