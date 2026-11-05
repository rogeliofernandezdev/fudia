import type {KitchenResponse,KitchenStatus,KitchenTicket} from "./types";

/** Estados que un plato puede recibir desde Cocina: listo o de vuelta a preparación. */
export type PlateStatus=Extract<KitchenStatus,"preparando"|"listo">;

/** Plato con estado propio en su estación; las comandas históricas no lo traen. */
export function isTrackedPlate(item:KitchenTicket["items"][number]){
  return item.status==="confirmado"||item.status==="preparando"||item.status==="listo";
}

/** Progreso por línea de la comanda («2 de 4 listos»); null si la comanda no lleva platos con estado. */
export function plateProgress(ticket:KitchenTicket){
  const tracked=ticket.items.filter(isTrackedPlate);
  if(!tracked.length)return null;
  return {ready:tracked.filter(item=>item.status==="listo").length,total:tracked.length};
}

/** Misma derivación que el backend: algún plato preparando, si no alguno confirmado, si no listo. */
export function ticketStatusFromPlates(ticket:KitchenTicket):KitchenStatus{
  const tracked=ticket.items.filter(isTrackedPlate);
  if(!tracked.length)return ticket.status;
  if(tracked.some(item=>item.status==="preparando"))return "preparando";
  if(tracked.some(item=>item.status==="confirmado"))return "confirmado";
  return "listo";
}

/** Aplica el cambio de un plato a la cola (actualización optimista o su reversión). */
export function applyPlateStatus(data:KitchenResponse,ticketId:string,itemId:string,status:KitchenStatus):KitchenResponse{
  const counts={...data.counts};
  const items=data.items.map(ticket=>{
    if(ticket.id!==ticketId)return ticket;
    const next={...ticket,items:ticket.items.map(item=>item.id===itemId?{...item,status}:item)};
    next.status=ticketStatusFromPlates(next);
    if(next.status!==ticket.status){
      counts[ticket.status]=Math.max(0,(counts[ticket.status]??0)-1);
      counts[next.status]=(counts[next.status]??0)+1;
    }
    return next;
  });
  return {...data,items,counts};
}
