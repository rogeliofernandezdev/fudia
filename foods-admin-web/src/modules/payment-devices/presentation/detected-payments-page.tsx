"use client";

import {useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {Button,PageHeader,Tabs,TabsBar} from "@/design-system";
import {useSession} from "@/providers";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {listPayDevices,listPayNotificationPage} from "../infrastructure/pay-devices-api";
import {PayDevicesPanel} from "./pay-devices-panel";
import {PayNotificationsPanel} from "./pay-notifications-panel";
import "./payment-devices.css";

type DetectedTab="payments"|"devices";

/**
 * Pagos detectados (Operación): página propia, fuera de Caja, porque la lista
 * crece con el historial del local. Dos pestañas: «Pagos», con filtros y
 * paginación en el servidor, y «Dispositivos», los Android que reportan Yape y
 * Plin. «Vincular dispositivo» vive en la barra de pestañas. La ve quien tiene
 * «Caja y turnos»: comparte su acceso y sus permisos.
 */
export function DetectedPaymentsPage(){
  const {location}=useSession();
  const [tab,setTab]=useState<DetectedTab>("payments");
  const [pairing,setPairing]=useState(false);
  const devices=useQuery({queryKey:["pay-devices"],queryFn:listPayDevices,refetchInterval:30000});
  // Solo el contador de la pestaña: los pagos que esperan revisión.
  const pending=useQuery({
    queryKey:["pay-notifications","pending-count"],
    queryFn:()=>listPayNotificationPage({status:"detected",provider:"",page:1,size:10}),
    refetchInterval:10000,
  });
  const dateTime=(value:string)=>formatRegionalDateTime(value,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium",timeStyle:"short"});
  const pendingCount=pending.data?.total??0;
  const deviceCount=devices.data?.items.filter(device=>device.active).length??0;

  return <div className="detected-payments-page">
    <PageHeader title="Pagos detectados" description="Pagos de Yape y Plin que reportan los teléfonos del local."/>
    <TabsBar action={<Button icon="plus" onClick={()=>{setTab("devices");setPairing(true)}}>Vincular dispositivo</Button>}>
      <Tabs label="Pagos detectados" value={tab} onChange={setTab} items={[
        {value:"payments",label:"Pagos",icon:"wallet",...(pendingCount?{count:pendingCount}:{})},
        {value:"devices",label:"Dispositivos",icon:"phoneApp",...(deviceCount?{count:deviceCount}:{})},
      ]}/>
    </TabsBar>
    <section className="panel standardized-management pay-page-panel">
      {tab==="payments"
        ?<><h2 className="sr-only">Pagos</h2><PayNotificationsPanel dateTime={dateTime}/></>
        :<><h2 className="sr-only">Dispositivos</h2><PayDevicesPanel dateTime={dateTime} pairing={pairing} onPairingChange={setPairing}/></>}
    </section>
  </div>;
}
