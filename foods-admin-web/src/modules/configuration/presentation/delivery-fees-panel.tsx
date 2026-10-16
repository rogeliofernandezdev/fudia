"use client";
import Link from "next/link";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ErrorState,InlineNotice,Input,MoneyInput,RowActionButton,Skeleton} from "@/design-system";
import {useFeedback} from "@/providers/feedback-provider";
import {useSession} from "@/providers/session-context";
import {pageRoutes} from "@/shared/routing/page-routes";
import {deliveryFeeErrors,deliveryFeeLimits,deliveryFeeRows,deliveryFeesPayload,type DeliveryFeeRow} from "../domain/delivery-fees";
import {getDeliveryFees,saveDeliveryFees} from "../infrastructure/concierge-settings-api";

const queryKey=["concierge-delivery-fees"];

/** Rangos de costo de envío por distancia en línea recta para pedidos sin mesa de Concierge. */
export function DeliveryFeesPanel(){
 const{can}=useSession();
 const{notify}=useFeedback();
 const client=useQueryClient();
 const canManage=can("organizations.manage");
 const query=useQuery({queryKey,queryFn:getDeliveryFees});
 const[draft,setDraft]=useState<DeliveryFeeRow[]|null>(null);
 const[submitted,setSubmitted]=useState(false);
 const rows=draft??(query.data?deliveryFeeRows(query.data.tiers):[]);
 const errors=submitted?deliveryFeeErrors(rows):{};
 const save=useMutation({
  mutationFn:()=>saveDeliveryFees(deliveryFeesPayload(rows)),
  onSuccess:data=>{
   client.setQueryData(queryKey,data);setDraft(null);setSubmitted(false);
   notify({tone:"success",title:"Tarifas guardadas",message:data.tiers.length?"Concierge cotizará el delivery con estos rangos.":"Concierge solo ofrecerá recojo en el local."});
  },
  onError:e=>notify({tone:"danger",title:"No se pudieron guardar",message:e.message}),
 });
 function update(key:string,patch:Partial<DeliveryFeeRow>){setDraft(rows.map(row=>row.key===key?{...row,...patch}:row))}
 function add(){setDraft([...rows,{key:globalThis.crypto.randomUUID(),maxDistanceKm:"",fee:""}])}
 function remove(key:string){setDraft(rows.filter(row=>row.key!==key))}
 function submit(){setSubmitted(true);if(!Object.keys(deliveryFeeErrors(rows)).length)save.mutate()}

 return <section className="panel concierge-delivery" aria-labelledby="concierge-delivery-title">
  <header><h2 id="concierge-delivery-title">Delivery y recojo por WhatsApp</h2><p>Cuando termina la atención en mesa, Concierge toma pedidos para recojo en el local o delivery. El costo de envío se calcula por la distancia en línea recta desde el local hasta la ubicación que comparte el cliente.</p></header>
  {query.isLoading?<DeliveryFeesSkeleton/>:query.isError?<ErrorState title="No pudimos cargar las tarifas de delivery" text={query.error.message} onRetry={()=>void query.refetch()} retrying={query.isFetching}/>:query.data&&<>
   {!query.data.hasCoordinates&&<InlineNotice tone="warning" title="Falta la ubicación del local" action={<Link className="button secondary" href={pageRoutes.locations}>Ir a Locales</Link>}>Ubica {query.data.locationName} en el mapa para cotizar el delivery. Mientras tanto Concierge solo ofrece recojo.</InlineNotice>}
   {rows.length?<div className="delivery-fee-table" role="table" aria-label="Rangos de costo de envío">
    <div className="delivery-fee-row head" role="row"><span role="columnheader">HASTA (KM)</span><span role="columnheader">COSTO DE ENVÍO</span>{canManage&&<span role="columnheader" className="sr-only">Acciones</span>}</div>
    {rows.map((row,index)=><div className="delivery-fee-row" role="row" key={row.key}>
     <span role="cell"><Input inputMode="decimal" className="tabular" aria-label={`Distancia máxima del rango ${index+1} en km`} aria-invalid={Boolean(errors[row.key]?.maxDistanceKm)} value={row.maxDistanceKm} disabled={!canManage||save.isPending} onChange={e=>update(row.key,{maxDistanceKm:e.target.value})}/>{errors[row.key]?.maxDistanceKm&&<small className="delivery-fee-error">{errors[row.key]?.maxDistanceKm}</small>}</span>
     <span role="cell"><MoneyInput symbol={query.data.currencySymbol} className="tabular" aria-label={`Costo de envío del rango ${index+1}`} aria-invalid={Boolean(errors[row.key]?.fee)} value={row.fee} disabled={!canManage||save.isPending} onChange={e=>update(row.key,{fee:e.target.value})}/>{errors[row.key]?.fee&&<small className="delivery-fee-error">{errors[row.key]?.fee}</small>}</span>
     {canManage&&<span role="cell"><RowActionButton action="remove" label={`Quitar rango ${index+1}`} disabled={save.isPending} onClick={()=>remove(row.key)}/></span>}
    </div>)}
   </div>:<div className="concierge-note">Sin rangos configurados: Concierge solo ofrecerá recojo en el local.</div>}
   <p className="delivery-fee-help">Cada rango empieza donde termina el anterior. Más allá del mayor límite no hay cobertura de delivery.</p>
   {canManage&&<footer className="delivery-fee-actions">
    <Button kind="secondary" icon="plus" onClick={add} disabled={rows.length>=deliveryFeeLimits.tiers||save.isPending}>Agregar rango</Button>
    {draft&&<Button kind="ghost" onClick={()=>{setDraft(null);setSubmitted(false)}} disabled={save.isPending}>Descartar cambios</Button>}
    <Button icon="save" onClick={submit} loading={save.isPending} disabled={!draft}>Guardar tarifas</Button>
   </footer>}
  </>}
 </section>;
}

function DeliveryFeesSkeleton(){
 return <div className="delivery-fee-table" role="status" aria-busy="true" aria-label="Cargando tarifas de delivery">
  {Array.from({length:3},(_,i)=><div className="delivery-fee-row" key={i} aria-hidden="true"><Skeleton height="var(--size-44)" radius="var(--radius-tile)"/><Skeleton height="var(--size-44)" radius="var(--radius-tile)"/></div>)}
 </div>;
}
