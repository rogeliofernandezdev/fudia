"use client";
import {useState} from "react";
import QrImage from "next/image";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,ErrorState,FormSection,Skeleton} from "@/design-system";
import {readCssToken} from "@/design-system/theme";
import {useFeedback} from "@/providers";
import {publicConciergePath} from "@/shared/routing/page-routes";
import {hasDeliveryLink} from "../domain/dispatch-status";
import {getDeliveryLink,rotateDeliveryLink} from "../infrastructure/delivery-api";
import "./delivery.css";

const loadQRCode=()=>import("qrcode").then(module=>module.default);

/**
 * Sección «Delivery» del formulario del local: enlace/QR para pedir por
 * WhatsApp ya ligado a este local. Las centrales y la activación de Delivery
 * Fudia son globales (Plataforma); el enlace se genera al instante.
 */
export function LocationDeliverySection({locationId,locationName,disabled,editable}:{locationId:string;locationName:string;disabled?:boolean;editable:boolean}){
  return <FormSection icon="motorbike" tone="teal" title="Delivery" bare>
    <DeliveryLinkPanel locationId={locationId} locationName={locationName} editable={editable} disabled={disabled}/>
  </FormSection>;
}

function DeliveryLinkPanel({locationId,locationName,editable,disabled}:{locationId:string;locationName:string;editable:boolean;disabled?:boolean}){
  const{notify}=useFeedback();
  const client=useQueryClient();
  const link=useQuery({queryKey:["delivery-link",locationId],queryFn:()=>getDeliveryLink(locationId),retry:false,refetchOnWindowFocus:false});
  const[confirm,setConfirm]=useState(false);
  const origin=typeof window==="undefined"?"":window.location.origin;
  const rotate=useMutation({
    mutationFn:()=>rotateDeliveryLink(locationId),
    onSuccess:(data,_,context)=>{
      client.setQueryData(["delivery-link",locationId],data);
      setConfirm(false);
      notify({tone:"success",title:context?"QR regenerado":"QR generado",message:context?"El QR y el enlace anteriores ya no abren el chat del local.":"Descárgalo para imprimirlo o copia el enlace para compartirlo."});
    },
    onMutate:()=>hasDeliveryLink(link.data),
    onError:e=>notify({tone:"danger",title:"No se pudo generar el QR",message:e.message}),
  });
  const url=hasDeliveryLink(link.data)&&origin?`${origin}${publicConciergePath(link.data.token)}`:"";
  const qr=useQuery({
    queryKey:["delivery-qr",url],
    queryFn:()=>createDeliveryQr(url),
    enabled:Boolean(url),staleTime:Infinity,retry:false,refetchOnWindowFocus:false,
  });
  const download=useMutation({
    mutationFn:async()=>{
      if(!qr.data)throw new Error("Espera a que termine de cargar el QR.");
      await downloadDeliveryQr(qr.data,locationName);
    },
    onError:e=>notify({tone:"danger",title:"No se pudo descargar el QR",message:e.message}),
  });
  const copy=useMutation({
    mutationFn:async()=>{await navigator.clipboard.writeText(url);},
    onSuccess:()=>notify({tone:"success",title:"Enlace copiado",message:"Pégalo donde tus clientes te encuentran."}),
    onError:()=>notify({tone:"danger",title:"No se pudo copiar",message:"Revisa el permiso de portapapeles de tu navegador y vuelve a intentarlo."}),
  });
  return <div className="delivery-link">
    <h4 className="delivery-link-title">QR para hacer pedidos</h4>
    {link.isLoading?<Skeleton height="var(--control-height)"/>
    :link.isError?<ErrorState title="No pudimos cargar el QR" text={link.error.message} onRetry={()=>void link.refetch()} retrying={link.isFetching}/>
    :url?<div className="delivery-link-content">
      <div className="delivery-link-preview" aria-busy={qr.isFetching}>
        {qr.isError?<ErrorState title="No pudimos preparar la imagen del QR" text={qr.error.message} onRetry={()=>void qr.refetch()} retrying={qr.isFetching}/>
        :qr.data?<QrImage className="delivery-link-qr" src={qr.data} width={200} height={200} alt={`QR para pedir delivery o recojo en ${locationName}`} unoptimized/>
        :<div role="status" aria-label="Preparando imagen del QR"><Skeleton className="delivery-link-qr" width="var(--size-200)" height="var(--size-200)"/></div>}
      </div>
      <div className="delivery-link-actions">
        <Button kind="secondary" icon="download" disabled={disabled||rotate.isPending||download.isPending||!qr.data} loading={download.isPending} onClick={()=>download.mutate()}>{download.isPending?"Descargando QR…":"Descargar QR"}</Button>
        <Button kind="secondary" icon="link" disabled={disabled||rotate.isPending||copy.isPending} loading={copy.isPending} onClick={()=>copy.mutate()}>{copy.isPending?"Copiando enlace…":"Copiar enlace"}</Button>
        {editable&&<Button kind="danger-outline" icon="refresh" disabled={disabled||rotate.isPending||download.isPending} onClick={()=>setConfirm(true)}>Regenerar QR</Button>}
      </div>
    </div>
    :editable?<Button kind="secondary" icon="qr" disabled={disabled||rotate.isPending} loading={rotate.isPending} onClick={()=>rotate.mutate()}>Generar QR</Button>
    :<p className="delivery-link-empty">Aún no hay QR de delivery.</p>}
    <ConfirmDialog open={confirm} title="Regenerar QR" subject={{label:"Local",name:locationName}} description="El QR y el enlace actuales dejarán de abrir el chat del local." tone="danger" confirmLabel="Regenerar QR" pending={rotate.isPending} onCancel={()=>setConfirm(false)} onConfirm={()=>rotate.mutate()}/>
  </div>;
}

async function createDeliveryQr(url:string){
  const QRCode=await loadQRCode();
  return QRCode.toDataURL(url,{width:480,margin:4,color:{dark:readCssToken("--navigation-900"),light:readCssToken("--surface")}});
}

/** Descarga la misma imagen del QR que se muestra, con el nombre del local. */
async function downloadDeliveryQr(qr:string,locationName:string){
  const canvas=document.createElement("canvas");
  canvas.width=560;canvas.height=680;
  const ctx=canvas.getContext("2d");
  if(!ctx)throw new Error("Tu navegador no permite preparar la imagen para descargar.");
  ctx.fillStyle=readCssToken("--surface");ctx.fillRect(0,0,canvas.width,canvas.height);
  ctx.fillStyle=readCssToken("--primary-700");ctx.textAlign="center";
  ctx.font=`${readCssToken("--weight-bold")} 30px ${readCssToken("--font-sans")}`;
  ctx.fillText(locationName,280,64,480);
  const image=new Image();
  await new Promise<void>((resolve,reject)=>{image.onload=()=>resolve();image.onerror=()=>reject(new Error("No se pudo preparar la imagen. Vuelve a intentarlo."));image.src=qr;});
  ctx.drawImage(image,40,96,480,480);
  ctx.fillStyle=readCssToken("--ink-700");
  ctx.font=`22px ${readCssToken("--font-sans")}`;
  ctx.fillText("Pide delivery o recojo desde este QR",280,630);
  const anchor=document.createElement("a");
  anchor.download=`qr-delivery-${locationName.trim().toLowerCase().replace(/\s+/g,"-")||"local"}.png`;
  anchor.href=canvas.toDataURL("image/png");
  anchor.click();
}
