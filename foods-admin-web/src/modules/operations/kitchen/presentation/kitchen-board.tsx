"use client";
import "../../styles/salon-comanda.css";
import "./kitchen.css";
import {useEffect,useState} from "react";
import {keepPreviousData,useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ErrorState,Icon,IconName,PageHeader,Skeleton,Tabs} from "@/design-system";
import {useFeedback,useSession} from "@/providers";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import type {KitchenStatus,KitchenTicket} from "../domain/types";
import {listKitchenTickets,updateKitchenTicketStatus} from "../infrastructure/kitchen-api";

const lanes:Array<{status:KitchenStatus;label:string;shortLabel:string;description:string;icon:IconName}>= [
  {status:"confirmado",label:"POR PREPARAR",shortLabel:"Pendientes",description:"Pedidos nuevos",icon:"clock"},
  {status:"preparando",label:"EN PREPARACIÓN",shortLabel:"Preparando",description:"Trabajo activo",icon:"cookingPot"},
  {status:"listo",label:"POR ENTREGAR",shortLabel:"Por entregar",description:"Esperando entrega",icon:"check"},
];

/** Icono por área de preparación; las áreas y sus nombres llegan del API. */
const destinationIcons:Record<string,IconName>={kitchen:"chefHat",bar:"drink"};
/** Icono por origen del pedido (canal), igual que en Pedidos y Salón. */
const channelIcons:Record<string,IconName>={salon:"utensils",mostrador:"box",recojo:"box",delivery:"truck",whatsapp:"share"};

function parseIsoDate(value:string){
  if(!value)return null;
  const normalized=value.replace(/([+-]\d{2})$/,"$1:00");
  const parsed=new Date(normalized);
  if(!Number.isNaN(parsed.getTime()))return parsed;
  const fallback=new Date(value);
  return Number.isNaN(fallback.getTime())?null:fallback;
}

function elapsedMinutes(ticket:KitchenTicket,now:number){
  const date=parseIsoDate(ticket.updatedAt);
  if(!date)return 0;
  return Math.max(0,Math.floor((now-date.getTime())/60000));
}

function urgency(ticket:KitchenTicket,now:number){
  if(ticket.status==="listo")return "ready";
  if(!ticket.targetMinutes)return "neutral";
  const ratio=elapsedMinutes(ticket,now)/ticket.targetMinutes;
  if(ratio>=1)return "late";
  if(ratio>=.75)return "warning";
  return "fresh";
}

function formatElapsed(minutes:number){
  if(minutes<1)return "Ahora";
  if(minutes<60)return `${minutes} min`;
  const hours=Math.floor(minutes/60);
  const remainingMinutes=minutes%60;
  if(hours<24)return remainingMinutes?`${hours} h ${remainingMinutes} min`:`${hours} h`;
  const days=Math.floor(hours/24);
  return days===1?"1 día":`${days} días`;
}

function urgencyCopy(tone:ReturnType<typeof urgency>):{label:string|null;icon:IconName}{
  if(tone==="late")return {label:"Con demora",icon:"alert"};
  if(tone==="warning")return {label:"Por vencer",icon:"alert"};
  if(tone==="ready")return {label:"Por entregar",icon:"check"};
  if(tone==="fresh")return {label:"A tiempo",icon:"clock"};
  return {label:null,icon:"clock"};
}

function priority(ticket:KitchenTicket,now:number){
  const elapsed=elapsedMinutes(ticket,now);
  if(ticket.status==="listo")return elapsed;
  if(!ticket.targetMinutes)return elapsed;
  return elapsed/ticket.targetMinutes*10000+elapsed;
}

