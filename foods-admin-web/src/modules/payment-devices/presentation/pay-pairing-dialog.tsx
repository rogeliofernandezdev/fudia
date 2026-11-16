"use client";

import {useEffect,useRef,useState} from "react";
import {useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ModalHeader,Skeleton} from "@/design-system";
import {Dialog} from "@/design-system/dialog";
import {createPayPairing,listPayDevices} from "../infrastructure/pay-devices-api";

type Props={onClose:()=>void};

/**
 * «Vincular dispositivo de pagos»: genera el código temporal, lo muestra como
 * QR y vigila los dispositivos del local hasta que el Android se activa. El QR
 * contiene solo el código de un solo uso; las credenciales viajan del backend
 * al dispositivo y nunca por esta pantalla. Se monta solo cuando se abre.
 */
export function PayPairingDialog({onClose}:Props){
  const qc=useQueryClient();
  const [draft,setDraft]=useState<{code:string;expiresAt:string}|null>(null);
  const [issuedAt,setIssuedAt]=useState<number>(0);
  const [error,setError]=useState("");
  const canvas=useRef<HTMLCanvasElement>(null);

  useEffect(()=>{
    let cancelled=false;
    void createPayPairing().then(result=>{
      if(cancelled)return;
      setDraft(result);
      setIssuedAt(Date.now());
    }).catch((cause:Error)=>{if(!cancelled)setError(cause.message)});
    return()=>{cancelled=true};
  },[]);

  // El Android aparece en la lista del local apenas canjea el código.
  const devices=useQuery({
    queryKey:["pay-devices","pairing-watch",issuedAt],
    queryFn:()=>listPayDevices(),
    enabled:Boolean(draft),
    refetchInterval:query=>query.state.data?.items.some(item=>Date.parse(item.createdAt)>=issuedAt-1000)?false:3000,
    retry:false,
    gcTime:0,
  });
  const linked=draft?devices.data?.items.find(item=>Date.parse(item.createdAt)>=issuedAt-1000):undefined;
  useEffect(()=>{
    if(linked)void qc.invalidateQueries({queryKey:["pay-devices"]});
  },[linked,qc]);

  useEffect(()=>{
    if(!draft?.code||linked||!canvas.current)return;
    let cancelled=false;
    void import("qrcode").then(module=>{
      if(cancelled||!canvas.current)return;
      return module.default.toCanvas(canvas.current,draft.code,{width:240,margin:4});
    }).catch(()=>{if(!cancelled)setError("No pudimos mostrar el QR. Vuelve a abrir la vinculación.")});
    return()=>{cancelled=true};
  },[draft,linked]);

  if(!draft&&!error)return null;
  return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={onClose} className="crud-modal pay-pairing-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="pay-pairing-title">
    <ModalHeader title="Vincular dispositivo de pagos" titleId="pay-pairing-title" onClose={onClose} closeLabel="Cerrar"/>
    <div className="pay-pairing-body">
      {error&&<p className="field-error" role="alert">{error}</p>}
      {!draft&&!error&&<><Skeleton className="pay-pairing-qr-placeholder"/><p className="form-help">Generando el código de vinculación…</p></>}
      {draft&&linked&&<>
        <p className="pay-pairing-done" role="status"><strong>Vinculado.</strong> El dispositivo «{linked.name}» quedó asociado a este local.</p>
        <Button kind="secondary" onClick={onClose}>Listo</Button>
      </>}
      {draft&&!linked&&<>
        <p>Instala Fudia Pay en el Android del local y escanea este QR.</p>
        <canvas ref={canvas} className="pay-pairing-qr" aria-label="QR de vinculación del dispositivo de pagos" role="img"/>
        <p className="form-help">El código vence a los 10 minutos y sirve una sola vez. Genera otro si vence.</p>
        <p role="status">Esperando la vinculación del dispositivo…</p>
        <Button kind="secondary" onClick={onClose}>Cancelar</Button>
      </>}
    </div>
  </Dialog></div>;
}
