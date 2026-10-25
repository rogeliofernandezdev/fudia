"use client";
import {useEffect,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {Button,ErrorState,ModalHeader,ModalFooter} from "@/design-system";
import {Dialog} from "@/design-system/dialog";
import {useSession} from "@/providers";
import {getSaleDetail} from "../infrastructure/sales-api";
import {paymentTicketRows,type TicketRow} from "../domain/payment-ticket";
import "./payment-ticket.css";

export function PaymentTicketDialog({orderId,paymentId,close}:{orderId:string;paymentId?:string;close:()=>void}){
 const{organization,location}=useSession();
 const detail=useQuery({queryKey:["payment-ticket",organization?.id,location?.id,orderId],queryFn:()=>getSaleDetail(orderId),staleTime:0,refetchOnMount:"always",refetchOnWindowFocus:false});
 const[prepared,setPrepared]=useState<{url:string;filename:string;rows:TicketRow[]}|null>(null);
 // Chrome en Android no muestra PDF dentro de un iframe: ahí la vista previa usa las mismas líneas en HTML.
 const[inlinePdf]=useState(()=>typeof navigator==="undefined"||navigator.pdfViewerEnabled!==false);
 const[error,setError]=useState<string|null>(null),[attempt,setAttempt]=useState(0);
 useEffect(()=>{
  if(!detail.data||detail.isFetching)return;
  let cancelled=false,url:string|undefined;const data=detail.data;
  import("../infrastructure/payment-ticket-pdf").then(builder=>{
   const layout=builder.layoutPaymentTicket(data,paymentId);
   const blob=builder.buildPaymentTicketPdf(layout);
   if(!cancelled){url=URL.createObjectURL(blob);setPrepared({url,filename:`ticket-${data.order.code.replace(/[^a-zA-Z0-9_-]/g,"-")}-80mm.pdf`,rows:paymentTicketRows(data,paymentId)});setError(null);}
  }).catch(cause=>{if(!cancelled)setError(cause instanceof Error?cause.message:"No pudimos preparar el ticket.");});
  return()=>{cancelled=true;if(url)URL.revokeObjectURL(url);};
 },[detail.data,detail.isFetching,paymentId,attempt]);
 const failure=detail.isError?detail.error.message:error;
 const loading=detail.isPending||detail.isFetching||(!prepared&&!failure);
 function download(){if(!prepared)return;const anchor=document.createElement("a");anchor.href=prepared.url;anchor.download=prepared.filename;anchor.click();}
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal payment-ticket-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="payment-ticket-title">
  <ModalHeader title="Ticket de pago" titleId="payment-ticket-title" icon="receipt" subtitle="Ticket · 80 mm" onClose={close} closeLabel="Cerrar ticket"/>
  <div className="payment-ticket-body">
   {failure?<ErrorState title="No pudimos preparar el ticket" text={failure} onRetry={()=>{setError(null);setPrepared(null);if(detail.isError)void detail.refetch();else setAttempt(value=>value+1);}}/>
   :loading?<div className="payment-ticket-skeleton" aria-busy="true" aria-label="Preparando ticket">{Array.from({length:8},(_,index)=><i className="remote-modal-skeleton-block" key={index}/>)}</div>
   :inlinePdf?<iframe title="Vista previa PDF del ticket de pago de 80 mm" src={prepared!.url}/>
   :<TicketPaper rows={prepared!.rows}/>}
  </div>
  <ModalFooter><Button icon="download" disabled={!prepared||loading} onClick={download}>Descargar PDF</Button></ModalFooter>
 </Dialog></div>;
}

/** Vista previa en papel de 80 mm con las mismas líneas del PDF (tamaños 8, 9, 11–13 pt). */
function TicketPaper({rows}:{rows:TicketRow[]}){
 return <div className="payment-ticket-paper" role="document" aria-label="Vista previa del ticket de pago de 80 mm">
  {rows.map((row,index)=>row.rule?<hr key={index}/>
   :<p key={index} className={[row.center?"is-center":"",row.bold?"is-bold":"",(row.size??9)>=12?"is-lg":(row.size??9)>=11?"is-md":(row.size??9)<=8?"is-sm":""].filter(Boolean).join(" ")||undefined}>
    <span>{row.text}</span>{row.right&&<span>{row.right}</span>}
   </p>)}
 </div>;
}
