import {apiFetch} from "@/shared/api/client";
import type {Payment,PaymentDraft,POSOrderDetail,POSOrdersResponse,RefundDraft} from "../domain/types";

export function listPOSOrders(input:{q:string;paymentStatus:string;page:number;pageSize:number}){
  const params=new URLSearchParams({
    q:input.q,
    paymentStatus:input.paymentStatus,
    page:String(input.page),
    pageSize:String(input.pageSize),
  });
  return apiFetch<POSOrdersResponse>(`pos/orders?${params.toString()}`);
}

export function getPOSOrder(id:string){
  return apiFetch<POSOrderDetail>(`pos/orders/${id}`);
}

export function createPayment(orderId:string,draft:PaymentDraft){
  return apiFetch<Payment>("payments",{
    method:"POST",
    body:JSON.stringify({
      orderId,
      method:draft.method,
      amount:Number(draft.amount),
      reference:draft.reference.trim(),
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
