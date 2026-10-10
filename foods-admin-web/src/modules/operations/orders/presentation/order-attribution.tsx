import {Icon} from "@/design-system/icons";
import {Skeleton} from "@/design-system/skeleton";

type Props={channel:string;waiterName?:string;collectedByNames?:string[]};

export function OrderAttribution({channel,waiterName,collectedByNames=[]}:Props){
  if(channel!=="salon"&&!collectedByNames.length)return null;
  return <dl className="order-attribution" aria-label="Responsables de la atención y del cobro">
    {channel==="salon"&&<div><dt><Icon name="users" size={15}/>Mozo</dt><dd>{waiterName||"Sin asignar"}</dd></div>}
    {collectedByNames.length>0&&<div><dt><Icon name="payment" size={15}/>Cobrado por</dt><dd>{collectedByNames.join(" · ")}</dd></div>}
  </dl>;
}

export function OrderAttributionSkeleton(){
  return <div className="order-attribution" aria-hidden="true">
    {Array.from({length:2},(_,index)=><div key={index}><Skeleton width="var(--size-48)" height="var(--size-9)"/><Skeleton width="var(--size-92)" height="var(--size-12)"/></div>)}
  </div>;
}
