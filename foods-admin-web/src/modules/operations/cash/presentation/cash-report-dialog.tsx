"use client";
import {useEffect,useState} from "react";
import {readCssToken} from "@/design-system/theme";
import {PdfViewer} from "@/shared/pdf-viewer";
import {loadRestaurantReportLogo} from "@/shared/api/restaurant-branding";
import {useCashReport} from "../application/use-cash-report";
import type {CashShiftReport} from "../domain/types";
import {CashReportContent} from "./cash-report-content";
import "./cash-report.css";

export function CashReportDialog({shiftId,close}:{shiftId:string;close:()=>void}){
 const report=useCashReport(shiftId);
 const [pdf,setPdf]=useState<{document:Blob;filename:string;source:CashShiftReport}|null>(null);
 const [error,setError]=useState<{message:string;source:CashShiftReport}|null>(null);
 const [attempt,setAttempt]=useState(0);
 useEffect(()=>{
  if(!report.data)return;
  let cancelled=false;
  const value=report.data;
  import("../infrastructure/cash-report-pdf").then(async builder=>{
   const logo=await loadRestaurantReportLogo()??await builder.loadFudiaReportLogo();
   if(cancelled)return;
   const document=builder.buildCashReportPdf(value,logo,{primary:readCssToken("--primary-600"),ink:readCssToken("--ink-950"),muted:readCssToken("--ink-500"),surface:readCssToken("--surface"),soft:readCssToken("--cloud-50"),line:readCssToken("--line"),brand:readCssToken("--brand-600")});
   setPdf({document,source:value,filename:`cierre-${`${value.shift.cashRegisterName}-${value.shift.businessDate}`.normalize("NFD").replace(/[̀-ͯ]/g,"").replace(/[^a-zA-Z0-9_-]+/g,"-").toLowerCase()}.pdf`});
   setError(null);
  }).catch(cause=>{if(!cancelled)setError({message:cause instanceof Error?cause.message:"No pudimos generar la vista previa.",source:value});});
  return()=>{cancelled=true;};
 },[report.data,attempt]);
 const current=pdf?.source===report.data?pdf:null;
 const failure=report.isError?report.error.message:error?.source===report.data?error?.message:null;
 return <PdfViewer title={report.data?.shift.cashRegisterName??"Vista previa del cierre"} subtitle="Informe de caja" size="lg" document={current?.document??null} filename={current?.filename??"informe-de-caja.pdf"} onClose={close}
  loading={report.isLoading||(!current&&!failure)} error={failure}
  onRetry={()=>{setError(null);setPdf(null);if(report.isError)void report.refetch();else setAttempt(value=>value+1);}}
  notice={report.data&&!report.data.persisted?`Este turno ${report.data.shift.status==="open"?"sigue abierto; sus datos pueden cambiar":"es anterior al informe persistido; el detalle se reconstruyó con los registros disponibles"}.`:undefined}
  details={report.data&&!report.isError?<CashReportContent report={report.data}/>:undefined}/>;
}
