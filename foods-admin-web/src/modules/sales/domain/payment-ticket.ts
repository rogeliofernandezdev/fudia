import {formatRegionalDateTime,formatRegionalNumber} from "@/shared/i18n/regional-format";
import type {TicketRow} from "@/shared/ticket/ticket-80mm";
import type {SaleDetail} from "./types";

export type {TicketRow};

export function paymentTicketRows(detail:SaleDetail,paymentId?:string,mode:"payment"|"prebill"="payment"):TicketRow[]{
 const identity=detail.receiptContext;
 if(!identity)throw new Error("No pudimos cargar los datos del restaurante. Actualiza el servicio y reintenta.");
 if(paymentId&&!detail.payments.some(payment=>payment.id===paymentId))throw new Error("El cobro fue confirmado, pero aún no aparece en el detalle. Reintenta el ticket sin volver a cobrar.");
 const payments=detail.payments.filter(payment=>Number(payment.netAmount)>0);
 if(!payments.length&&mode==="payment")throw new Error("Este pedido no tiene cobros vigentes para generar un ticket de venta.");
 const number=(value:string|number)=>formatRegionalNumber(Number(value),identity.country,{minimumFractionDigits:identity.currencyDecimals,maximumFractionDigits:identity.currencyDecimals});
 const money=(value:string|number)=>identity.currencyPosition==="before"?`${identity.currencySymbol} ${number(value)}`:`${number(value)} ${identity.currencySymbol}`;
 const date=(value:string)=>formatRegionalDateTime(value,{country:identity.country,timeZone:identity.timezone},{dateStyle:"short"});
 const rows:TicketRow[]=[];
 const text=(value:string,options:Omit<TicketRow,"text">={})=>{if(value)rows.push({text:value,...options});};
 const rule=()=>rows.push({text:"",rule:true});
 // Mirrors the printed electronic document section by section: what the receipt or
 // invoice does not print, the ticket does not print either. Only the fiscal block
 // (tax breakdown, amount in words, QR, legends) is replaced by the non-fiscal notice.
 text(identity.organizationName,{bold:true,center:true,size:13});
 if(identity.legalName!==identity.organizationName)text(identity.legalName,{center:true});
 text(identity.taxId?`${identity.country==="PE"?"RUC":"Identificación fiscal"}: ${identity.taxId}`:"",{center:true});
 text(identity.address,{center:true});rule();
 text(mode==="prebill"?"PRECUENTA":"TICKET DE VENTA",{bold:true,center:true,size:11});text(detail.order.code,{bold:true,center:true,size:11});rule();
 // The date belongs to the actual latest payment, never the print request.
 const latest=[...payments].sort((a,b)=>Date.parse(b.createdAt)-Date.parse(a.createdAt))[0];
 text(`Fecha de emisión: ${date(latest?.createdAt??detail.order.createdAt)}`);
 text(detail.order.customerName?`Cliente: ${detail.order.customerName}`:"");rule();
 const quantity=(value:string|number)=>formatRegionalNumber(Number(value),identity.country,{maximumFractionDigits:3});
 if(!detail.order.items?.length)text("Sin detalle de productos registrado.");
 for(const item of detail.order.items??[])text(`${quantity(item.qty)} x ${item.name}`,{right:money(Number(item.qty)*Number(item.unitPrice))});
 // Delivery is one more line, as in the document breakdown.
 if(Number(detail.order.deliveryFee)>0)text(`${quantity(1)} x Delivery`,{right:money(detail.order.deliveryFee)});
 rule();
 text("TOTAL",{right:money(detail.order.total),bold:true,size:12});
 const policy=detail.tipPolicy,channel=detail.order.channel==="mostrador"?"recojo":detail.order.channel;
 if(mode==="prebill"&&!Number(detail.tip?.amount??0)&&policy&&policy.mode!=="disabled"&&policy.showSuggestion&&policy.channels.includes(channel)){
  const factor=10**identity.currencyDecimals;
  const suggested=Math.round(Math.max(0,Number(detail.order.total)-Number(detail.order.deliveryFee))*policy.percentage/100*factor)/factor;
  text(`Propina sugerida (${policy.percentage} %, voluntaria)`,{right:money(suggested)});
  text("Total si acepta la sugerencia",{right:money(Number(detail.order.total)+suggested)});
  text("Puede rechazarla o elegir otro monto. No es un cobro.",{size:8});
 }
 // A partial payment must never read as a paid account.
 const remaining=Number(detail.remainingAmount);
 if(mode==="payment"&&remaining>0){text("Pagado",{right:money(detail.paidAmount)});text("Saldo pendiente",{right:money(detail.remainingAmount),bold:true});}
 rule();
 text("No es un comprobante fiscal",{center:true,size:8});
 if(mode==="prebill")text("CUENTA POR PAGAR",{center:true,bold:true,size:9});
 else if(remaining>0)text("PAGO PARCIAL",{center:true,bold:true,size:9});
 text("Gestionado con Fudia",{center:true,size:8});
 return rows;
}
