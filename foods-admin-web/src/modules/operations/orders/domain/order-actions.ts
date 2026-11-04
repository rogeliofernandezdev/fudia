type OperationalOrder={channel:string;status:string;paymentStatus?:string;completedAt?:string;serviceItems?:{status:string}[]};
type OrderAction={status:string;label:string;icon:"receipt"|"truck"|"check"};
type CancellableOrder={status:string;completedAt?:string;paidAmount?:string;serviceItems?:{destination:string;status:string}[]};

export function canCancelOrder(order:CancellableOrder):boolean{
  if(order.completedAt||Number(order.paidAmount??0)>0.00001)return false;
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
  if(order.status==="listo"){
    if(order.channel==="delivery")return{status:"en_camino",label:"En camino",icon:"truck"};
    return{status:"entregado",label:order.channel==="salon"?"Confirmar entrega":"Entregar",icon:"check"};
  }
  if(order.status==="en_camino")return{status:"entregado",label:"Entregar",icon:"check"};
  return null;
}
