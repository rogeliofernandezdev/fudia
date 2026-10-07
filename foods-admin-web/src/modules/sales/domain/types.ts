export type Sale={
  id:string;code:string;channel:string;status:string;customerName:string;tableName:string;
  total:string;paidAmount:string;remainingAmount:string;paymentStatus:"pending"|"partial"|"paid";createdAt:string;
};
export type SalesResponse={items:Sale[];total:number;page:number;pageSize:number};
export type SaleItem={
  id:string;name:string;qty:string;unitPrice:string;note:string;
  selections?:{groupName:string;name:string}[];
  modifiers?:{groupName:string;name:string}[];
};
export type SalePayment={
  id:string;method:string;methodName?:string;amount:string;refundedAmount:string;netAmount:string;
  reference:string;cashRegisterName:string;createdByName:string;createdAt:string;
};
export type SaleDetail={
  order:Pick<Sale,"id"|"code"|"channel"|"status"|"customerName"|"tableName"|"total"|"createdAt">&{subtotal:string;deliveryFee:string;notes:string;items?:SaleItem[]};
  paidAmount:string;remainingAmount:string;paymentStatus:"pending"|"partial"|"paid";
  payments:SalePayment[];
};
