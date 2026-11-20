"use client";

import {useEffect,useRef,useState,type ReactNode} from "react";
import {Button,EmptyState,Icon,IconTile,SearchField,Status,Tabs} from "@/design-system";
import {courierWorkload,DISPATCH_BATCH_SIZE,dispatchOrders,dispatchWindow,needsAssignment,needsAttention,type DispatchFilter} from "../domain/dispatch";
import {formatCountdown,remainingSeconds,stageMeta,waitLabel} from "../domain/panel";
import type {PanelOrder} from "../domain/types";
import {useTickingClock} from "./use-ticking-clock";
import "./dispatch-workspace.css";

/** Cuenta regresiva de una oferta en la lista: su propio reloj, sin re-renderizar la lista. */
function OfferCountdown({order,receivedAt}:{order:PanelOrder;receivedAt:number}){
  const clock=useTickingClock(true,receivedAt);
  return <>{formatCountdown(remainingSeconds(order,receivedAt,clock))}</>;
}

export function DispatchWorkspace({orders,receivedAt,renderOrder}:{orders:PanelOrder[];receivedAt:number;renderOrder:(order:PanelOrder)=>ReactNode}){
  const [filter,setFilter]=useState<DispatchFilter>("all");
  const [search,setSearch]=useState("");
  const [attentionOnly,setAttentionOnly]=useState(false);
  const [selectedCourier,setSelectedCourier]=useState("");
  const [visibleCount,setVisibleCount]=useState(DISPATCH_BATCH_SIZE);
  const [selectedToken,setSelectedToken]=useState(()=>dispatchOrders(orders,"all","")[0]?.token??"");
  const [mobileDetail,setMobileDetail]=useState(false);
  const detail=useRef<HTMLDivElement>(null);
  const list=useRef<HTMLElement>(null);
  const stream=useRef<HTMLDivElement>(null);
  const loadTarget=useRef<HTMLDivElement>(null);
  const filtered=dispatchOrders(orders,filter,search,attentionOnly,selectedCourier);
  const visible=dispatchWindow(filtered,visibleCount);
  const total=visible.total;
  const hasMore=visible.hasMore;
  const viewKey=JSON.stringify([filter,search,attentionOnly,selectedCourier]);
  // Tras asignar desde «Por asignar», conserva el detalle para compartir el enlace
  // recién generado, aunque ese pedido ya haya salido del listado filtrado.
  const assignedSelection=mobileDetail&&filter==="unassigned"?orders.find(order=>order.token===selectedToken&&order.stage==="accepted"):undefined;
  const selected=filtered.find(order=>order.token===selectedToken)??assignedSelection??visible.items[0];
  const active=orders.filter(order=>order.stage!=="delivered");
  const couriers=courierWorkload(orders);
  const attention=active.filter(needsAttention).length;
  const unassigned=active.filter(needsAssignment).length;
  const newOrders=active.filter(order=>order.stage==="offer").length;
  const tabs=[
    {value:"all" as const,label:"Despacho",icon:"route" as const,tone:"primary" as const,count:active.length},
    {value:"unassigned" as const,label:"Por asignar",icon:"userCheck" as const,tone:"orange" as const,count:unassigned},
    {value:"on_route" as const,label:"En camino",icon:"motorbike" as const,tone:"indigo" as const,count:active.filter(order=>order.stage==="picked_up").length},
    {value:"delivered" as const,label:"Entregados",icon:"circleCheck" as const,tone:"green" as const,count:orders.length-active.length},
  ];
  const currentToken=selected?.token;
  useEffect(()=>{
    if(mobileDetail&&currentToken&&window.matchMedia("(max-width: 999px)").matches){
      detail.current?.focus();
      detail.current?.scrollIntoView({block:"start"});
    }
  },[mobileDetail,currentToken]);
  useEffect(()=>{
    const target=loadTarget.current;
    if(!hasMore||!target||typeof IntersectionObserver==="undefined")return;
    // La intersección con el viewport respeta también el scroll interno de escritorio.
    const observer=new IntersectionObserver(entries=>{
      if(!entries.some(entry=>entry.isIntersecting&&entry.intersectionRatio>0)||target.contains(document.activeElement))return;
      observer.unobserve(target);
      setVisibleCount(count=>Math.min(count+DISPATCH_BATCH_SIZE,total));
    },{threshold:0.01});
    observer.observe(target);
    return ()=>observer.disconnect();
  },[hasMore,total,visibleCount,viewKey,mobileDetail]);
  const resetWindow=()=>{setVisibleCount(DISPATCH_BATCH_SIZE);setSelectedToken("");setMobileDetail(false);stream.current?.scrollTo({top:0})};
  const reset=()=>{setFilter("all");setSearch("");setAttentionOnly(false);setSelectedCourier("");resetWindow()};
  const back=()=>{setMobileDetail(false);requestAnimationFrame(()=>{list.current?.focus();list.current?.scrollIntoView({block:"start"})})};
  const loadMore=()=>{
    const firstNew=visible.items.length;
    setVisibleCount(count=>Math.min(count+DISPATCH_BATCH_SIZE,total));
    requestAnimationFrame(()=>{
      const next=stream.current?.querySelectorAll<HTMLButtonElement>(".dispatch-order")[firstNew];
      next?.focus({preventScroll:true});next?.scrollIntoView({block:"nearest"});
    });
  };

  return <div className={`dispatch-workspace${mobileDetail&&selected?" has-detail":""}`}>
    <div className="dispatch-navigation">
      <Tabs items={tabs} value={filter} onChange={value=>{setFilter(value);setAttentionOnly(false);setSelectedCourier("");resetWindow()}} label="Vista de reparto"/>
    </div>
    <div className="dispatch-columns">
      <div className="dispatch-overview">
        <section className="panel dispatch-board" aria-labelledby="dispatch-board-title" ref={list} tabIndex={-1}>
          <header>
            <div><h2 id="dispatch-board-title">{filter==="delivered"?"Entregados recientes":"Pedidos de reparto"}</h2><p>{filter==="delivered"?"Consulta los repartos completados recientemente.":"Acepta pedidos y asigna al responsable de cada entrega."}</p></div>
            <span className="partner-panel-count" aria-label={`${filtered.length} pedidos`}>{filtered.length}</span>
          </header>
          <div className="dispatch-filters">
            <SearchField label="Buscar pedidos o motorizados" placeholder="Buscar pedido, cliente o motorizado" value={search} onChange={event=>{setSearch(event.target.value);resetWindow()}}/>
            {selectedCourier&&<Button kind="ghost" appearance="strong" icon="close" aria-label={`Quitar filtro de ${selectedCourier}`} onClick={()=>{setSelectedCourier("");resetWindow()}}>{selectedCourier}</Button>}
            {(attention>0||attentionOnly)&&filter!=="delivered"&&<Button kind="secondary" appearance="strong" icon="alert" aria-pressed={attentionOnly} onClick={()=>{setAttentionOnly(!attentionOnly);resetWindow()}}>Con alertas ({attention})</Button>}
          </div>
          {newOrders>0&&filter==="all"&&!search&&!attentionOnly&&<p className="dispatch-incoming"><Icon name="bell" size={16}/>{newOrders===1?"Tienes un pedido nuevo por aceptar.":`Tienes ${newOrders} pedidos nuevos por aceptar.`}</p>}
          {filtered.length>0?<div className="dispatch-order-stream" ref={stream}>
            <ul className="dispatch-orders" aria-label="Pedidos de reparto">{visible.items.map(order=>{
            const meta=stageMeta[order.stage];
            const alert=needsAttention(order);
            return <li key={order.token}><button type="button" className={`dispatch-order${selected?.token===order.token?" is-selected":""}`} aria-pressed={selected?.token===order.token} aria-label={`Ver pedido ${order.orderCode}, ${order.customerName||"Sin nombre"}${needsAssignment(order)?", sin motorizado":""}`} onClick={()=>{setSelectedToken(order.token);setMobileDetail(true)}}>
              <span className="dispatch-order-identity"><b>{order.orderCode}</b><Status tone={meta.tone} icon={meta.icon}>{meta.label}</Status></span>
              <span className="dispatch-order-customer">{order.customerName||"Sin nombre"}<span>{order.currencySymbol} {order.total}</span></span>
              <span className="dispatch-order-route"><Icon name="store" size={14}/><span>{order.locationName}</span></span>
              <span className="dispatch-order-route"><Icon name="mapPin" size={14}/><span>{order.address||"Dirección por confirmar"}</span></span>
              <span className="dispatch-order-footer"><span><Icon name={order.courierName?"motorbike":"user"} size={16}/>{order.courierName|| (order.stage==="offer"?"Acepta para asignar":"Sin motorizado")}</span>
                {order.stage==="offer"?<span className="dispatch-order-deadline"><Icon name="clock" size={14}/><OfferCountdown order={order} receivedAt={receivedAt}/></span>:<Icon name="chevron" size={16}/>}</span>
              {alert&&<span className="dispatch-order-alert"><Icon name="alert" size={14}/>{order.issueReportedAt?"Reportó un problema en la entrega":`Recogida pendiente · ${waitLabel(order.pickupWaitSeconds)}`}</span>}
            </button></li>;
          })}</ul>
            {total>DISPATCH_BATCH_SIZE&&<div className="dispatch-load-more" ref={loadTarget}>
              <p role="status">Mostrando {visible.items.length} de {total} pedidos{!hasMore&&" · Llegaste al final"}</p>
              {hasMore&&<Button kind="secondary" appearance="strong" icon="plus" onClick={loadMore}>Cargar más pedidos</Button>}
            </div>}
          </div>:search||filter!=="all"||attentionOnly||selectedCourier?<EmptyState icon="search" title="Sin coincidencias" text="Prueba otra búsqueda o vuelve a todos los pedidos." action={<Button kind="secondary" appearance="strong" onClick={reset}>Ver todos los pedidos</Button>}/>:<EmptyState icon="motorbike" title="Sin pedidos por ahora" text="Los pedidos nuevos aparecerán aquí. Activa los avisos para recibir una alerta." className="partner-panel-empty"/>}
        </section>
        <section className="panel dispatch-couriers" aria-labelledby="dispatch-couriers-title">
          <header><h2 id="dispatch-couriers-title">Motorizados en curso</h2><Icon name="motorbike" size={20}/></header>
          {couriers.length>0?<ul>{couriers.map(courier=><li key={courier.name}><button type="button" onClick={()=>{setSearch("");setSelectedCourier(courier.name);setFilter("all");setAttentionOnly(false);resetWindow();list.current?.scrollIntoView({block:"start"});list.current?.focus()}} aria-label={`Ver repartos de ${courier.name}`}>
            <span className="dispatch-courier-avatar" aria-hidden="true">{courier.name.split(/\s+/).slice(0,2).map(part=>part[0]).join("")}</span>
            <span><b>{courier.name}</b><small>{[courier.collecting>0?`${courier.collecting} por recoger`:"",courier.onRoute>0?`${courier.onRoute} en camino`:""].filter(Boolean).join(" · ")}</small></span><Icon name="chevron" size={16}/>
          </button></li>)}</ul>:<p className="dispatch-courier-empty">Al asignar un pedido, verás aquí al motorizado y sus repartos en curso.</p>}
        </section>
      </div>
      <div className="dispatch-detail" ref={detail} tabIndex={-1} onFocusCapture={()=>{if(selected&&selectedToken!==selected.token)setSelectedToken(selected.token)}} aria-label={selected?`Gestión del pedido ${selected.orderCode}`:"Asignación de motorizados"}>
        {selected?<><div className="dispatch-back"><Button kind="ghost" appearance="strong" icon="chevronLeft" onClick={back}>Volver a pedidos</Button></div><section className="panel dispatch-detail-panel">{renderOrder(selected)}</section></>:<section className="panel dispatch-guide">
          <IconTile name="userCheck" tone="primary"/>
          <h2>{active.length>0?"Selecciona un pedido":"Todo listo para el próximo reparto"}</h2><p>{active.length>0?"Ajusta los filtros para ver y gestionar tus repartos.":"Gestiona la asignación desde un solo lugar."}</p>
          <dl><div><dt><Icon name="check" size={18}/>Acepta un pedido</dt><dd>Revisa el local, la dirección y los importes.</dd></div><div><dt><Icon name="userCheck" size={18}/>Asigna un motorizado</dt><dd>Registra su nombre y genera su enlace de reparto.</dd></div><div><dt><Icon name="link" size={18}/>Comparte su enlace</dt><dd>El motorizado registra la recogida y la entrega desde su celular.</dd></div></dl>
        </section>}
      </div>
    </div>
  </div>;
}
