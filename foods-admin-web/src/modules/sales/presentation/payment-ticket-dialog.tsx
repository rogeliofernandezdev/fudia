"use client";
import {useEffect,useRef,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {Button,Icon} from "@/design-system";
import {Dialog} from "@/design-system/dialog";
import {useSession} from "@/providers";
import {getSaleDetail} from "../infrastructure/sales-api";
import "./payment-ticket.css";

export function PaymentTicketDialog({orderId,paymentId,close}:{orderId:string;paymentId?:string;close:()=>void}){
 const{organization,location}=useSession();
 const detail=useQuery({queryKey:["payment-ticket",organization?.id,location?.id,orderId],queryFn:()=>getSaleDetail(orderId),staleTime:0,refetchOnMount:"always",refetchOnWindowFocus:false});
 const[prepared,setPrepared]=useState<{html:string;url:string;filename:string}|null>(null);
 const[error,setError]=useState<string|null>(null),[attempt,setAttempt]=useState(0),[ready,setReady]=useState(false);
 const frame=useRef<HTMLIFrameElement>(null);
 useEffect(()=>{
  if(!detail.data||detail.isFetching)return;
  let cancelled=false,url:string|undefined;const data=detail.data;
  import("../infrastructure/payment-ticket-pdf").then(builder=>{
   const layout=builder.layoutPaymentTicket(data,paymentId);
   const blob=builder.buildPaymentTicketPdf(layout);
   if(!cancelled){url=URL.createObjectURL(blob);setReady(false);setPrepared({html:builder.paymentTicketPrintHtml(layout),url,filename:`ticket-${data.order.code.replace(/[^a-zA-Z0-9_-]/g,"-")}-80mm.pdf`});setError(null);}
  }).catch(cause=>{if(!cancelled)setError(cause instanceof Error?cause.message:"No pudimos preparar el ticket.");});
  return()=>{cancelled=true;if(url)URL.revokeObjectURL(url);};
 },[detail.data,detail.isFetching,paymentId,attempt]);
 const failure=detail.isError?detail.error.message:error;
 const loading=detail.isPending||detail.isFetching||(!prepared&&!failure);
 function download(){if(!prepared)return;const anchor=document.createElement("a");anchor.href=prepared.url;anchor.download=prepared.filename;anchor.click();}
 function print(){try{if(!ready||!frame.current?.contentWindow)return;frame.current.contentWindow.focus();frame.current.contentWindow.print();}catch{setError("No pudimos abrir la impresión. Descarga el PDF para imprimirlo.");}}
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal payment-ticket-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="payment-ticket-title">
  <div className="modal-accent"/><header><span className="modal-title-icon"><Icon name="receipt" size={18}/></span><div><small>TICKET · 80 MM</small><h2 id="payment-ticket-title">Ticket de pago</h2></div><button type="button" aria-label="Cerrar ticket" onClick={close}><Icon name="close"/></button></header>
  <div className="payment-ticket-body">
   <p>Imprime al 100 %, con papel de 80 mm y sin cabeceras ni pies del navegador. Este ticket no reemplaza una boleta o factura.</p>
   {failure?<div className="payment-ticket-error" role="alert"><b>No pudimos preparar el ticket</b><p>{failure}</p><Button kind="secondary" icon="refresh" onClick={()=>{setError(null);setPrepared(null);setReady(false);if(detail.isError)void detail.refetch();else setAttempt(value=>value+1);}}>Reintentar</Button></div>
   :loading?<div className="payment-ticket-skeleton" aria-busy="true" aria-label="Preparando ticket">{Array.from({length:8},(_,index)=><i className="remote-modal-skeleton-block" key={index}/>)}</div>
   :<><iframe title="Vista previa PDF del ticket de pago de 80 mm" src={prepared!.url}/><iframe className="payment-ticket-print-frame" ref={frame} title="Documento de impresión del ticket" aria-hidden="true" tabIndex={-1} srcDoc={prepared!.html} onLoad={()=>setReady(true)}/></>}
  </div>
  <footer><Button kind="secondary" icon="download" disabled={!prepared||loading} onClick={download}>Descargar PDF</Button><Button icon="receipt" disabled={!ready||loading||Boolean(failure)} onClick={print}>Imprimir ticket</Button></footer>
 </Dialog></div>;
}
