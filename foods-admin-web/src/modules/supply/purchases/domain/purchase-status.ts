import type {PurchaseStatus} from "./types";

/**
 * Mapeo central de estados de la orden de compra a la matriz de `Status`
 * (docs/02_DESIGN_SYSTEM.md › Compras). Ámbar = resolución pendiente; rojo = cancelación.
 */
export const purchaseStatusMeta:Record<PurchaseStatus,{label:string;tone:"green"|"blue"|"orange"|"gray"|"red"}>={
  draft:{label:"Borrador",tone:"gray"},
  pending_approval:{label:"Por aprobar",tone:"orange"},
  approved:{label:"Aprobada",tone:"blue"},
  partially_received:{label:"Recepción parcial",tone:"orange"},
  received:{label:"Recibida",tone:"green"},
  cancelled:{label:"Cancelada",tone:"red"},
};

/** «Revisar» solo se ofrece cuando el usuario tiene una acción real en ese estado. */
export function canReviewPurchase(status:PurchaseStatus,access:{manage:boolean;approve:boolean;receive:boolean}){
  if(access.manage&&(status==="draft"||status==="pending_approval"||status==="approved"))return true;
  if(access.approve&&status==="pending_approval")return true;
  return access.receive&&(status==="approved"||status==="partially_received");
}
