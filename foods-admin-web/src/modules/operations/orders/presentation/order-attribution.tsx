import {Icon} from "@/design-system/icons";

type Props={channel:string;waiterName?:string;collectedByNames?:string[]};

/** Responsables en una línea de la identidad: mozo (solo salón) y quién cobró. */
export function OrderAttribution({channel,waiterName,collectedByNames=[]}:Props){
  const parts=[channel==="salon"&&`Salón · Atendido por ${waiterName||"sin asignar"}`,collectedByNames.length>0&&`Cobrado por ${collectedByNames.join(", ")}`].filter(Boolean);
  if(!parts.length)return null;
  return <p className="order-attribution" aria-label="Responsables de la atención y del cobro"><Icon name={channel==="salon"?"utensils":"users"} size={14}/><span>{parts.join(" · ")}</span></p>;
}
