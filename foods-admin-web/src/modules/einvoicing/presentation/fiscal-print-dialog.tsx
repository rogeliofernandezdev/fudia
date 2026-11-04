"use client";
import {useEffect,useMemo,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {Button,Spinner} from "@/design-system";
import {useMoney,useSession} from "@/providers";
import {loadRestaurantReportLogo} from "@/shared/api/restaurant-branding";
import {formatRegionalCalendarDate,formatRegionalNumber} from "@/shared/i18n/regional-format";
import {PdfViewer} from "@/shared/pdf-viewer";
import {AUTHORITY_WAIT_MS,fiscalPrintPhase,type FiscalPrintFormat} from "../domain/fiscal-printout";
import {getFiscalDocument} from "../infrastructure/einvoicing-api";
import {FiscalStatus} from "./fiscal-status";
import "./einvoicing.css";

/**
 * Printed representation of an electronic document. From checkout (`wait`) it
 * first waits a few seconds for the tax authority, then prints the signed
 * document even if the answer is still pending; the send keeps retrying.
 */
export function FiscalPrintDialog({documentId,close,onShowTicket,wait=false}:{documentId:string;close:()=>void;onShowTicket?:()=>void;wait?:boolean}){
 const{location}=useSession();const money=useMoney();const country=location?.country;
 const[waited,setWaited]=useState(!wait);
 useEffect(()=>{if(waited)return;const timer=window.setTimeout(()=>setWaited(true),AUTHORITY_WAIT_MS);return()=>window.clearTimeout(timer);},[waited]);
 const detail=useQuery({queryKey:["fiscal-document",documentId],queryFn:()=>getFiscalDocument(documentId),staleTime:0,refetchOnWindowFocus:false,
  refetchInterval:query=>{const data=query.state.data;if(!data)return false;const phase=fiscalPrintPhase(data,waited);return phase==="ready"||phase==="rejected"?false:waited?4000:1000;}});
 const data=detail.data,phase=data?fiscalPrintPhase(data,waited):null;
 const printable=phase==="ready"||phase==="pending";
 const key=printable&&data?`${data.id}:${data.digestValue}`:"";
 const format=useMemo<FiscalPrintFormat>(()=>({
  money:value=>money(Number(value)),
  date:value=>formatRegionalCalendarDate(value,country,{dateStyle:"short"}),
  quantity:value=>formatRegionalNumber(Number(value),country,{maximumFractionDigits:3}),
  percent:rate=>`${formatRegionalNumber(Math.round(Number(rate)*1_000_000)/10_000,country,{maximumFractionDigits:4})} %`,
 }),[money,country]);
 const[prepared,setPrepared]=useState<{key:string;document:Blob}|null>(null);
 const[failure,setFailure]=useState<{key:string;message:string}|null>(null),[attempt,setAttempt]=useState(0);
 useEffect(()=>{
  // The printout never changes once signed: polls that only update the status do not rebuild the PDF.
  if(!key||!data||prepared?.key===key)return;
  let cancelled=false;
  Promise.all([import("../infrastructure/fiscal-printout-pdf"),loadRestaurantReportLogo()])
   .then(([builder,logo])=>builder.buildFiscalPrintoutPdf(data,format,logo))
   .then(document=>{if(!cancelled){setPrepared({key,document});setFailure(null);}})
   .catch(cause=>{if(!cancelled)setFailure({key,message:cause instanceof Error?cause.message:"No pudimos preparar el comprobante."});});
  return()=>{cancelled=true;};
 },[key,data,format,prepared,attempt]);

 const title=data?`${data.label} ${data.number}`:"Comprobante";
 const ticket=onShowTicket&&<Button kind="ghost" icon="eye" onClick={onShowTicket}>Ver ticket de venta</Button>;
 let notice=null,error:string|null=null,errorTitle:string|undefined;
 if(detail.isError){error=detail.error.message;errorTitle="No pudimos cargar el comprobante";}
 else if(!data||phase==="waiting")notice=<span className="fiscal-print-notice" role="status" aria-live="polite"><Spinner size={16}/>{wait?"Cobro registrado. Enviando el comprobante…":"Cargando el comprobante…"}{data?.printout&&<Button kind="ghost" onClick={()=>setWaited(true)}>Imprimir sin esperar</Button>}</span>;
 else if(phase==="rejected"){error=data.statusMessage||"La autoridad tributaria no aceptó el comprobante.";errorTitle="Comprobante rechazado";notice=<span className="fiscal-print-notice"><FiscalStatus status={data.status}/>El cobro quedó registrado.{ticket}</span>;}
 else if(phase==="unsigned"){error=data.statusMessage||"El comprobante aún no está firmado. Se reintentará automáticamente.";errorTitle="Comprobante aún no disponible";notice=<span className="fiscal-print-notice"><FiscalStatus status={data.status}/>{ticket}</span>;}
 else if(phase==="pending")notice=<span className="fiscal-print-notice" role="status"><FiscalStatus status={data.status}/>Firmado; la confirmación se reintenta sola.</span>;
 else if(data.status==="OBSERVED")notice=<span className="fiscal-print-notice"><FiscalStatus status={data.status}/>{data.statusMessage}</span>;
 else notice=<span className="fiscal-print-notice"><FiscalStatus status={data.status}/></span>;
 const current=prepared?.key===key?prepared:null;
 if(!error&&printable&&failure?.key===key)error=failure.message;
 return <PdfViewer title={title} subtitle="Comprobante · 80 mm" size="sm" document={current?.document??null} filename={data?`${data.number}-80mm.pdf`:"comprobante.pdf"} onClose={close}
  loading={!error&&(!data||phase==="waiting"||(printable&&!current))} error={error} errorTitle={errorTitle} notice={notice}
  onRetry={detail.isError||phase==="unsigned"?()=>void detail.refetch():printable?()=>{setFailure(null);setPrepared(null);setAttempt(value=>value+1);}:undefined}/>;
}
