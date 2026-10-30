import {Icon} from "@/design-system/icons";
import {Skeleton} from "@/design-system/skeleton";

type Props={channel:string;waiterName?:string;collectedByNames?:string[]};

/** Responsables en una línea de la identidad: mozo (solo salón) y quién cobró. */
export function OrderAttribution({channel,waiterName,collectedByNames=[]}:Props){
  const parts=[channel==="salon"&&`Salón · Atendido por ${waiterName||"sin asignar"}`,collectedByNames.length>0&&`Cobrado por ${collectedByNames.join(", ")}`].filter(Boolean);
  if(!parts.length)return null;
  return <p className="order-attribution" aria-label="Responsables de la atención y del cobro"><Icon name={channel==="salon"?"utensils":"users"} size={14}/><span>{parts.join(" · ")}</span></p>;
}

export function OrderAttributionSkeleton(){
  return <div className="order-attribution" aria-hidden="true">
    {Array.from({length:2},(_,index)=><div key={index}><Skeleton width="var(--size-48)" height="var(--size-9)"/><Skeleton width="var(--size-92)" height="var(--size-12)"/></div>)}
  </div>;
}
