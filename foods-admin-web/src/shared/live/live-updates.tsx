"use client";
import {useEffect} from "react";
import {useQueryClient,type QueryClient} from "@tanstack/react-query";
import {useSession} from "@/providers/session-context";
import {matchesLiveTopics,setLiveConnected} from "./live-state";

const EVENTS_URL="/api/admin/events";
const RETRY_MIN_MS=5_000;
const RETRY_MAX_MS=60_000;

function refreshLive(client:QueryClient,topics?:ReadonlySet<string>){
  // cancelRefetch:false reutiliza una lectura en curso (por ejemplo, la que lanzó
  // la propia mutación) en vez de cancelarla y repetirla.
  void client.invalidateQueries({predicate:query=>matchesLiveTopics(query,topics)},{cancelRefetch:false});
}

/**
 * Abre el canal en vivo del local de la sesión. Cada evento `change` refresca solo
 * las consultas que declararon esos temas en `meta.live`. Al conectar o perder el
 * canal se refrescan todas las consultas en vivo: así cambian de intervalo al
 * momento y recuperan lo que pudo cambiar mientras no había canal. Si el canal no
 * está disponible, las pantallas siguen con su consulta periódica y se reintenta
 * con espera creciente.
 */
export function LiveUpdates(){
  const client=useQueryClient();
  const {organization,location}=useSession();
  const scope=organization&&location?`${organization.id}:${location.id}`:"";
  useEffect(()=>{
    if(!scope||typeof EventSource==="undefined")return;
    let source:EventSource|null=null;
    let retry:ReturnType<typeof setTimeout>|undefined;
    let wait=RETRY_MIN_MS;
    let disposed=false;
    let isConnected=false;
    const disconnect=()=>{
      if(source){source.close();source=null}
      if(retry)clearTimeout(retry);
      retry=undefined;
    };
    const setConnected=(value:boolean)=>{
      const changed=value!==isConnected;
      isConnected=value;
      setLiveConnected(value);
      if(changed)refreshLive(client);
    };
    const connect=()=>{
      if(disposed)return;
      source=new EventSource(EVENTS_URL);
      source.addEventListener("ready",()=>{wait=RETRY_MIN_MS;setConnected(true)});
      source.addEventListener("change",event=>{
        try{
          const topics=(JSON.parse((event as MessageEvent<string>).data) as {topics?:string[]}).topics??[];
          if(topics.length)refreshLive(client,new Set(topics));
        }catch{/* Un evento ilegible se ignora: la consulta de respaldo lo cubre. */}
      });
      source.onerror=()=>{
        // EventSource reconecta solo tras un corte (readyState CONNECTING); si el
        // servidor respondió un error (503 sin canal, 401) queda cerrado y se
        // reintenta aquí con espera creciente.
        if(source?.readyState===EventSource.CONNECTING){setConnected(false);return}
        disconnect();
        setConnected(false);
        if(disposed)return;
        retry=setTimeout(connect,wait);
        wait=Math.min(wait*2,RETRY_MAX_MS);
      };
    };
    connect();
    return ()=>{disposed=true;disconnect();setLiveConnected(false)};
  },[client,scope]);
  return null;
}
