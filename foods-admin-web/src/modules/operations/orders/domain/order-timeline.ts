import type {Order} from "./types";

export type OrderTimelineStep={
  key:"registered"|"service"|"bill"|"payment"|"completed"|"cancelled";
  label:string;
  state:"done"|"current"|"pending";
  /** Hora real registrada por el backend; sin hora no se inventa una. */
  at?:string;
};

const serviceLabels:Record<string,string>={
  nuevo:"Sin enviar a preparación",
  confirmado:"Comanda enviada",
  preparando:"En preparación",
  listo:"Listo para entregar",
  en_camino:"En camino",
  entregado:"Entregado",
};

/**
 * Seguimiento del pedido con los hitos que el backend registra: alta, cierre de
 * cuenta y cierre de la atención tienen hora; preparación y pago muestran su
 * estado actual sin inventar una hora que no existe.
 */
export function orderTimeline(order:Order):OrderTimelineStep[]{
  const salon=order.channel==="salon";
  const steps:OrderTimelineStep[]=[{key:"registered",label:"Pedido registrado",state:"done",at:order.createdAt}];
  if(order.status==="cancelado")return [...steps,{key:"cancelled",label:"Pedido cancelado",state:"done"}];
  const delivered=order.status==="entregado";
  steps.push({key:"service",label:serviceLabels[order.status]??order.status,state:delivered?"done":"current"});
  if(salon)steps.push({key:"bill",label:"Cuenta cerrada",state:order.billClosedAt?"done":"pending",at:order.billClosedAt});
  const paid=order.paymentStatus==="paid"||order.accountState==="paid";
  const partial=order.paymentStatus==="partial";
  steps.push({key:"payment",label:paid?"Pago completo":partial?"Pago parcial":"Pago",state:paid?"done":partial?"current":"pending"});
  steps.push({key:"completed",label:salon?"Mesa liberada":"Pedido completado",state:order.completedAt?"done":"pending",at:order.completedAt});
  return steps;
}
