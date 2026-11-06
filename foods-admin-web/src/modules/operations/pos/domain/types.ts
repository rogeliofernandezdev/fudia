/** Código de un medio del catálogo de la empresa (Configuración › Medios de pago). */
import type {TipPolicy,TipReceipt} from "../../tips";
import type {FiscalDocumentRef,FiscalRequest} from "@/modules/einvoicing";
import type {ServiceItem} from "../../orders";
export type PaymentMethod=string;
/** Medio activo y habilitado para ventas, en el orden del catálogo. */
export type PaymentMethodOption={code:string;name:string;affectsCash:boolean};
export type PaymentStatus="pending"|"partial"|"paid";

export type POSOrderSummary={
 deliveryFee?:string;
  waiterId?:string;
  id:string;
  code:string;
  channel:string;
  /** "counter" en la venta por mostrador. */
  origin?:string;
  /** Productos resumidos por el backend («2 × Ceviche, 1 × Chicha»). */
  itemsText?:string;
  status:string;
  completedAt?:string;
  billClosedAt?:string;
  billClosed?:boolean;
  customerName:string;
  tableName:string;
  total:string;
  paidAmount:string;
  remainingAmount:string;
  paymentStatus:PaymentStatus;
  createdAt:string;
};

export type POSOrderItem={
  id:string;
  productId:string;
  name:string;
  qty:string;
  unitPrice:string;
  note:string;
  itemType?:"product"|"combo";
  selections?:{groupName:string;name:string;surcharge?:string}[];
  modifiers?:{groupName:string;name:string}[];
};

export type POSOrder={
  waiterId?:string;
  waiterName?:string;
  id:string;
  code:string;
  channel:string;
  origin?:string;
  status:string;
  completedAt?:string;
  billClosedAt?:string;
  billClosed?:boolean;
  customerName:string;
  tableName:string;
  notes:string;
  total:string;
  createdAt:string;
  items?:POSOrderItem[];
  /** Estado de preparación de cada línea, el mismo que muestra el detalle de pedido. */
  serviceItems?:ServiceItem[];
};

export type Payment={
  /** Present on the payment that requested an electronic document. */
  fiscalDocument?:FiscalDocumentRef;
  id:string;
  orderId:string;
  orderCode:string;
  shiftId:string;
  cashRegisterName:string;
  method:PaymentMethod;
  methodName:string;
  affectsCash:boolean;
  amount:string;
  refundedAmount:string;
  netAmount:string;
  reference:string;
  createdByName:string;
  createdAt:string;
};

export type POSOrderDetail={
 tipPolicy?:TipPolicy;
 tip?:TipReceipt;
  order:POSOrder;
  paidAmount:string;
  remainingAmount:string;
  paymentStatus:PaymentStatus;
  payments:Payment[];
  /** Electronic document delivered for the order; absent when charged without one. */
  fiscalDocument?:FiscalDocumentRef;
};

export type POSOrdersResponse={
  items:POSOrderSummary[];
  total:number;
  page:number;
  pageSize:number;
};

export type POSCollectionShift={id:string;cashRegisterName:string;openedByName:string};
export type POSCollectionAccess={shift:POSCollectionShift|null;assignedShift?:POSCollectionShift|null;shifts?:POSCollectionShift[];canChooseShift?:boolean};

export type PaymentDraft={
 shiftId?:string;
 /** Electronic document for the payment that settles the account. */
 fiscal?:FiscalRequest;
 tipEnabled?:boolean;
 tipKind?:"percentage"|"fixed";
 /** Monto libre; el porcentaje no se escribe, viene de la configuración. */
 tipValue?:string;
  method:PaymentMethod;
  amount:string;
  reference:string;
  /** Local cash tender, used to calculate change; never sent as payment revenue. */
  receivedAmount?:string;
};

export type RefundDraft={
  amount:string;
  reason:string;
  note:string;
};
