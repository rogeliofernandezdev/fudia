import {pageRoutes} from "@/shared/routing/page-routes";
import type {StaffNotice} from "./types";

/** Pantallas donde se actúa sobre el aviso, en orden de preferencia; se usa la primera que el usuario puede abrir. */
export function noticeTargets(notice:Pick<StaffNotice,"kind"|"tableId">):string[]{
 switch(notice.kind){
  case "table_order":case "assistance_requested":return [pageRoutes.diningRoom,pageRoutes.orders];
  case "bill_requested":return [pageRoutes.pos,pageRoutes.diningRoom];
  case "order_ready":return notice.tableId?[pageRoutes.diningRoom,pageRoutes.orders]:[pageRoutes.orders,pageRoutes.pos];
  default:return [pageRoutes.pos,pageRoutes.orders];
 }
}

/**
 * Enlace final del aviso: Salón y Pedidos abren el detalle del pedido (`?orderId=`)
 * para atenderlo de inmediato (por ejemplo, «Confirmar entrega»). En POS solo la
 * cuenta pedida abre el cobro del pedido; los demás avisos llegan a la pantalla.
 */
export function noticeHref(target:string,notice:Pick<StaffNotice,"kind"|"orderId">){
 const opensDetail=target===pageRoutes.diningRoom||target===pageRoutes.orders||(target===pageRoutes.pos&&notice.kind==="bill_requested");
 return notice.orderId&&opensDetail?`${target}?orderId=${encodeURIComponent(notice.orderId)}`:target;
}

/** Tono del icono de cada aviso sin leer, alineado con la matriz de estados: listo = verde (Por entregar),
 *  cuenta pedida = naranja tenue (Por cobrar), llamado = celeste; pedidos nuevos usan el índigo de la marca. */
export type NoticeTone="primary"|"brand"|"peach"|"sky";
export function noticeTone(kind:StaffNotice["kind"]):NoticeTone{
 return kind==="order_ready"?"brand":kind==="bill_requested"?"peach":kind==="assistance_requested"?"sky":"primary";
}

/** Avisos que llegaron después de la última lectura del feed y siguen sin leer, del más reciente al más antiguo. */
export function freshNotices(items:readonly StaffNotice[],seen:ReadonlySet<string>){
 return items.filter(item=>!item.read&&!seen.has(item.id));
}

/** Antigüedad corta y legible: «Ahora», «Hace 5 min», «Hace 2 h». */
export function noticeAge(createdAt:string,now=Date.now()){
 const minutes=Math.floor((now-new Date(createdAt).getTime())/60000);
 if(!Number.isFinite(minutes)||minutes<1)return "Ahora";
 if(minutes<60)return `Hace ${minutes} min`;
 return `Hace ${Math.floor(minutes/60)} h`;
}

/** El contador no crece sin límite en la campana. */
export function unreadLabel(unread:number){return unread>99?"99+":String(unread)}
