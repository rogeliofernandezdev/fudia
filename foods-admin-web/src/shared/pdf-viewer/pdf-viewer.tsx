"use client";
import {useId,useLayoutEffect,useRef,useState} from "react";
import {Button,Dialog,ErrorState,ModalHeader,Skeleton} from "@/design-system";
import type {PdfViewerProps} from "./types";

type ViewerWindow=Window&{FudiaPdfViewer?:{dispose:()=>Promise<void>}};
const channel="fudia-pdf-viewer";

export function PdfViewerModal({document:pdf,filename,title,subtitle,size="lg",onClose,loading=false,error,onRetry,notice,details}:PdfViewerProps){
 const id=useId();
 const[showDetails,setShowDetails]=useState(false);
 const releaseRef=useRef<(()=>Promise<void>)|null>(null),closing=useRef(false);
 async function close(){
  if(closing.current)return;
  closing.current=true;
  let timer:number|undefined;
  try{await Promise.race([releaseRef.current?.(),new Promise(resolve=>{timer=window.setTimeout(resolve,1200);})]);}
  catch{/* Closing must remain possible even if a damaged document cannot release cleanly. */}
  finally{window.clearTimeout(timer);onClose();}
 }
 return <div className="modal-backdrop modal-overlay-in"><Dialog className={`crud-modal pdf-viewer-modal pdf-viewer-${size} modal-panel-in`} onResponseClose={onClose} aria-labelledby={`${id}-title`}>
  <ModalHeader title={title} subtitle={subtitle} titleId={`${id}-title`} icon="receipt" onClose={()=>void close()} closeLabel="Cerrar visor PDF"/>
  {(notice||details)&&<div className="pdf-viewer-context">{notice&&<div>{notice}</div>}{details&&<Button kind="ghost" icon="eye" onClick={()=>setShowDetails(value=>!value)}>{showDetails?"Ver PDF":"Ver detalle"}</Button>}</div>}
  {showDetails?<div className="pdf-viewer-details">{details}</div>:error?<div className="pdf-viewer-state"><ErrorState title="No pudimos preparar el PDF" text={error} onRetry={onRetry}/></div>
   :loading?<PdfLoading/>:!pdf?<div className="pdf-viewer-state"><ErrorState title="No hay un PDF disponible" text="Vuelve a generar el documento para previsualizarlo." onRetry={onRetry}/></div>:<PdfFrame document={pdf} filename={filename} releaseRef={releaseRef} onRetry={onRetry}/>}
 </Dialog></div>;
}

function PdfLoading(){return <div className="pdf-viewer-state" role="status" aria-label="Cargando PDF" aria-busy="true"><Skeleton width="65%"/><Skeleton width="85%"/><Skeleton width="75%"/><Skeleton width="100%" height="var(--size-220)"/></div>;}

function PdfFrame({document:pdf,filename,releaseRef,onRetry}:{document:Blob;filename:string;releaseRef:React.RefObject<(()=>Promise<void>)|null>;onRetry?:()=>void}){
 const frame=useRef<HTMLIFrameElement>(null),connect=useRef<(()=>void)|null>(null);
 const[state,setState]=useState<"loading"|"opening"|"ready"|"error">("loading"),[error,setError]=useState("");
 const[attempt,setAttempt]=useState(0);
 useLayoutEffect(()=>{
  const element=frame.current;
  if(!element)return;
  let cancelled=false,sent=false;
  const session=crypto.randomUUID();
  setState("loading");setError("");
  const timeout=window.setTimeout(()=>{if(!cancelled){setError("El visor no respondió. Reintenta cargar el documento.");setState("error");}},30000);
  const dispose=async()=>{
   await (element.contentWindow as ViewerWindow|null)?.FudiaPdfViewer?.dispose();
  };
  releaseRef.current=dispose;
  const send=(type:string,extra:Record<string,unknown>={},transfer:Transferable[]=[])=>element.contentWindow?.postMessage({channel,session,type,...extra},window.location.origin,transfer);
  const listener=async(event:MessageEvent)=>{
   if(cancelled||event.origin!==window.location.origin||event.source!==element.contentWindow||event.data?.channel!==channel||event.data?.session!==session)return;
   if(event.data.type==="connected"){
    if(sent)return;
    sent=true;window.clearTimeout(timeout);setState("opening");
    try{const data=await pdf.arrayBuffer();if(!cancelled)send("open",{data,filename},[data]);}
    catch{if(!cancelled){window.clearTimeout(timeout);setError("No pudimos leer el documento. Vuelve a generarlo.");setState("error");}}
   }else if(event.data.type==="loaded"){
    window.clearTimeout(timeout);setState("ready");
   }else if(event.data.type==="error"){
    window.clearTimeout(timeout);setError("No pudimos abrir el PDF. Reintenta o vuelve a generarlo.");setState("error");
   }
  };
  window.addEventListener("message",listener);
  connect.current=()=>send("connect");
  // Start exactly one navigation after the listener exists. A distinct URL also
  // prevents a cancelled StrictMode/source navigation from reusing about:blank.
  // The nonce identifies only this viewer instance; no document URL or credential.
  element.src=`/pdfjs/web/viewer.html?file=&session=${session}`;
  return()=>{
   cancelled=true;window.clearTimeout(timeout);window.removeEventListener("message",listener);
   connect.current=null;releaseRef.current=null;void dispose().catch(()=>{});
  };
 },[pdf,filename,attempt,releaseRef]);
 return <div className="pdf-viewer-frame" aria-busy={state==="loading"||state==="opening"}>
  <iframe key={attempt} ref={frame} title={`PDF · ${filename}`} onLoad={()=>connect.current?.()} allow="fullscreen" allowFullScreen sandbox="allow-scripts allow-same-origin allow-modals allow-downloads allow-popups"/>
  {state==="loading"&&<div className="pdf-viewer-overlay"><PdfLoading/></div>}
  {state==="error"&&<div className="pdf-viewer-overlay"><ErrorState title="No pudimos mostrar el PDF" text={error} onRetry={()=>onRetry?onRetry():setAttempt(value=>value+1)}/></div>}
 </div>;
}
