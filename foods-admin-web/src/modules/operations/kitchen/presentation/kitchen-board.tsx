"use client";
import "./kitchen.css";
import {useEffect,useMemo,useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,Icon,IconName,PageHeader,Select} from "@/design-system";
import {useFeedback,useSession} from "@/providers";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import type {KitchenStatus,KitchenTicket} from "../domain/types";
import {listKitchenTickets,updateKitchenTicketStatus} from "../infrastructure/kitchen-api";

const lanes:Array<{status:KitchenStatus;label:string;shortLabel:string;icon:IconName}>= [
  {status:"confirmado",label:"Por preparar",shortLabel:"Pendientes",icon:"clock"},
  {status:"preparando",label:"En preparación",shortLabel:"Preparando",icon:"chefHat"},
  {status:"listo",label:"Listos para entregar",shortLabel:"Listos",icon:"check"},
];

const channelIcons:Record<string,IconName>={
  salon:"utensils",
  mostrador:"store",
  recojo:"box",
  delivery:"truck",
  whatsapp:"share",
};

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

function progress(ticket:KitchenTicket,now:number){
  if(!ticket.targetMinutes||ticket.status==="listo")return null;
  return Math.min(100,Math.max(0,Math.round(elapsedMinutes(ticket,now)/ticket.targetMinutes*100)));
}

