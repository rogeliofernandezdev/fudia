"use client";
import {useState} from "react";
import {useMutation,useQueryClient} from "@tanstack/react-query";
import {Dialog} from "@/design-system/dialog";
import {Button,ConfirmDialog,FormField,Input,ModalBody,ModalHeader} from "@/design-system";
import {useFeedback} from "@/providers";
import {publicPartnerPanelPath} from "@/shared/routing/page-routes";
import type {DeliveryPartner} from "../domain/types";
import {rotateDeliveryPartnerPanel} from "../infrastructure/platform-api";

/**
 * Enlace privado del panel de reparto de una central: quien lo abre recibe sus
 * pedidos y confirma la atención, la recogida y la entrega. Plataforma lo
 * copia, lo abre o lo reemplaza si se filtró.
 */
export function DeliveryPartnerPanelLink({partner,queryKey,onClose}:{partner:DeliveryPartner;queryKey:string[];onClose:()=>void}){
  const{notify}=useFeedback();
  const client=useQueryClient();
  const[confirm,setConfirm]=useState(false);
  const origin=typeof window==="undefined"?"":window.location.origin;
  const url=`${origin}${publicPartnerPanelPath(partner.panelToken)}`;
  const copy=useMutation({
    mutationFn:async()=>{await navigator.clipboard.writeText(url);},
    onSuccess:()=>notify({tone:"success",title:"Enlace copiado",message:"Envíalo solo a la central: abre su panel sin contraseña."}),
    onError:()=>notify({tone:"danger",title:"No se pudo copiar",message:"Revisa el permiso de portapapeles de tu navegador y vuelve a intentarlo."}),
  });
  const rotate=useMutation({
    mutationFn:()=>rotateDeliveryPartnerPanel(partner.id),
    onSuccess:()=>{
      setConfirm(false);
      void client.invalidateQueries({queryKey});
      notify({tone:"success",title:"Enlace nuevo generado",message:"El enlace anterior ya no abre el panel. Envía el nuevo a la central."});
    },
    onError:e=>{setConfirm(false);notify({tone:"danger",title:"No se pudo generar el enlace",message:e.message});},
  });
  return <>
    <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={onClose} className="crud-modal platform-partner-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="delivery-partner-panel-title">
      <ModalHeader title="Panel de reparto" titleId="delivery-partner-panel-title" onClose={onClose} closeLabel="Cerrar"/>
      <ModalBody><div className="partner-panel-link">
        <p>En este panel <b>{partner.name}</b> recibe los pedidos, confirma la atención, la recogida y la entrega, y comparte su ubicación con el cliente.</p>
        <FormField label="Enlace del panel" help="Abre el panel sin contraseña: compártelo solo con la central."><Input value={url} readOnly onFocus={event=>event.target.select()}/></FormField>
        <div className="partner-panel-link-actions">
          <Button kind="secondary" icon="link" disabled={copy.isPending||rotate.isPending} loading={copy.isPending} onClick={()=>copy.mutate()}>Copiar enlace</Button>
          <Button kind="secondary" icon="arrowUpRight" disabled={rotate.isPending} onClick={()=>window.open(url,"_blank","noopener,noreferrer")}>Abrir panel</Button>
          <Button kind="danger-outline" icon="refresh" disabled={rotate.isPending} onClick={()=>setConfirm(true)}>Generar enlace nuevo</Button>
        </div>
      </div></ModalBody>
    </Dialog></div>
    <ConfirmDialog open={confirm} title="Generar enlace nuevo" subject={{label:"Central",name:partner.name}} description="El enlace actual dejará de abrir el panel de inmediato. La central necesitará el enlace nuevo para seguir recibiendo pedidos." tone="danger" confirmLabel="Generar enlace" pending={rotate.isPending} onCancel={()=>setConfirm(false)} onConfirm={()=>rotate.mutate()}/>
  </>;
}
