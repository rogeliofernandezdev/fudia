import {Icon} from "@/design-system/icons";
import {Skeleton} from "@/design-system/skeleton";

type Props={channel:string;waiterName?:string;collectedByNames?:string[]};

/** Identidad legible: etiqueta y persona, con el icono como apoyo. */
export function OrderAttribution({channel,waiterName,collectedByNames=[]}:Props){
  if(channel!=="salon"&&!collectedByNames.length)return null;
  return <div className="order-attribution" aria-label="Responsables de la atención y del cobro">
    {channel==="salon"&&<div><small><Icon name="users" size={16}/>Atendido por</small><b>{waiterName||"Sin asignar"}</b></div>}
    {collectedByNames.length>0&&<div className="order-attribution-collector"><small><Icon name="payment" size={16}/>Cobrado por</small><b>{collectedByNames.join(", ")}</b></div>}
  </div>;
}

export function OrderAttributionSkeleton(){
  return <div className="order-attribution" aria-hidden="true">
    {Array.from({length:2},(_,index)=><div key={index}><Skeleton width="var(--size-48)" height="var(--size-9)"/><Skeleton width="var(--size-92)" height="var(--size-12)"/></div>)}
  </div>;
}