export function KitchenBoard(){
  const qc=useQueryClient();
  const{notify}=useFeedback();
  const{can,location}=useSession();
  const[destination,setDestination]=useState(()=>can("kitchen.manage")?"kitchen":can("bar.manage")?"bar":"kitchen");
  const canManage=can(destination==="bar"?"bar.manage":"kitchen.manage");
  const[channel,setChannel]=useState("");
  const[mobileLane,setMobileLane]=useState<KitchenStatus>("confirmado");
  const[clock,setClock]=useState(()=>Date.now());

  useEffect(()=>{
    const timer=window.setInterval(()=>setClock(Date.now()),15000);
    return()=>window.clearInterval(timer);
  },[]);

  const tickets=useQuery({
    queryKey:["kitchen-tickets",channel,destination],
    queryFn:()=>listKitchenTickets(channel,destination),
    refetchInterval:10000,
    refetchIntervalInBackground:true,
    placeholderData:keepPreviousData,
  });

  const advance=useMutation({
    mutationFn:({ticket,status}:{ticket:KitchenTicket;status:Extract<KitchenStatus,"preparando"|"listo">})=>updateKitchenTicketStatus(ticket.id,status),
    onSuccess:(_,variables)=>{
      void qc.invalidateQueries({queryKey:["kitchen-tickets"]});
      for(const key of ["orders","salon-floor","pos-orders","pos-order","dashboard"])void qc.invalidateQueries({queryKey:[key]});
      void qc.invalidateQueries({queryKey:["order",variables.ticket.orderId]});
    },
    onError:error=>notify({tone:"danger",title:"No se pudo actualizar la comanda",message:error.message}),
  });

  const serverBase=tickets.data?parseIsoDate(tickets.data.serverTime)?.getTime():null;
  const now=serverBase&&tickets.dataUpdatedAt?serverBase+Math.max(0,clock-tickets.dataUpdatedAt):clock;
  const items=tickets.data?.items??[];
  const counts=tickets.data?.counts??{confirmado:0,preparando:0,listo:0};
  const attention=items.filter(ticket=>{
    const tone=urgency(ticket,now);
    return tone==="warning"||tone==="late";
  }).length;

  const channelLabel=(value:string)=>tickets.data?.channelOptions.find(option=>option.value===value)?.label??value;
  const busyId=advance.isPending?advance.variables?.ticket.id:null;

  return <div className="kitchen-page">
    <PageHeader
      title={destination==="bar"?"Barra":"Cocina"}
      description="Prepara y libera pedidos del local con una cola clara por estado."
      action={<div className="kitchen-header-tools">
        {attention>0&&<span className="kitchen-attention"><Icon name="alert" size={14}/>{attention} con demora</span>}
        {/* Estación: qué pantalla se opera (Cocina o Barra). No es un filtro de pedidos. */}
        {(tickets.data?.destinationOptions?.length??0)>1&&<div className="kitchen-station">
          <span className="kitchen-filter-label">Estación</span>
          <Tabs className="kitchen-area-tabs" label="Estación de preparación" value={destination} onChange={setDestination} items={(tickets.data?.destinationOptions??[]).map(option=>({value:option.value,label:option.label,icon:destinationIcons[option.value]??"kitchen"}))}/>
        </div>}
      </div>}
    />

    {/* Filtro por origen del pedido (canal), con icono por canal; afecta solo la cola de esta estación. */}
    <section className="kitchen-filters" aria-label="Filtrar por origen del pedido">
      <span className="kitchen-filter-label">Origen del pedido</span>
      {tickets.isPending?<Skeleton width="min(var(--size-320),100%)" height="var(--size-44)" radius="var(--radius-pill)"/>
      :<div className="cmd-chips" role="group" aria-label="Origen del pedido">
        {[{value:"",label:"Todos"},...(tickets.data?.channelOptions??[])].map(option=><button type="button" key={option.value} className="cmd-chip" aria-pressed={channel===option.value} onClick={()=>setChannel(option.value)}>{channel===option.value?<span className="cmd-chip-check" aria-hidden="true"><Icon name="check" size={14}/></span>:option.value&&<Icon name={channelIcons[option.value]??"receipt"} size={16}/>}{option.label}</button>)}
      </div>}
    </section>

    <Tabs className="kitchen-mobile-tabs" label="Estado de las comandas" value={mobileLane} onChange={setMobileLane} items={lanes.map(lane=>({value:lane.status,label:lane.shortLabel,icon:lane.icon,count:counts[lane.status]}))}/>

    {tickets.isPending?<KitchenLoading/>:tickets.isError?<ErrorState className="kitchen-state" title="No pudimos cargar la cola de cocina" text={tickets.error.message} onRetry={()=>void tickets.refetch()}/>:<div className="kitchen-board">
      {lanes.map(lane=>{
        const laneItems=items
          .filter(ticket=>ticket.status===lane.status)
          .sort((left,right)=>priority(right,now)-priority(left,now));
        return <section className="kitchen-lane" data-status={lane.status} data-mobile-active={mobileLane===lane.status} key={lane.status}>
          <header><span className="kitchen-lane-icon"><Icon name={lane.icon} size={16}/></span><span className="kitchen-lane-copy"><h2>{lane.label}</h2><small>{lane.description}</small></span><b>{laneItems.length}</b></header>
          <div className="kitchen-lane-list">
            {laneItems.map(ticket=>{
              const tone=urgency(ticket,now);
              const elapsed=elapsedMinutes(ticket,now);
              const elapsedText=formatElapsed(elapsed);
              const urgencyMeta=urgencyCopy(tone);
              const subject=ticket.tableName?.toUpperCase()||ticket.customerName||channelLabel(ticket.channel)||"Pedido";
              const next=ticket.status==="confirmado"?"preparando":ticket.status==="preparando"?"listo":null;
              return <article className="kitchen-ticket" key={ticket.id}>
                <div className="kitchen-ticket-head">
                  <div className="kitchen-ticket-identity">
                    <b>{subject}</b>
                    <small>{ticket.code}{ticket.targetMinutes&&ticket.status!=="listo"?` · Objetivo ${ticket.targetMinutes} min`:""}</small>
                  </div>
                  <span className={`kitchen-ticket-time ${tone}`}><Icon name={urgencyMeta.icon} size={14}/><strong>{elapsedText}</strong>{urgencyMeta.label&&<span>· {urgencyMeta.label}</span>}</span>
                </div>

                <div className="kitchen-ticket-items">
                  {ticket.items.map(item=><div className="kitchen-ticket-item" key={item.id}>
                    <b>{formatRegionalNumber(Number(item.qty),location?.country,{maximumFractionDigits:2})}×</b>
                    <div>
                      <span>{item.name}</span>
                      {(item.selections??[]).map(selection=><small key={selection.groupId+selection.productId}>{selection.groupName}: {selection.name}</small>)}
                      {item.note&&<em><Icon name="edit" size={14}/>{item.note}</em>}
                    </div>
                  </div>)}
                </div>

                {ticket.notes&&<p className="kitchen-ticket-note"><Icon name="edit" size={14}/><span><b>Nota del pedido:</b> {ticket.notes}</span></p>}

                {canManage&&next?<Button className="kitchen-ticket-action" kind="primary" icon={next==="listo"?"check":"cookingPot"} disabled={advance.isPending} onClick={()=>advance.mutate({ticket,status:next})} loading={busyId===ticket.id}>{busyId===ticket.id?"Actualizando…":next==="preparando"?"Iniciar":"Marcar listo"}</Button>:ticket.status==="listo"?<span className="kitchen-ready-label"><Icon name="check" size={14}/>Listo para entregar</span>:null}
              </article>;
            })}
            {!laneItems.length&&<p className="kitchen-lane-empty">{lane.status==="confirmado"?"Sin pedidos por preparar.":lane.status==="preparando"?"Nada en preparación.":"Nada esperando entrega."}</p>}
          </div>
        </section>;
      })}
    </div>}
  </div>;
}

function KitchenLoading(){return <div className="kitchen-board kitchen-loading" aria-label="Cargando comandas" aria-busy="true">{lanes.map(lane=><section className="kitchen-lane" data-status={lane.status} key={lane.status}><header><span className="kitchen-lane-icon"><Icon name={lane.icon} size={16}/></span><span className="kitchen-lane-copy"><h2>{lane.label}</h2><small>{lane.description}</small></span><b>—</b></header><div className="kitchen-lane-list">{Array.from({length:2},(_,index)=><article className="kitchen-ticket kitchen-ticket-skeleton" key={index}><div className="kitchen-skeleton-head"><span><i/><small/></span><i/></div><div className="kitchen-skeleton-items"><i/><i/><i/></div><i className="kitchen-skeleton-action"/></article>)}</div></section>)}</div>}
