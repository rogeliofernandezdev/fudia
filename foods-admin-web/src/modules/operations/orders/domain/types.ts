import type {ServiceItem} from "../../orders/domain/service-flow";
export type Option={value:string;label:string};

export type OrderItemSelection={
  groupId:string;
  groupName:string;
  productId:string;
  name:string;
  surcharge:string;
};

export type OrderItem={
  id:string;
  productId:string;
  name:string;
  qty:string;
  unitPrice:string;
  note:string;
  itemType?:"product"|"combo";
  selections?:OrderItemSelection[];
  modifiers?:{groupId:string;groupName:string;optionId:string;name:string;surcharge:string}[];
};

export type Order={
  id:string;
  code:string;
  channel:string;
  /** "counter" en la venta por mostrador (se atiende como Recojo en local). */
  origin?:"standard"|"counter";
  status:string;completedAt?:string;billClosedAt?:string;accountState?:"open"|"awaiting_payment"|"paid";serviceItems?:ServiceItem[];
  customerId:string;
  customerName:string;
  customerPhone:string;
  address:string;
  reference:string;
  deliveryLatitude?:number|null;
  deliveryLongitude?:number|null;
  deliveryDistanceKm?:number|null;
  /** Despacho a las centrales de Delivery Fudia; ausente si el pedido no usa centrales. */
  deliveryDispatchStatus?:string;
  deliveryPartnerName?:string;
  /** Envío que cobra la central al cliente; ya no forma parte del total del local. */
  deliveryPartnerFee?:string;
  tableId:string;
  tableName:string;
  waiterId?:string;
  waiterName?:string;
  collectedByNames?:string[];
  notes:string;
  subtotal:string;
  deliveryFee:string;
  total:string;
  createdAt:string;
  updatedAt:string;
  items?:OrderItem[];
  paidAmount?:string;
  remainingAmount?:string;
  paymentStatus?:"pending"|"partial"|"paid";
  /** User-entered operation numbers of recorded payments; detail only. */
  paymentReferences?:{methodName:string;reference:string}[];
};

export type OrdersResponse={
  items:Order[];
  total:number;
  channelCounts:Record<string,number>;
  channelOptions:Option[];
  /** Tipos sin mesa del alta manual: Mostrador, Recojo en local y Delivery. */
  manualChannelOptions?:Option[];
  statusOptions:Option[];
};

/** Costo de envío sugerido por las tarifas del local para un punto del mapa. */
export type DeliveryQuote={covered:boolean;distanceKm:string;fee:string;maxDistanceKm:string;currencySymbol:string};
