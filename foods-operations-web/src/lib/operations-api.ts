export type CashShift={
  id:string; code:string; cashRegisterId:string; cashRegisterName:string; blindClose:boolean; status:"open"|"closed";
  openingAmount:string; incomeAmount:string; expenseAmount:string; expectedAmount:string; expectedVisible:boolean;
  openedByName:string; openedAt:string; businessDate:string;
};
export type POSOrderItem={id:string;name:string;qty:string;unitPrice:string;note:string};
export type Payment={id:string;method:"cash"|"card"|"transfer"|"other";amount:string;reference:string;createdAt:string;createdByName:string;cashRegisterName:string;refundedAmount:string;netAmount:string};
export type POSOrder={id:string;code:string;channel:string;status:string;customerName:string;tableName:string;total:string;createdAt:string;items?:POSOrderItem[]};
export type POSOrderDetail={order:POSOrder;paidAmount:string;remainingAmount:string;paymentStatus:"pending"|"partial"|"paid";payments:Payment[]};
export type POSOrderSummary={id:string;code:string;channel:string;status:string;customerName:string;tableName:string;total:string;paidAmount:string;remainingAmount:string;paymentStatus:"pending"|"partial"|"paid";createdAt:string};

export class OperationsApiError extends Error{
  status:number;
  code:string;
  constructor(message:string,status:number,code="request_failed"){super(message);this.name="OperationsApiError";this.status=status;this.code=code;}
}

export async function operationsFetch<T>(path:string,init?:RequestInit):Promise<T>{
  const response=await fetch(`/api/operations/${path}`,{...init,headers:{Accept:"application/json",...(init?.body?{"Content-Type":"application/json"}:{}),...(init?.headers??{})},cache:"no-store"});
  if(response.status===204)return undefined as T;
  const text=await response.text();
  let data:any={};
  if(text){try{data=JSON.parse(text)}catch{data={message:text}}}
  if(!response.ok)throw new OperationsApiError(data?.message??"No se pudo completar la operación.",response.status,data?.code);
  return data as T;
}
