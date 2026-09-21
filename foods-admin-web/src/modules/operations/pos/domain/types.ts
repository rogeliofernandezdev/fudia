export type PaymentMethod="cash"|"card"|"transfer"|"other";
export type PaymentStatus="pending"|"partial"|"paid";

export type POSOrderSummary={
  id:string;
  code:string;
  channel:string;
  status:string;
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
};

export type POSOrder={
  id:string;
  code:string;
  channel:string;
  status:string;
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
