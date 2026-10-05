type OperationalOrder={channel:string;status:string;paymentStatus?:string;completedAt?:string};
type OrderAction={status:string;label:string;icon:"receipt"|"truck"|"check"};

export function nextOrderAction(order:OperationalOrder):OrderAction|null{
  if(order.status==="nuevo")return{status:"confirmado",label:"Enviar a cocina",icon:"receipt"};
  if(order.status==="listo"){
    if(order.channel==="delivery")return{status:"en_camino",label:"En camino",icon:"truck"};
    return{status:"entregado",label:order.channel==="salon"?"Marcar como entregado":"Entregar",icon:"check"};
  }
  if(order.status==="en_camino")return{status:"entregado",label:"Entregar",icon:"check"};
  if(order.channel==="salon"&&order.status==="entregado"&&!order.completedAt&&order.paymentStatus==="paid"){
    return{status:"entregado",label:"Liberar mesa",icon:"check"};
  }
  return null;
}
