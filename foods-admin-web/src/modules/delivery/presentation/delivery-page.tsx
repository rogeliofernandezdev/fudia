"use client";
import {useEffect,useState} from "react";
import {PageHeader,Tabs,TabsBar} from "@/design-system";
import {DeliveryPartnersAdmin} from "@/modules/platform/presentation/delivery-partners-admin";
import {deliveryTabFrom,deliveryTabPath,deliveryTabs,type DeliveryTab} from "../domain/board";
import {DeliveryFeesPanel} from "./delivery-fees-panel";
import {FailedDeliveries} from "./failed-deliveries";
import "./delivery.css";

/**
 * Delivery: todo el delivery lo administra Fudia, así que la opción del menú es
 * solo de Plataforma (2026-11-23). La pestaña viaja en `?tab=` para compartir y
 * recargar sin perderla; tarifas e incidencias son del local activo.
 */
export function DeliveryPage({initialTab}:{initialTab?:string}){
  const[requested,setRequested]=useState(initialTab??"providers");
  const tab=deliveryTabFrom(requested);
  // En móvil la pista de pestañas se desplaza: la activa (por ejemplo, abierta desde la URL) queda a la vista.
  useEffect(()=>{document.querySelector(".delivery-tabs [aria-selected=true]")?.scrollIntoView({block:"nearest",inline:"nearest"});},[tab]);
  const change=(next:DeliveryTab)=>{
    setRequested(next);
    window.history.replaceState(window.history.state,"",deliveryTabPath(next));
  };
  return <>
    <PageHeader title="Delivery" description="Centrales, tarifas de envío e incidencias del local activo."/>
    <TabsBar><Tabs className="delivery-tabs" label="Delivery" value={tab} onChange={change} items={deliveryTabs.map(({value,label,icon})=>({value,label,icon}))}/></TabsBar>
    {tab==="providers"&&<DeliveryPartnersAdmin/>}
    {tab==="fees"&&<DeliveryFeesPanel/>}
    {tab==="failed"&&<FailedDeliveries/>}
  </>;
}
