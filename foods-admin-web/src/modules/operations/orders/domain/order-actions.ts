type OperationalOrder={channel:string;status:string;paymentStatus?:string;remainingAmount?:string;deliveryDispatchStatus?:string;completedAt?:string;serviceItems?:{status:string}[]};
type OrderAction={status:string;label:string;icon:"receipt"|"truck"|"check"};
type CancellableOrder={status:string;completedAt?:string;paidAmount?:string;deliveryDispatchStatus?:string;serviceItems?:{destination:string;status:string}[]};

/** Delivery Fudia con central asignada y pedido listo o en camino: ya está en proceso de entrega. */
export function deliveryInProgress(order:Pick<CancellableOrder,"status"|"deliveryDispatchStatus">):boolean{
  return ["accepted","picked_up","delivered"].includes(order.deliveryDispatchStatus??"")&&(order.status==="listo"||order.status==="en_camino");
}

/**
 * Delivery Fudia buscando, sin central o asignado: el repartidor
 * recoge («En camino») y entrega («Entregado») desde su panel, y eso es lo que ve
 * el cliente en el chat. Caja solo cobra; el delivery propio sigue en caja.
 */
export function partnerHandlesDelivery(order:Pick<OperationalOrder,"channel"|"deliveryDispatchStatus">):boolean{
  return order.channel==="delivery"&&["searching","unassigned","accepted","picked_up","delivered"].includes(order.deliveryDispatchStatus??"");
}

export function canCancelOrder(order:CancellableOrder):boolean{
  if(order.completedAt||Number(order.paidAmount??0)>0.00001)return false;
  // El backend responde delivery_in_progress_not_cancellable; el delivery propio no tiene central y no cambia.
  if(deliveryInProgress(order))return false;
  if(!order.serviceItems?.length)return order.status==="nuevo"||order.status==="confirmado";
  if(!["nuevo","confirmado","preparando","listo"].includes(order.status))return false;
  return order.serviceItems.every(item=>item.status==="nuevo"||item.status==="confirmado"||(item.destination==="direct"&&item.status==="listo"));
}

export function canManageOrderService(order:{channel:string;waiterId?:string},userId?:string):boolean{
  return order.channel!=="salon"||!order.waiterId||Boolean(userId&&order.waiterId===userId);
}

/**
 * Comanda guardada sin enviar: un pedido «nuevo» o una ronda que el cliente pidió
 * por WhatsApp (ítems «Sin enviar»). Cocina no la ve hasta «Enviar comanda».
 */
export function hasUnsentItems(order:Pick<OperationalOrder,"status"|"serviceItems">):boolean{
  return order.status==="nuevo"||Boolean(order.serviceItems?.some(item=>item.status==="nuevo"));
}

export function nextOrderAction(order:OperationalOrder):OrderAction|null{
  if(order.completedAt)return null;
  // «confirmado» es enviar: la acción usa POST /orders/{id}/send (sendOrder).
  if(hasUnsentItems(order))return{status:"confirmado",label:"Enviar comanda",icon:"receipt"};
  if(order.channel==="salon"&&order.serviceItems?.some(item=>item.status!=="listo"&&item.status!=="entregado"))return null;
  if(partnerHandlesDelivery(order)&&(order.status==="listo"||order.status==="en_camino"))return null;
  if(order.status==="listo"){
    if(order.channel==="delivery")return{status:"en_camino",label:"En camino",icon:"truck"};
    if(order.channel==="salon")return{status:"entregado",label:"Confirmar entrega",icon:"check"};
    return awaitingPayment(order)?null:{status:"entregado",label:"Entregar",icon:"check"};
  }
  if(order.status==="en_camino")return awaitingPayment(order)?null:{status:"entregado",label:"Entregar",icon:"check"};
  return null;
}

/**
 * Fuera del salón primero se cobra y luego se entrega (2026-11-14): con saldo
 * pendiente, «Entregar» no aparece junto a «Cobrar saldo». El salón sigue su
 * flujo propio (se entrega, se cierra la cuenta y se cobra). El backend lo valida
 * con `payment_required_before_delivery`.
 */
function awaitingPayment(order:Pick<OperationalOrder,"remainingAmount">):boolean{
  return Number(order.remainingAmount??0)>0.00001;
}
