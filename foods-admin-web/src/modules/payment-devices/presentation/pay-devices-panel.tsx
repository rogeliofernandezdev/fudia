"use client";

import {useCallback,useState} from "react";
import {createPortal} from "react-dom";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,EmptyState,ErrorState,Icon,IconButton,Pagination,Select,Status,TableSkeleton} from "@/design-system";
import {DataTable} from "@/design-system/data-table";
import {useFeedback} from "@/providers";
import type {PayDevice,PayDeviceStatusFilter} from "../domain/types";
import {listPayDevices,revokePayDevice} from "../infrastructure/pay-devices-api";
import {PayPairingDialog} from "./pay-pairing-dialog";
import "./payment-devices.css";

/** Plataformas del contrato: hoy solo existe la app Android; iPhone queda listo. */
const platformMeta={ANDROID:{label:"Android",icon:"android"},IOS:{label:"iPhone",icon:"apple"}} as const;
const statusOptions:Array<{value:PayDeviceStatusFilter;label:string}>=[{value:"active",label:"Activos"},{value:"revoked",label:"Revocados"},{value:"",label:"Todos"}];
const PAGE_SIZE=10;

/** Clave compartida con el contador de la pestaña del local. */
export const payDevicesKey=(locationId:string,status:PayDeviceStatusFilter,page:number)=>["pay-devices",locationId,status,page,PAGE_SIZE] as const;
export const listActivePayDevices=(locationId:string)=>listPayDevices(locationId,{status:"active",page:1,size:PAGE_SIZE});

type Props={
  locationId:string;
  locationName:string;
  dateTime:(value:string)=>string;
};

/**
 * Pestaña «Dispositivos de pago» del formulario del local: los teléfonos que
 * reportan a la caja de ESE local los pagos de Yape y Plin. Se vincula por QR
 * y se revoca; no se edita: el nombre es el modelo que declara la app y volver
 * a vincular crea un registro nuevo. Los revocados quedan como historial
 * filtrable y la lista pagina en el servidor.
 */
