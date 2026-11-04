import {buildTicketPdf,layoutTicket,type TicketLayout} from "@/shared/ticket/ticket-80mm";
import {paymentTicketRows} from "../domain/payment-ticket";
import type {SaleDetail} from "../domain/types";

export type PaymentTicketLayout=TicketLayout;

export function layoutPaymentTicket(detail:SaleDetail,paymentId?:string,mode:"payment"|"prebill"="payment",withLogo=false):PaymentTicketLayout{
 return layoutTicket(paymentTicketRows(detail,paymentId,mode),detail.order.code,withLogo);
}

export function buildPaymentTicketPdf(layout:PaymentTicketLayout,logo?:Uint8Array|null){
 return buildTicketPdf(layout,{title:`Ticket de venta - ${layout.code}`,subject:"Ticket operativo no fiscal de 80 mm",logo});
}
