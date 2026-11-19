"use client";

import {useCallback,useEffect,useRef,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {Button,Icon,ModalHeader,Skeleton,Spinner} from "@/design-system";
import {Dialog} from "@/design-system/dialog";
import type {PayPairingDraft} from "../domain/types";
import {createPayPairing,getPayPairingStatus} from "../infrastructure/pay-devices-api";

type Props={
  /** Local al que quedará ligado el teléfono (elegido en Locales). */
  locationId:string;
  locationName:string;
  onClose:()=>void;
  /** El teléfono quedó vinculado: el panel cierra el modal, avisa y refresca la lista. */
  onLinked:(deviceName:string)=>void;
};

/** Cada cuánto se consulta el estado del código mientras el QR está a la vista. */
const STATUS_POLL_MS=1500;

/**
 * «Vincular dispositivo de pagos», como WhatsApp Web: genera el código
 * temporal y lo muestra como QR; en cuanto Fudia Pay lo escanea, el QR se
 * reemplaza por «Vinculando «modelo»…»; al terminar, el modal se cierra solo
 * y el panel avisa. Si el código vence, ofrece generar otro. El QR contiene
 * solo el código de un solo uso; las credenciales viajan del backend al
 * dispositivo y nunca por esta pantalla. Se monta solo cuando se abre.
 */
export function PayPairingDialog({locationId,locationName,onClose,onLinked}:Props){
  const [draft,setDraft]=useState<PayPairingDraft|null>(null);
  const [error,setError]=useState("");
  const [requesting,setRequesting]=useState(true);
  const canvas=useRef<HTMLCanvasElement>(null);
  const linkedOnce=useRef(false);

  // Pide un código nuevo; los estados se fijan cuando responde el backend.
  const request=useCallback((isCurrent:()=>boolean)=>{
    void createPayPairing(locationId).then(result=>{if(isCurrent())setDraft(result)})
      .catch((cause:Error)=>{if(isCurrent())setError(cause.message)})
      .finally(()=>{if(isCurrent())setRequesting(false)});
  },[locationId]);
  useEffect(()=>{
    let current=true;
    request(()=>current);
    return()=>{current=false};
  },[request]);
  /** «Generar otro código»: tras un error o un código vencido. */
  const generate=()=>{
    setRequesting(true);
    setError("");
    setDraft(null);
    linkedOnce.current=false;
    request(()=>true);
  };

  // Pendiente → escaneado → vinculado (o vencido): lo decide el backend.
  const status=useQuery({
    queryKey:["pay-pairing-status",locationId,draft?.id],
    queryFn:()=>getPayPairingStatus(locationId,draft!.id),
    enabled:Boolean(draft?.id),
    refetchInterval:query=>["linked","expired"].includes(query.state.data?.status??"")?false:STATUS_POLL_MS,
    refetchIntervalInBackground:true,
    retry:false,
    gcTime:0,
  });
  const state=status.data?.status??"pending";

  useEffect(()=>{
    if(state!=="linked"||linkedOnce.current)return;
    linkedOnce.current=true;
    onLinked(status.data?.deviceName||"Fudia Pay");
  },[state,status.data?.deviceName,onLinked]);

  useEffect(()=>{
    if(!draft?.code||state!=="pending"||!canvas.current)return;
    let cancelled=false;
    void import("qrcode").then(module=>{
      if(cancelled||!canvas.current)return;
      return module.default.toCanvas(canvas.current,draft.code,{width:240,margin:4});
    }).catch(()=>{if(!cancelled)setError("No pudimos mostrar el QR. Vuelve a abrir la vinculación.")});
    return()=>{cancelled=true};
  },[draft,state]);

  const scanned=state==="scanned";
  const expired=state==="expired";
  return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={onClose} className="crud-modal pay-pairing-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="pay-pairing-title">
    <ModalHeader title="Vincular dispositivo de pagos" titleId="pay-pairing-title" onClose={onClose} closeLabel="Cerrar"/>
    <div className="pay-pairing-body">
      {error&&<>
        <p className="field-error" role="alert">{error}</p>
        <Button kind="secondary" icon="refresh" disabled={requesting} loading={requesting} onClick={()=>generate()}>{requesting?"Generando…":"Generar otro código"}</Button>
      </>}
      {!error&&!draft&&<><Skeleton className="pay-pairing-qr-placeholder"/><p className="form-help">Generando el código de vinculación…</p></>}
      {!error&&draft&&state==="pending"&&<>
        <p>Instala Fudia Pay en el Android de «{locationName}» y escanea este QR.</p>
        <canvas ref={canvas} className="pay-pairing-qr" aria-label="QR de vinculación del dispositivo de pagos" role="img"/>
        <p className="form-help">El código vence a los 10 minutos y sirve una sola vez.</p>
        <p role="status">Esperando el escaneo…</p>
        <Button kind="secondary" onClick={onClose}>Cancelar</Button>
      </>}
      {!error&&draft&&scanned&&<div className="pay-pairing-progress" role="status" aria-live="polite">
        <Spinner size={40}/>
        <p className="pay-pairing-progress-title">Vinculando{status.data?.model?` «${status.data.model}»`:" el dispositivo"}…</p>
        <p className="form-help">Mantén Fudia Pay abierto en el teléfono hasta que termine.</p>
      </div>}
      {!error&&draft&&state==="linked"&&<div className="pay-pairing-progress is-done" role="status"><Icon name="circleCheck" size={40}/><p className="pay-pairing-progress-title">Vinculado</p></div>}
      {!error&&draft&&expired&&<>
        <p className="pay-pairing-progress-title" role="status">El código venció sin usarse.</p>
        <Button icon="refresh" disabled={requesting} loading={requesting} onClick={()=>generate()}>{requesting?"Generando…":"Generar otro código"}</Button>
      </>}
    </div>
  </Dialog></div>;
}
