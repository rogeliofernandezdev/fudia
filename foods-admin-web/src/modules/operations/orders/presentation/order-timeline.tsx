import {IconTile,type IconName,type IconTileTone} from "@/design-system";
import type {Order} from "../domain/types";
import {orderTimeline,type OrderTimelineStep} from "../domain/order-timeline";
import {registeredAt} from "./order-meta";

const look:Record<OrderTimelineStep["key"],{icon:IconName;tone:IconTileTone}>={
  registered:{icon:"orders",tone:"primary"},
  service:{icon:"flame",tone:"warning"},
  bill:{icon:"receipt",tone:"sky"},
  payment:{icon:"cash",tone:"brand"},
  completed:{icon:"circleCheck",tone:"neutral"},
  cancelled:{icon:"cancel",tone:"danger"},
};

/** El hito de preparación toma el tono de su estado según la matriz de operación. */
const serviceLook:Record<string,{icon:IconName;tone:IconTileTone}>={
  nuevo:{icon:"clock",tone:"neutral"},
  confirmado:{icon:"send",tone:"primary"},
  preparando:{icon:"flame",tone:"warning"},
  listo:{icon:"bell",tone:"purple"},
  en_camino:{icon:"motorbike",tone:"navy"},
  entregado:{icon:"check",tone:"brand"},
};

/** Seguimiento del pedido: hitos reales con su hora; los pendientes quedan en gris. */
export function OrderTimeline({order,country,timeZone}:{order:Order;country?:string;timeZone?:string}){
  const steps=orderTimeline(order);
  return <section className="order-timeline" aria-labelledby="order-timeline-title">
    <h3 id="order-timeline-title" className="order-timeline-title">Progreso del pedido</h3>
    <ol>{steps.map(step=>{
      const style=step.key==="service"?serviceLook[order.status]??look.service:look[step.key];
      // Los hitos alcanzados usan su color vivo con icono blanco; los pendientes quedan en gris tenue.
      return <li key={step.key} className={`is-${step.state}`}>
        {step.state==="pending"?<IconTile className="order-timeline-dot" name={style.icon} tone="neutral"/>:<IconTile className="order-timeline-dot" name={style.icon} tone={style.tone} appearance="vivid"/>}
        <div><b>{step.label}</b><small>{step.at?<time dateTime={step.at}>{registeredAt(step.at,country,timeZone)}</time>:step.state==="current"?"Estado actual":step.state==="pending"?"Pendiente":"Registrado"}</small></div>
      </li>;
    })}</ol>
  </section>;
}