export function KitchenBoard(){
  const qc=useQueryClient();
  const{notify}=useFeedback();
  const{can,location}=useSession();
  const canManage=can("kitchen.manage");
  const[channel,setChannel]=useState("");
  const[mobileLane,setMobileLane]=useState<KitchenStatus>("confirmado");
  const[clock,setClock]=useState(()=>Date.now());

  useEffect(()=>{
    const timer=window.setInterval(()=>setClock(Date.now()),15000);
    return()=>window.clearInterval(timer);
  },[]);

  const tickets=useQuery({
    queryKey:["kitchen-tickets",channel],
    queryFn:()=>listKitchenTickets(channel),
    refetchInterval:10000,
    refetchIntervalInBackground:true,
  });

  const advance=useMutation({
    mutationFn:({ticket,status}:{ticket:KitchenTicket;status:Extract<KitchenStatus,"preparando"|"listo">})=>updateKitchenTicketStatus(ticket.id,status),
    onSuccess:(_,variables)=>{
      void qc.invalidateQueries({queryKey:["kitchen-tickets"]});
      void qc.invalidateQueries({queryKey:["orders"]});
      void qc.invalidateQueries({queryKey:["order",variables.ticket.id]});
      notify({
        tone:"success",
        title:variables.status==="preparando"?"Preparación iniciada":"Comanda lista",
        message:variables.status==="preparando"?`${variables.ticket.code} pasó a preparación.`:`${variables.ticket.code} está lista para entregar.`,
      });
    },
    onError:error=>notify({tone:"danger",title:"No se pudo actualizar la comanda",message:error.message}),
  });

  const serverBase=tickets.data?parseIsoDate(tickets.data.serverTime)?.getTime():null;
  const now=serverBase&&tickets.dataUpdatedAt?serverBase+Math.max(0,clock-tickets.dataUpdatedAt):clock;
  const items=tickets.data?.items??[];
  const counts=tickets.data?.counts??{confirmado:0,preparando:0,listo:0};
  const attention=useMemo(()=>items.filter(ticket=>{
    const tone=urgency(ticket,now);
    return tone==="warning"||tone==="late";
  }).length,[items,now]);

  const channelLabel=(value:string)=>tickets.data?.channelOptions.find(option=>option.value===value)?.label??value;
  const busyId=advance.isPending?advance.variables?.ticket.id:null;

  return <div className="kitchen-page">
    <PageHeader
      eyebrow="OPERACIÓN EN COCINA"
      title="Cocina"
      description="Prioriza comandas confirmadas, controla el tiempo de preparación y marca lo que ya está listo."
      action={<div className={`kitchen-live ${tickets.isFetching?"refreshing":""}`}><i/><span>{tickets.isFetching?"Actualizando…":"Actualiza cada 10 s"}</span></div>}
    />

    <section className="kitchen-summary" aria-label="Resumen de cocina">
      <SummaryItem icon="clock" label="POR PREPARAR" value={counts.confirmado} tone="pending"/>
      <SummaryItem icon="chefHat" label="EN PREPARACIÓN" value={counts.preparando} tone="cooking"/>
      <SummaryItem icon="check" label="LISTOS" value={counts.listo} tone="ready"/>
      <SummaryItem icon="alert" label="REQUIEREN ATENCIÓN" value={attention} tone="attention"/>
    </section>

    <section className="kitchen-toolbar">
      <div><small>COLA DE PRODUCCIÓN</small><h2>Comandas por estado</h2><p>Los tiempos objetivo se toman de la configuración de preparación de los productos.</p></div>
      <label>Canal<Select value={channel} onChange={event=>setChannel(event.target.value)} aria-label="Filtrar comandas por canal"><option value="">Todos los canales</option>{(tickets.data?.channelOptions??[]).map(option=><option value={option.value} key={option.value}>{option.label}</option>)}</Select></label>
    </section>

    <nav className="kitchen-mobile-tabs" aria-label="Estado de las comandas">
      {lanes.map(lane=><button type="button" aria-pressed={mobileLane===lane.status} className={mobileLane===lane.status?"active":""} onClick={()=>setMobileLane(lane.status)} key={lane.status}><Icon name={lane.icon} size={16}/><span>{lane.shortLabel}</span><b>{counts[lane.status]}</b></button>)}
    </nav>

    {tickets.isLoading?<KitchenLoading/>:tickets.isError?<KitchenError message={tickets.error.message} retry={()=>tickets.refetch()}/>:<div className="kitchen-board">
      {lanes.map(lane=>{
        const laneItems=items.filter(ticket=>ticket.status===lane.status);
        return <section className={`kitchen-lane kitchen-lane-${lane.status}`} data-mobile-active={mobileLane===lane.status} key={lane.status}>
          <header><span><Icon name={lane.icon} size={17}/></span><h2>{lane.label}</h2><b>{laneItems.length}</b></header>
          <div className="kitchen-lane-list">
            {laneItems.map(ticket=>{
              const tone=urgency(ticket,now);
              const elapsed=elapsedMinutes(ticket,now);
              const currentProgress=progress(ticket,now);
              const subject=ticket.tableName||ticket.customerName||channelLabel(ticket.channel)||"Pedido";
              const next=ticket.status==="confirmado"?"preparando":ticket.status==="preparando"?"listo":null;
              return <article className={`kitchen-ticket ${tone}`} key={ticket.id}>
                <div className="kitchen-ticket-head">
                  <div className="kitchen-ticket-identity"><span><Icon name={channelIcons[ticket.channel]??"receipt"} size={16}/></span><div><b>{subject}</b><small>{channelLabel(ticket.channel)} · {ticket.code}</small></div></div>
                  <div className={`kitchen-ticket-time ${tone}`}><strong>{elapsed}</strong><span>min</span></div>
                </div>

                {ticket.targetMinutes&&ticket.status!=="listo"&&<div className="kitchen-ticket-target">Objetivo {ticket.targetMinutes} min</div>}
                {currentProgress!==null&&<div className="kitchen-progress" aria-label={`Avance de tiempo ${currentProgress}%`}><i style={{width:`${currentProgress}%`}}/></div>}

                <div className="kitchen-ticket-items">
                  {ticket.items.map(item=><div className="kitchen-ticket-item" key={item.id}>
                    <b>{formatRegionalNumber(Number(item.qty),location?.country,{maximumFractionDigits:2})}×</b>
                    <div><span>{item.name}</span>{(item.selections??[]).map(selection=><small key={selection.groupId+selection.productId}><strong>{selection.groupName}:</strong> {selection.name}</small>)}{item.note&&<em>{item.note}</em>}</div>
                  </div>)}
                </div>

                {ticket.notes&&<div className="kitchen-ticket-note"><Icon name="edit" size={14}/><span>{ticket.notes}</span></div>}

                <footer>{canManage&&next?<Button kind={next==="listo"?"success":"primary"} icon={next==="listo"?"check":"chefHat"} disabled={advance.isPending} onClick={()=>advance.mutate({ticket,status:next})}>{busyId===ticket.id?"Actualizando…":next==="preparando"?"Iniciar preparación":"Marcar listo"}</Button>:ticket.status==="listo"?<span className="kitchen-ready-label"><Icon name="check" size={14}/>Esperando entrega</span>:null}</footer>
              </article>;
            })}
            {!laneItems.length&&<div className="kitchen-lane-empty"><Icon name={lane.icon} size={20}/><b>Sin comandas</b><span>{lane.status==="confirmado"?"Los pedidos confirmados aparecerán aquí.":lane.status==="preparando"?"Nada se está preparando ahora.":"No hay pedidos esperando entrega."}</span></div>}
          </div>
        </section>;
      })}
    </div>}
  </div>;
}

function SummaryItem({icon,label,value,tone}:{icon:IconName;label:string;value:number;tone:string}){return <div className={`kitchen-summary-item ${tone}`}><span><Icon name={icon} size={17}/></span><p><small>{label}</small><strong>{value}</strong></p></div>}

function KitchenLoading(){return <div className="kitchen-board kitchen-loading" aria-label="Cargando comandas">{lanes.map(lane=><section className="kitchen-lane" key={lane.status}><header><span/><h2>{lane.label}</h2><b>—</b></header><div className="kitchen-lane-list">{Array.from({length:2},(_,index)=><i className="kitchen-ticket-skeleton" key={index}/>)}</div></section>)}</div>}

function KitchenError({message,retry}:{message:string;retry:()=>void}){return <div className="kitchen-error"><span><Icon name="alert" size={24}/></span><b>No pudimos cargar la cola de cocina</b><p>{message}</p><Button kind="secondary" icon="refresh" onClick={retry}>Reintentar</Button></div>}
