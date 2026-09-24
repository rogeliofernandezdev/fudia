"use client";

import {useCallback,useEffect,useState} from "react";
import {Icon} from "@/components/icon";
import {operationsFetch} from "@/lib/operations-api";

type Handoff={
 id:string;
 tableId:string;
 tableName:string;
 conversationId:string;
 customerPhone:string;
 reason:string;
 status:"pending"|"resolved";
 requestedAt:string;
 resolvedAt:string|null;
 resolvedByName:string|null;
};

type HandoffList={items:Handoff[];total:number};

function ageLabel(value:string){
 const timestamp=new Date(value).getTime();
 if(!Number.isFinite(timestamp))return "";
 const minutes=Math.max(0,Math.floor((Date.now()-timestamp)/60000));
 if(minutes<1)return "Ahora";
 if(minutes===1)return "Hace 1 min";
 if(minutes<60)return "Hace "+minutes+" min";
 const hours=Math.floor(minutes/60);
 return hours===1?"Hace 1 h":"Hace "+hours+" h";
}

export function ConciergeHandoffNotifications(){
 const[open,setOpen]=useState(false);
 const[items,setItems]=useState<Handoff[]>([]);
 const[loading,setLoading]=useState(true);
 const[error,setError]=useState("");
 const[resolving,setResolving]=useState<string|null>(null);

 const load=useCallback(async()=>{
  try{
   const data=await operationsFetch<HandoffList>("concierge-handoffs?status=pending");
   setItems(data.items);
   setError("");
  }catch(e){
   setError(e instanceof Error?e.message:"No pudimos cargar las solicitudes.");
  }finally{
   setLoading(false);
  }
 },[]);

 useEffect(()=>{
  const first=window.setTimeout(()=>void load(),0);
  const timer=window.setInterval(()=>void load(),15000);
  return()=>{
   window.clearTimeout(first);
   window.clearInterval(timer);
  };
 },[load]);

 const resolve=async(id:string)=>{
  if(resolving)return;
  setResolving(id);
  try{
   await operationsFetch<void>("concierge-handoffs/"+id+"/resolve",{method:"PATCH"});
   setItems(current=>current.filter(item=>item.id!==id));
   setError("");
  }catch(e){
   setError(e instanceof Error?e.message:"No pudimos cerrar la solicitud.");
  }finally{
   setResolving(null);
  }
 };

 return <div className="handoff-notifications">
  <button
   className="notification-button"
   type="button"
   aria-label={items.length?items.length+" solicitudes de atención de Fudia Concierge":"Solicitudes de atención"}
   aria-expanded={open}
   onClick={()=>setOpen(value=>!value)}
  >
   <Icon name="bell"/>
   {items.length>0&&<><i/><em>{items.length>99?"99+":items.length}</em></>}
  </button>
  {open&&<section className="handoff-popover" aria-label="Solicitudes de atención">
   <header>
    <div><small>FUDIA CONCIERGE</small><b>Atención solicitada</b></div>
    <button type="button" onClick={()=>void load()} aria-label="Actualizar"><Icon name="refresh" size={16}/></button>
   </header>
   {loading?<div className="handoff-state">Cargando solicitudes…</div>:
    error&&items.length===0?<div className="handoff-state error"><Icon name="wifi" size={18}/><span>{error}</span><button type="button" onClick={()=>void load()}>Reintentar</button></div>:
    items.length===0?<div className="handoff-state"><Icon name="check" size={19}/><span>No hay mesas esperando atención.</span></div>:
    <div className="handoff-list">{items.map(item=><article key={item.id}>
     <span className="handoff-table"><Icon name="tables" size={17}/></span>
     <div>
      <b>{item.tableName}</b>
      <p>{item.reason}</p>
      <small>{ageLabel(item.requestedAt)}{item.customerPhone?" · WhatsApp "+item.customerPhone:""}</small>
     </div>
     <button type="button" disabled={resolving===item.id} onClick={()=>void resolve(item.id)}>
      {resolving===item.id?"Guardando…":"Atendido"}
     </button>
    </article>)}</div>}
   {error&&items.length>0&&<p className="handoff-inline-error">{error}</p>}
  </section>}
 </div>;
}
