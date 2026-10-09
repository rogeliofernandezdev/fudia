export type Sale={
  id:string;code:string;channel:string;status:string;customerName:string;tableName:string;
  total:string;paidAmount:string;remainingAmount:string;paymentStatus:"pending"|"partial"|"paid";paymentMethods:string[];createdAt:string;
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
  receiptContext?:PaymentReceiptContext;
  order:Pick<Sale,"id"|"code"|"channel"|"status"|"customerName"|"tableName"|"total"|"createdAt">&{waiterName?:string;subtotal:string;deliveryFee:string;notes:string;items?:SaleItem[]};
  paidAmount:string;remainingAmount:string;paymentStatus:"pending"|"partial"|"paid";
  payments:SalePayment[];
};
export type PaymentReceiptContext={organizationName:string;legalName:string;taxId:string;locationName:string;address:string;phone:string;country:string;timezone:string;currency:string;currencySymbol:string;currencyPosition:"before"|"after";currencyDecimals:number};
