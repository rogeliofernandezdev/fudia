"use client";
import {useEffect,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {useSession} from "@/providers";
import {PdfViewer} from "@/shared/pdf-viewer";
import {loadRestaurantReportLogo} from "@/shared/api/restaurant-branding";
import {getSaleDetail} from "../infrastructure/sales-api";
import type {SaleDetail} from "../domain/types";

export function PaymentTicketDialog({orderId,paymentId,close,mode="payment"}:{orderId:string;paymentId?:string;mode?:"payment"|"prebill";close:()=>void}){
 const{organization,location}=useSession();
 const detail=useQuery({queryKey:["payment-ticket",organization?.id,location?.id,orderId],queryFn:()=>getSaleDetail(orderId),staleTime:0,refetchOnMount:"always",refetchOnWindowFocus:false});
 const[prepared,setPrepared]=useState<{document:Blob;filename:string;source:SaleDetail;paymentId?:string;mode?:"payment"|"prebill"}|null>(null);
 const[error,setError]=useState<{message:string;source:SaleDetail;paymentId?:string;mode?:"payment"|"prebill"}|null>(null),[attempt,setAttempt]=useState(0);
 useEffect(()=>{
  if(!detail.data||detail.isFetching)return;
  let cancelled=false;const data=detail.data;
  import("../infrastructure/payment-ticket-pdf").then(async builder=>{
   const logo=await loadRestaurantReportLogo();
   const layout=builder.layoutPaymentTicket(data,paymentId,mode,Boolean(logo));
   const document=builder.buildPaymentTicketPdf(layout,logo);
   if(!cancelled){setPrepared({document,filename:`${mode=== "prebill"?"precuenta":"ticket"}-${data.order.code.replace(/[^a-zA-Z0-9_-]/g,"-")}-80mm.pdf`,source:data,paymentId,mode});setError(null);}
  }).catch(cause=>{if(!cancelled)setError({message:cause instanceof Error?cause.message:"No pudimos preparar el ticket.",source:data,paymentId,mode});});
  return()=>{cancelled=true;};
 },[detail.data,detail.isFetching,paymentId,mode,attempt]);
 const current=prepared?.source===detail.data&&prepared?.paymentId===paymentId&&(prepared?.mode??"payment")===mode?prepared:null;
 const failure=detail.isError?detail.error.message:error?.source===detail.data&&error?.paymentId===paymentId&&(error?.mode??"payment")===mode?error?.message:null;
 return <PdfViewer title={mode==="prebill"?"Precuenta":"Ticket de pago"} subtitle="Ticket · 80 mm" size="sm" document={current?.document??null} filename={current?.filename??"ticket.pdf"} onClose={close}
  loading={detail.isPending||detail.isFetching||(!current&&!failure)} error={failure}
  onRetry={()=>{setError(null);setPrepared(null);if(detail.isError)void detail.refetch();else setAttempt(value=>value+1);}}/>;
}
