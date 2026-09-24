export type CashShift={
  id:string; code:string; cashRegisterId:string; cashRegisterName:string; blindClose:boolean; status:"open"|"closed";
  openingAmount:string; incomeAmount:string; expenseAmount:string; expectedAmount:string; expectedVisible:boolean;
  openedByName:string; openedAt:string; businessDate:string;
};
export type POSOrderItem={id:string;name:string;qty:string;unitPrice:string;note:string};
export type Payment={id:string;method:string;amount:string;reference:string;createdAt:string;createdByName:string;cashRegisterName:string;refundedAmount:string;netAmount:string};
export type PaymentMethod={code:string;name:string;description:string;active:boolean;salesEnabled:boolean;expensesEnabled:boolean;affectsCash:boolean;sortOrder:number};
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
  let data:unknown={};
  if(text){try{data=JSON.parse(text)}catch{data={message:text}}}
  const payload=typeof data==="object"&&data!==null?data as Record<string,unknown>:{};
  const message=typeof payload.message==="string"?payload.message:"No se pudo completar la operación.";
  const code=typeof payload.code==="string"?payload.code:"request_failed";
  if(!response.ok)throw new OperationsApiError(message,response.status,code);
  return data as T;
}
