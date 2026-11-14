"use client";
import {useEffect,useState} from "react";
import {EmptyState,PageHeader,Tabs,TabsBar} from "@/design-system";
import {useSession} from "@/providers";
import {deliveryTabFrom,deliveryTabPath,visibleDeliveryTabs,type DeliveryTab} from "../domain/board";
import {DeliveryBoard} from "./delivery-board";
import {DeliveryFeesPanel} from "./delivery-fees-panel";
import {DeliveryProviders} from "./delivery-providers";
import {FailedDeliveries} from "./failed-deliveries";
import "./delivery.css";

function ComingSoon(){
  return <section className="panel management"><EmptyState icon="settings" title="Próximamente" text="Este módulo está en desarrollo. Pronto estará disponible para tu empresa."/></section>;
}

/**
 * Delivery: una sola opción del menú con todo el delivery del local en
 * pestañas. La pestaña viaja en `?tab=` para compartir y recargar sin perderla.
 */
export function DeliveryPage({initialTab}:{initialTab?:string}){
  const{modules}=useSession();
  const[requested,setRequested]=useState(initialTab??"orders");
  const tab=deliveryTabFrom(requested,modules);
  // En móvil la pista de pestañas se desplaza: la activa (por ejemplo, abierta desde la URL) queda a la vista.
  useEffect(()=>{document.querySelector(".delivery-tabs [aria-selected=true]")?.scrollIntoView({block:"nearest",inline:"nearest"});},[tab]);
  const change=(next:DeliveryTab)=>{
    setRequested(next);
    window.history.replaceState(window.history.state,"",deliveryTabPath(next));
  };
  return <>
    <PageHeader title="Delivery" description="Repartos en curso, centrales, tarifas e incidencias de tu local."/>
    <TabsBar><Tabs className="delivery-tabs" label="Delivery" value={tab} onChange={change} items={visibleDeliveryTabs(modules).map(({value,label,icon})=>({value,label,icon}))}/></TabsBar>
    {tab==="orders"&&<DeliveryBoard/>}
    {tab==="providers"&&<DeliveryProviders/>}
    {tab==="fees"&&<DeliveryFeesPanel/>}
    {tab==="failed"&&<FailedDeliveries/>}
    {(tab==="apps"||tab==="couriers")&&<ComingSoon/>}
  </>;
}
