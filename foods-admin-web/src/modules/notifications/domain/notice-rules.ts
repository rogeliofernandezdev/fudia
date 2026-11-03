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

/** Antigüedad corta y legible: «Ahora», «Hace 5 min», «Hace 2 h». */
export function noticeAge(createdAt:string,now=Date.now()){
 const minutes=Math.floor((now-new Date(createdAt).getTime())/60000);
 if(!Number.isFinite(minutes)||minutes<1)return "Ahora";
 if(minutes<60)return `Hace ${minutes} min`;
 return `Hace ${Math.floor(minutes/60)} h`;
}

/** El contador no crece sin límite en la campana. */
export function unreadLabel(unread:number){return unread>99?"99+":String(unread)}
