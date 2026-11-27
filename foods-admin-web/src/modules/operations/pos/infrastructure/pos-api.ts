import {apiFetch} from "@/shared/api/client";
import type {POSCollectionAccess,Payment,PaymentDraft,PaymentMethodOption,POSOrderDetail,POSOrdersResponse,RefundDraft} from "../domain/types";

export type POSCollectionShift={id:string;cashRegisterName:string;openedByName:string};
export function getPOSCollectionAccess(){return apiFetch<POSCollectionAccess>("pos/access");}

export function listPOSOrders(input:{q:string;paymentStatus:string;page:number;pageSize:number},signal?:AbortSignal){
  const params=new URLSearchParams({
    q:input.q,
    paymentStatus:input.paymentStatus,
    page:String(input.page),
    pageSize:String(input.pageSize),
    // Cobros lista solo lo listo para cobrar: mesa con la cuenta cerrada, recojo o delivery
    // con la comida lista y mostrador siempre (regla del backend `ready`, 2026-11-25).
    ready:"true",
  });
  return apiFetch<POSOrdersResponse>(`pos/orders?${params.toString()}`,{signal});
}

export function getPOSOrder(id:string){
  return apiFetch<POSOrderDetail>(`pos/orders/${id}`);
}

/** Medios activos y habilitados para ventas del catálogo de la empresa. */
export function listSalesPaymentMethods(){
  return apiFetch<{items:PaymentMethodOption[]}>("payment-methods/operational?usage=sales");
}

export function createPayment(orderId:string,draft:PaymentDraft){
  return apiFetch<Payment>("payments",{
    method:"POST",
    body:JSON.stringify({
      orderId,
      ...(draft.shiftId?{shiftId:draft.shiftId}:{}),
      method:draft.method,
      amount:Number(draft.amount),
      reference:draft.reference.trim(),
      ...(draft.depositPaymentId?{depositPaymentId:draft.depositPaymentId}:{}),
      ...(draft.fiscal?{fiscal:draft.fiscal}:{}),
      ...(draft.tipEnabled?{tip:draft.tipKind==="fixed"?{kind:"fixed",value:Number(draft.tipValue)}:{kind:"percentage"}}:{}),
    }),
  });
}

export function refundPayment(paymentId:string,draft:RefundDraft){
  return apiFetch<void>(`payments/${paymentId}/refund`,{
    method:"POST",
    body:JSON.stringify({
      amount:Number(draft.amount),
      reason:draft.reason.trim(),
      note:draft.note.trim(),
    }),
  });
}
