"use client";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,ErrorState,Skeleton,Status} from "@/design-system";
import {useFeedback} from "@/providers";
import {getOrganizationDelivery,setOrganizationDelivery} from "../infrastructure/platform-api";

/**
 * Detalle de empresa (Plataforma) › Delivery Fudia: se activa por empresa y
 * aplica a todos sus locales; las centrales y su orden son globales.
 */
export function DeliveryCompanyPanel({organizationId}:{organizationId:string}){
 const{notify}=useFeedback();
 const client=useQueryClient();
 const key=["platform-organization-delivery",organizationId];
 const delivery=useQuery({queryKey:key,queryFn:()=>getOrganizationDelivery(organizationId)});
 const[confirm,setConfirm]=useState(false);
 const toggle=useMutation({
  mutationFn:(enabled:boolean)=>setOrganizationDelivery(organizationId,enabled),
  onSuccess:data=>{
   client.setQueryData(key,data);
   setConfirm(false);
   notify({tone:"success",title:data.enabled?"Delivery Fudia activado":"Delivery Fudia desactivado",message:data.enabled?"Los pedidos de delivery de sus locales se ofrecerán a las centrales.":"Sus locales vuelven a su delivery propio; los pedidos ya ofrecidos siguen su curso."});
  },
  onError:e=>notify({tone:"danger",title:"No se pudo guardar",message:e.message}),
 });
 const enabled=Boolean(delivery.data?.enabled);
 return <section className="panel concierge-panel delivery-company-panel" aria-labelledby="delivery-company-title">
  <header><div><small>Delivery Fudia</small><h2 id="delivery-company-title">Centrales de reparto</h2></div>{delivery.data&&<Status active={enabled}>{enabled?"Activo":"Inactivo"}</Status>}</header>
  {delivery.isLoading?<div className="delivery-company-body" role="status" aria-busy="true" aria-label="Cargando Delivery Fudia"><Skeleton height="var(--size-11)"/><Skeleton width="var(--size-160)" height="var(--control-height)"/></div>
  :delivery.isError?<ErrorState title="No pudimos cargar Delivery Fudia" text={delivery.error.message} retrying={delivery.isFetching} onRetry={()=>void delivery.refetch()}/>
  :<div className="delivery-company-body">
   <p>{enabled?"Los pedidos de delivery de todos sus locales se ofrecen a las centrales de Fudia.":"Sus locales usan su propio delivery; Fudia solo recibe los pedidos."}</p>
   <Button kind={enabled?"secondary":"primary"} icon="motorbike" disabled={toggle.isPending} onClick={()=>setConfirm(true)}>{enabled?"Desactivar Delivery Fudia":"Activar Delivery Fudia"}</Button>
  </div>}
  <ConfirmDialog open={confirm} title={enabled?"Desactivar Delivery Fudia":"Activar Delivery Fudia"} description={enabled?"Sus locales volverán a su delivery propio. Los pedidos ya ofrecidos a una central siguen su curso.":"Los pedidos de delivery confirmados en todos sus locales se ofrecerán a las centrales, en su orden."} tone={enabled?"danger":"success"} confirmLabel={enabled?"Desactivar":"Activar"} pending={toggle.isPending} onCancel={()=>setConfirm(false)} onConfirm={()=>toggle.mutate(!enabled)}/>
 </section>;
}
