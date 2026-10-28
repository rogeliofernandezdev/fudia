/** Código de un medio del catálogo de la empresa (Configuración › Medios de pago). */
export type PaymentMethod=string;
/** Medio activo y habilitado para ventas, en el orden del catálogo. */
export type PaymentMethodOption={code:string;name:string;affectsCash:boolean};
export type PaymentStatus="pending"|"partial"|"paid";

export type POSOrderSummary={
  waiterId?:string;
  id:string;
  code:string;
  channel:string;
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
};

export type Payment={
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
  order:POSOrder;
  paidAmount:string;
  remainingAmount:string;
  paymentStatus:PaymentStatus;
  payments:Payment[];
};

export type POSOrdersResponse={
  items:POSOrderSummary[];
  total:number;
  page:number;
  pageSize:number;
};

export type PaymentDraft={
  method:PaymentMethod;
  amount:string;
  reference:string;
};

export type RefundDraft={
  amount:string;
  reason:string;
  note:string;
};
