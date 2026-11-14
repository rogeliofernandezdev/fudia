/** Delivery Fudia: enlace/QR de cada local y despacho del pedido a las centrales (globales). */
/** Enlace/QR de delivery del local; token vacío si aún no se generó. */
export type DeliveryLink={locationId:string;token:string;active:boolean;updatedAt?:string};

export type DispatchStatus="searching"|"accepted"|"unassigned"|"picked_up"|"delivered"|"own"|"cancelled";
export type DispatchOffer={partnerName:string;round:number;priority:number;status:"pending"|"accepted"|"rejected"|"expired"|"withdrawn";createdAt:string;expiresAt?:string;respondedAt?:string};
export type DeliveryDispatch={orderId:string;status:DispatchStatus;round:number;partnerName?:string;acceptedAt?:string;pickedUpAt?:string;deliveredAt?:string;offers:DispatchOffer[]};

/** Alerta de recogida que calcula el backend: 3, 5 y 8 minutos con el pedido listo sin recoger. */
export type PickupAlert=""|"warning"|"urgent"|"critical";

/** Delivery del local en el tablero de Delivery › Pedidos. `dispatchStatus` vacío: sin centrales (delivery propio). */
export type DeliveryBoardOrder={
  id:string;
  code:string;
  status:string;
  customerName:string;
  customerPhone:string;
  address:string;
  total:string;
  currencySymbol:string;
  createdAt:string;
  dispatchStatus:""|DispatchStatus;
  partnerName:string;
  partnerPhone:string;
  courierName:string;
  pickupWaitSeconds:number;
  pickupAlert:PickupAlert;
  pickupAlertAcknowledged:boolean;
  /** El motorizado marcó «No pude entregar». */
  issueReportedAt?:string;
  /** La central registró la entrega fallida. */
  failedDelivery:boolean;
  acceptedAt?:string;
  pickedUpAt?:string;
  deliveredAt?:string;
};

/** Centrales activas que cubren el distrito del local (solo lectura para el restaurante). */
export type DeliveryProviders={enabled:boolean;districtName:string;items:{name:string;whatsappPhone:string}[]};

/** Entrega fallida registrada por la central para un pedido del local. */
export type LocationFailedDelivery={
  orderCode:string;
  partnerName:string;
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