export function PayDevicesPanel({locationId,locationName,dateTime}:Props){
  const qc=useQueryClient();
  const {notify}=useFeedback();
  const [status,setStatus]=useState<PayDeviceStatusFilter>("active");
  const [page,setPage]=useState(1);
  const [pairing,setPairing]=useState(false);
  const [revokeTarget,setRevokeTarget]=useState<PayDevice|null>(null);
  const devices=useQuery({
    queryKey:payDevicesKey(locationId,status,page),
    queryFn:()=>listPayDevices(locationId,{status,page,size:PAGE_SIZE}),
    refetchInterval:30000,
  });
  const refresh=useCallback(()=>void qc.invalidateQueries({queryKey:["pay-devices",locationId]}),[qc,locationId]);
  const closePairing=useCallback(()=>setPairing(false),[]);
  // Como WhatsApp Web: al terminar la vinculación el modal se cierra solo.
  const linked=useCallback((name:string)=>{
    setPairing(false);
    refresh();
    notify({tone:"success",title:"Dispositivo vinculado",message:`«${name}» ya reporta los pagos de Yape y Plin de ${locationName}.`});
  },[refresh,notify,locationName]);
  const revoke=useMutation({
    mutationFn:(id:string)=>revokePayDevice(locationId,id),
    onSuccess:device=>{
      setRevokeTarget(null);
      notify({tone:"success",title:"Dispositivo revocado",message:`«${device.name}» ya no puede reportar pagos hasta vincularse de nuevo.`});
      refresh();
    },
    onError:(error:Error)=>notify({tone:"danger",title:"No se pudo revocar",message:error.message}),
  });

  const items=devices.data?.items??[];
  // Con el filtro Activos o Revocados el estado ya lo dice el filtro: la
  // columna solo aparece en «Todos» y el nombre gana ancho en 390 px.
  const showState=status==="";
  const showActions=status!=="revoked";
  const columns=["Dispositivo",...(showState?["Estado"]:[]),...(showActions?["Acciones"]:[])];
  const empty=status==="revoked"
    ?{title:"Sin dispositivos revocados",text:"Aquí quedan los teléfonos que dejaron de reportar pagos."}
    :{title:"Aún no hay dispositivos vinculados",text:"Pulsa «Vincular dispositivo» y escanea el QR con Fudia Pay desde el Android del local."};
  return <section className="pay-devices-section" aria-label={`Dispositivos de pago de ${locationName}`}>
    <div className="pay-devices-toolbar">
      <Select chip aria-label="Filtrar por estado" value={status} onChange={event=>{setStatus(event.target.value as PayDeviceStatusFilter);setPage(1)}}>
        {statusOptions.map(option=><option key={option.value} value={option.value}>{option.label}</option>)}
      </Select>
      <Button icon="plus" onClick={()=>setPairing(true)}>Vincular dispositivo</Button>
    </div>
    {devices.isLoading?<TableSkeleton columns={columns} rows={3} label="Cargando dispositivos"/>
    :devices.isError?<ErrorState className="pay-empty" title="No pudimos cargar los dispositivos" text={devices.error.message} onRetry={()=>void devices.refetch()}/>
    :!items.length?<EmptyState className="pay-empty" icon="phoneApp" title={empty.title} text={empty.text}/>
    :<DataTable label="Dispositivos de pago" containerClassName="hover-scroll pay-devices-table">
      <thead><tr>{columns.map(column=><th scope="col" key={column}>{column}</th>)}</tr></thead>
      <tbody>
        {items.map((device,index)=>{
          const platform=platformMeta[device.platform]??platformMeta.ANDROID;
          const contact=device.active
            ?device.lastSeenAt?`Visto ${dateTime(device.lastSeenAt)}`:"Aún sin contacto"
            :`Revocado ${dateTime(device.revokedAt??device.createdAt)}`;
          return <tr className={index%2?"alternate":""} key={device.id}>
            <td><div className="pay-device-cell"><span className={`row-icon pay-device-platform ${device.platform==="IOS"?"is-ios":"is-android"}`} role="img" aria-label={platform.label} title={platform.label}><Icon name={platform.icon}/></span><div><b className="pay-device-name">{device.name}</b><small className="pay-device-contact">{contact}</small></div></div></td>
            {showState&&<td><Status tone={device.active?"green":"gray"} icon={device.active?"circleCheck":"circleMinus"}>{device.active?"Activo":"Revocado"}</Status></td>}
            {showActions&&<td><div className="table-actions">
              {device.active&&<IconButton icon="cancel" label={`Revocar ${device.name}`} className="danger" onClick={()=>setRevokeTarget(device)}/>}
            </div></td>}
          </tr>;
        })}
      </tbody>
    </DataTable>}
    <Pagination mode="simple" page={page} size={PAGE_SIZE} total={devices.data?.total??0} onPage={setPage}/>
    {/* El panel vive dentro del modal del local, cuyo fondo (backdrop-filter) y
        overflow recortarían otro modal fijo: el QR y la confirmación van al body. */}
    {pairing&&createPortal(<PayPairingDialog locationId={locationId} locationName={locationName} onClose={closePairing} onLinked={linked}/>,document.body)}
    {revokeTarget&&createPortal(<ConfirmDialog open tone="danger" title="Revocar dispositivo"
      description={`«${revokeTarget.name}» dejará de reportar pagos de inmediato. Podrás vincularlo de nuevo con un QR nuevo.`}
      subject={{label:"Dispositivo",name:revokeTarget.name}}
      confirmLabel="Revocar" pending={revoke.isPending} onCancel={()=>setRevokeTarget(null)}
      onConfirm={()=>revoke.mutate(revokeTarget.id)}/>,document.body)}
  </section>;
}

/** Activos del local para el contador de la pestaña; comparte caché con la lista. */
export function useActivePayDeviceCount(locationId:string|undefined,enabled:boolean){
  const query=useQuery({
    queryKey:payDevicesKey(locationId??"","active",1),
    queryFn:()=>listActivePayDevices(locationId!),
    enabled:enabled&&Boolean(locationId),
    refetchInterval:30000,
  });
  return query.data?.total;
}
