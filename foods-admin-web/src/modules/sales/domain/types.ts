export type Sale={
  id:string;code:string;channel:string;status:string;customerName:string;tableName:string;
  total:string;paidAmount:string;remainingAmount:string;paymentStatus:"pending"|"partial"|"paid";createdAt:string;
};
export type SalesResponse={items:Sale[];total:number;page:number;pageSize:number};
