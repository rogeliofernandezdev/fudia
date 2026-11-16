"use client";

import {useCallback,useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,EmptyState,ErrorState,IconButton,Input,Status,TableSkeleton,Tag} from "@/design-system";
import {useFeedback} from "@/providers";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import type {PayDevice} from "../domain/types";
import {listPayDevices,renamePayDevice,revokePayDevice} from "../infrastructure/pay-devices-api";
import {PayPairingDialog} from "./pay-pairing-dialog";
import "./payment-devices.css";

const platformLabel=(platform:string)=>platform==="IOS"?"iPhone":"Android";

type Props={dateTime:(value:string)=>string};

/** Dispositivos de pago vinculados al local: alta por QR, renombrar y revocar. */
export function PayDevicesPanel({dateTime}:Props){
  const qc=useQueryClient();
  const {notify}=useFeedback();
  const [pairing,setPairing]=useState(false);
  const [revokeTarget,setRevokeTarget]=useState<PayDevice|null>(null);
  const [renameTarget,setRenameTarget]=useState<PayDevice|null>(null);
  const devices=useQuery({queryKey:["pay-devices"],queryFn:listPayDevices,refetchInterval:30000});
  const closePairing=useCallback(()=>setPairing(false),[]);
  // Como WhatsApp Web: al terminar la vinculación el modal se cierra solo.
  const linked=useCallback((name:string)=>{
    setPairing(false);
    void qc.invalidateQueries({queryKey:["pay-devices"]});
    notify({tone:"success",title:"Dispositivo vinculado",message:`«${name}» ya reporta los pagos de Yape y Plin de este local.`});
  },[qc,notify]);

  const revoke=useMutation({
    mutationFn:(id:string)=>revokePayDevice(id),
    onSuccess:device=>{
      setRevokeTarget(null);
      notify({tone:"success",title:"Dispositivo revocado",message:`«${device.name}» ya no puede reportar pagos hasta vincularse de nuevo.`});
      void qc.invalidateQueries({queryKey:["pay-devices"]});
    },
    onError:(error:Error)=>notify({tone:"danger",title:"No se pudo revocar",message:error.message}),
  });
  const rename=useMutation({
    mutationFn:({id,name}:{id:string;name:string})=>renamePayDevice(id,name),
    onSuccess:()=>{setRenameTarget(null);void qc.invalidateQueries({queryKey:["pay-devices"]})},
    onError:(error:Error)=>notify({tone:"danger",title:"No se pudo renombrar",message:error.message}),
  });

  const items=devices.data?.items??[];
  return <section className="pay-devices-section" aria-label="Dispositivos de pago">
    <header className="pay-devices-toolbar">
      <div><h2>Dispositivos de pago</h2><p>Cada Android del local que reporta los pagos de Yape y Plin.</p></div>
      <Button icon="plus" onClick={()=>setPairing(true)}>Vincular dispositivo</Button>
    </header>
    {devices.isLoading?<TableSkeleton columns={["Dispositivo","Plataforma","Último contacto","Estado","Acciones"]} rows={3} label="Cargando dispositivos"/>
    :devices.isError?<ErrorState title="No pudimos cargar los dispositivos" text={devices.error.message} onRetry={()=>void devices.refetch()}/>
    :!items.length?<EmptyState icon="wallet" title="Aún no hay dispositivos vinculados" text="Genera el QR de vinculación e instálalo en el Android del local."/>
    :<table className="pay-devices-table">
      <thead><tr><th>Dispositivo</th><th>Plataforma</th><th>Último contacto</th><th>Estado</th><th className="pay-devices-actions">Acciones</th></tr></thead>
      <tbody>
        {items.map(device=><tr key={device.id}>
          <td><strong>{device.name}</strong><small>{device.model??""}</small></td>
          <td>{platformLabel(device.platform)}</td>
          <td>{device.lastSeenAt?dateTime(device.lastSeenAt):"Sin contacto"}</td>
          <td><Status tone={device.active?"green":"gray"} icon={device.active?"circleCheck":"circleMinus"}>{device.active?"Activo":"Revocado"}</Status></td>
          <td className="pay-devices-actions">
            <IconButton icon="edit" label={`Renombrar ${device.name}`} onClick={()=>setRenameTarget(device)}/>
            <IconButton icon="cancel" label={`Revocar ${device.name}`} className="danger" onClick={()=>setRevokeTarget(device)}/>
          </td>
        </tr>)}
      </tbody>
    </table>}
    {pairing&&<PayPairingDialog onClose={closePairing} onLinked={linked}/>}
    <ConfirmDialog open={Boolean(revokeTarget)} tone="danger" title="Revocar dispositivo"
      description={`«${revokeTarget?.name}» dejará de reportar pagos de inmediato. Podrás vincularlo de nuevo con un QR nuevo.`}
      subject={revokeTarget?{label:"Dispositivo",name:revokeTarget.name}:undefined}
      confirmLabel="Revocar dispositivo" pending={revoke.isPending} onCancel={()=>setRevokeTarget(null)}
      onConfirm={()=>revokeTarget&&revoke.mutate(revokeTarget.id)}/>
    <RenameDialog target={renameTarget} pending={rename.isPending} onCancel={()=>setRenameTarget(null)} onConfirm={(id,name)=>rename.mutate({id,name})}/>
  </section>;
}

function RenameDialog({target,pending,onCancel,onConfirm}:{target:PayDevice|null;pending:boolean;onCancel:()=>void;onConfirm:(id:string,name:string)=>void}){
  const [name,setName]=useState("");
  return <ConfirmDialog open={Boolean(target)} title="Renombrar dispositivo" description="Usa un nombre que el administrador reconozca, p. ej. «Android de caja 1»."
    subject={target?{label:"Dispositivo",name:target.name}:undefined}
    confirmLabel="Guardar nombre" pending={pending} confirmDisabled={!name.trim()}
    onCancel={()=>{onCancel();setName("")}}
    onConfirm={()=>{if(target){onConfirm(target.id,name.trim());setName("")}}}>
    <Input aria-label="Nuevo nombre del dispositivo" maxLength={60} value={name} placeholder={target?.name??""} onChange={event=>setName(event.target.value)}/>
  </ConfirmDialog>;
}
