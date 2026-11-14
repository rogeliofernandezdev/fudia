/** Panel web del proveedor de Delivery Fudia (central de reparto) y enlace del motorizado. */
export type PanelStage="offer"|"accepted"|"picked_up"|"delivered";
/** Acciones de la central: responde la oferta, libera el reparto, mantiene ante una alerta o registra la entrega fallida. */
export type PanelAction="accept"|"reject"|"release"|"keep"|"fail";
/** Acciones del motorizado desde su enlace. */
export type CourierAction="pickup"|"deliver"|"not_delivered";
export type PanelOutcome="accepted"|"already_accepted"|"rejected"|"taken"|"expired"|"cancelled"|"not_ready"|"picked_up"|"delivered"|"not_picked_up"|"not_found"|"payment_conflict"|"payment_pending"
  |"released"|"kept"|"delivery_failed"|"already_picked_up"|"not_reported"|"too_soon"|"courier_assigned"|"issue_reported";
/** Alerta de recogida: 3 min aviso, 5 min urgente, 8 min crítica (también al restaurante). */
export type PickupAlert=""|"warning"|"urgent"|"critical";

export type PanelOrderItem={quantity:number;name:string;note?:string};
export type PanelOrder={
  /** Identifica la oferta en las acciones del panel (vacío en el enlace del motorizado). */
  token:string;
  stage:PanelStage;
  orderCode:string;
  /** La recogida exige el pedido listo y el pago completo confirmado por caja. */
  orderStatus:string;
  paymentConfirmed:boolean;
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
  /** Motorizado responsable; vacío hasta que la central lo registra. */
  courierName:string;
  /** Enlace privado del motorizado (32 hex); vacío sin motorizado y siempre vacío en su propio enlace. */
  courierToken:string;
  /** Segundos esperando la recogida con el pedido listo (0 fuera de esa situación). */
  pickupWaitSeconds:number;
  pickupAlert:PickupAlert;
  /** La central eligió «Mantener» en el nivel actual de la alerta. */
  pickupAlertAcknowledged:boolean;
  /** El motorizado marcó «No pude entregar». */
  issueReportedAt?:string;
  /** Segundos que faltan para registrar la entrega fallida (0 = ya puede). */
  failAvailableInSeconds:number;
};
/** Entrega fallida registrada: Fudia conserva datos y montos para el reclamo. */
export type FailedDelivery={
  orderCode:string;
  customerName:string;
  address:string;
  courierName:string;
  currencySymbol:string;
  foodAmount:string;
  courierReimbursement:string;
  partnerShare:string;
  fudiaShare:string;
  fudiaShareEligible:boolean;
  validationNotes:string[];
  registeredAt:string;
};
export type PartnerPanel={partnerName:string;active:boolean;orders:PanelOrder[];failedDeliveries?:FailedDelivery[]};
export type PanelActionResult={outcome:PanelOutcome;orderCode?:string};
export type CourierAssignment={outcome:PanelOutcome;courierName?:string;courierToken?:string};
export type CourierView={partnerName:string;courierName:string;order:PanelOrder};
export type CourierPosition={latitude:number;longitude:number};
