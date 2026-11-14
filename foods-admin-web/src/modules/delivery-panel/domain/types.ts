/** Panel web del proveedor de Delivery Fudia (central de reparto). */
export type PanelStage="offer"|"accepted"|"picked_up"|"delivered";
export type PanelAction="accept"|"reject"|"pickup"|"deliver";
export type PanelOutcome="accepted"|"already_accepted"|"rejected"|"taken"|"expired"|"cancelled"|"not_ready"|"picked_up"|"delivered"|"not_picked_up"|"not_found"|"payment_conflict";

export type PanelOrderItem={quantity:number;name:string;note?:string};
export type PanelOrder={
  /** Identifica la oferta en las acciones del panel. */
  token:string;
  stage:PanelStage;
  orderCode:string;
  /** Estado del pedido en el local: `listo` permite registrar la recogida. */
  orderStatus:string;
  organizationName:string;
  locationName:string;
  locationAddress:string;
  locationLatitude:number|null;
  locationLongitude:number|null;
  customerName:string;
  customerPhone:string;
  address:string;
  reference:string;
  latitude:number|null;
  longitude:number|null;
  distanceKm:number|null;
  notes:string;
  items:PanelOrderItem[];
  currencySymbol:string;
  /** Lo que la central paga al local al recoger. */
  subtotal:string;
  deliveryFee:string;
  /** Lo que la central cobra al cliente al entregar. */
  total:string;
  expiresInSeconds:number;
  createdAt:string;
  acceptedAt?:string;
  pickedUpAt?:string;
  deliveredAt?:string;
};
export type PartnerPanel={partnerName:string;active:boolean;orders:PanelOrder[]};
export type PanelActionResult={outcome:PanelOutcome;orderCode?:string};
export type CourierPosition={latitude:number;longitude:number};
