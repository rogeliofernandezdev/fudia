"use client";
import {useEffect,useState} from "react";
import {Button,Icon,RemoteModalSkeleton} from "@/design-system";
import {Dialog} from "@/design-system/dialog";
import {readCssToken} from "@/design-system/theme";
import {useCashReport} from "../application/use-cash-report";
import {CashReportContent} from "./cash-report-content";
import "./cash-report.css";

export function CashReportDialog({shiftId,close}:{shiftId:string;close:()=>void}){
 const report=useCashReport(shiftId);
 const [pdf,setPdf]=useState<{url:string;filename:string}|null>(null);
 const [error,setError]=useState<string|null>(null);
 const [attempt,setAttempt]=useState(0);
 const [detail,setDetail]=useState(false);
 useEffect(()=>{
  if(!report.data)return;
  let cancelled=false,url:string|undefined;
  const value=report.data;
  Promise.all([import("../infrastructure/cash-report-pdf")]).then(async([builder])=>{
   const logo=await builder.loadFudiaReportLogo();
   if(cancelled)return;
   const blob=builder.buildCashReportPdf(value,logo,{primary:readCssToken("--primary-600"),ink:readCssToken("--ink-950"),muted:readCssToken("--ink-500"),surface:readCssToken("--surface"),soft:readCssToken("--cloud-50"),line:readCssToken("--line"),brand:readCssToken("--brand-600")});
   url=URL.createObjectURL(blob);
   setPdf({url,filename:`cierre-${value.shift.code.replace(/[^a-zA-Z0-9_-]/g,"-")}.pdf`});
   setError(null);
  }).catch(cause=>{if(!cancelled)setError(cause instanceof Error?cause.message:"No pudimos generar la vista previa.");});
  return()=>{cancelled=true;if(url)URL.revokeObjectURL(url);};
 },[report.data,attempt]);
 if(report.isLoading)return <RemoteModalSkeleton className="cash-report-preview" label="Cargando informe de caja" rows={6} close={close}/>;
 const failure=report.isError?report.error.message:error;
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal cash-report-preview modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="cash-report-title">
  <div className="modal-accent"/><header><span className="modal-title-icon"><Icon name="receipt" size={18}/></span><div><small>INFORME DE CAJA</small><h2 id="cash-report-title">{report.data?.shift.code??"Vista previa del cierre"}</h2></div><button type="button" aria-label="Cerrar informe" onClick={close}><Icon name="close"/></button></header>
  <div className="cash-report-preview-body">
   {failure?<div className="cash-report-error" role="alert"><b>{report.isError?"No pudimos cargar el informe":report.data?.shift.status==="closed"?"El cierre está guardado, pero no pudimos generar el PDF":"No pudimos generar el PDF del turno"}</b><p>{failure}</p><Button icon="refresh" onClick={()=>{setError(null);if(report.isError)void report.refetch();else setAttempt(value=>value+1);}}>Reintentar</Button></div>
   :report.data&&<>
    {!report.data.persisted&&<p>Este turno {report.data.shift.status==="open"?"sigue abierto; sus datos pueden cambiar":"es anterior al informe persistido; el detalle se reconstruyó con los registros disponibles"}.</p>}
    {detail?<CashReportContent report={report.data}/>:pdf?<iframe key={pdf.url} src={pdf.url} title="Vista previa del PDF de cierre de caja"/>:<div role="status" aria-label="Generando PDF" aria-busy="true" className="cash-report-pdf-skeleton"><div className="remote-modal-skeleton-grid" aria-hidden="true">{Array.from({length:6},(_,index)=><div key={index} className="remote-modal-skeleton-field"><i/><b/></div>)}</div><div className="remote-modal-skeleton-section" aria-hidden="true"><span/><i/><i/></div></div>}
    <p>Si tu navegador no muestra el PDF, usa «Descargar PDF» o consulta «Ver detalle».</p>
   </>}
  </div>
  {report.data&&!report.isError&&<footer><Button kind="ghost" onClick={()=>setDetail(value=>!value)}>{detail?"Ver PDF":"Ver detalle"}</Button><Button icon="download" disabled={!pdf||Boolean(failure)} onClick={()=>{if(!pdf)return;const anchor=document.createElement("a");anchor.href=pdf.url;anchor.download=pdf.filename;anchor.click();}}>Descargar PDF</Button></footer>}
 </Dialog></div>;
}
