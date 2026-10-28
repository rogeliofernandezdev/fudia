import {formatRegionalDateTime,formatRegionalNumber} from "@/shared/i18n/regional-format";
import type {SaleDetail} from "./types";

export type TicketRow={text:string;right?:string;bold?:boolean;center?:boolean;size?:number;rule?:boolean};

export function paymentTicketRows(detail:SaleDetail,paymentId?:string,mode:"payment"|"prebill"="payment"):TicketRow[]{
 const identity=detail.receiptContext;
 if(!identity)throw new Error("No pudimos cargar los datos del restaurante. Actualiza el servicio y reintenta.");
 if(paymentId&&!detail.payments.some(payment=>payment.id===paymentId))throw new Error("El cobro fue confirmado, pero aún no aparece en el detalle. Reintenta el ticket sin volver a cobrar.");
 const payments=detail.payments.filter(payment=>Number(payment.netAmount)>0);
 if(!payments.length&&mode==="payment")throw new Error("Este pedido no tiene cobros vigentes para generar un ticket de pago.");
 const number=(value:string|number)=>formatRegionalNumber(Number(value),identity.country,{minimumFractionDigits:identity.currencyDecimals,maximumFractionDigits:identity.currencyDecimals});
 const money=(value:string|number)=>identity.currencyPosition==="before"?`${identity.currencySymbol} ${number(value)}`:`${number(value)} ${identity.currencySymbol}`;
 const date=(value:string)=>formatRegionalDateTime(value,{country:identity.country,timeZone:identity.timezone},{dateStyle:"short",timeStyle:"short"});
 const rows:TicketRow[]=[];
 const text=(value:string,options:Omit<TicketRow,"text">={})=>{if(value)rows.push({text:value,...options});};
 const rule=()=>rows.push({text:"",rule:true});
 text(identity.organizationName,{bold:true,center:true,size:13});
 if(identity.legalName!==identity.organizationName)text(identity.legalName,{center:true});
 text(identity.taxId?`${identity.country==="PE"?"RUC":"Identificación fiscal"}: ${identity.taxId}`:"",{center:true});
 text(identity.locationName,{center:true});text(identity.address,{center:true});text(identity.phone?`Tel.: ${identity.phone}`:"",{center:true});rule();
 text(mode==="prebill"?"PRECUENTA":"TICKET DE PAGO",{bold:true,center:true,size:11});text(detail.order.code,{center:true});
 // The timestamp belongs to the actual latest payment, never the print request.
 const latest=[...payments].sort((a,b)=>Date.parse(b.createdAt)-Date.parse(a.createdAt))[0];
 text(`Fecha: ${date(latest?.createdAt??detail.order.createdAt)}`);
 text(detail.order.tableName?`Mesa: ${detail.order.tableName}`:"");text(detail.order.customerName?`Cliente: ${detail.order.customerName}`:"");
 text(detail.order.waiterName?`Mozo: ${detail.order.waiterName}`:"");rule();
 text("CONSUMO",{bold:true});
 if(!detail.order.items?.length)text("Sin detalle de productos registrado.");
 for(const item of detail.order.items??[]){
  text(`${formatRegionalNumber(Number(item.qty),identity.country,{maximumFractionDigits:3})} x ${item.name}`,{bold:true});
  text(`P. unit. ${money(item.unitPrice)}`,{right:money(Number(item.qty)*Number(item.unitPrice))});
  for(const selection of item.selections??[])text(`  ${selection.groupName}: ${selection.name}`,{size:8});
  for(const modifier of item.modifiers??[])text(`  ${modifier.groupName}: ${modifier.name}`,{size:8});
 }
 rule();text("Subtotal",{right:money(detail.order.subtotal)});
 if(Number(detail.order.deliveryFee)>0)text("Delivery",{right:money(detail.order.deliveryFee)});
 text("TOTAL",{right:money(detail.order.total),bold:true,size:12});
 const tip=Number(detail.tip?.amount??0);
 if(tip>0){
  text(detail.tip?.directToStaff?"Propina directa al personal":"Propina aceptada",{right:money(tip)});
  if(!detail.tip?.directToStaff)text("Consumo y propina",{right:money(Number(detail.order.total)+tip),bold:true});
 }
 const policy=detail.tipPolicy,channel=detail.order.channel==="mostrador"?"recojo":detail.order.channel;
 if(mode==="prebill"&&!tip&&policy&&policy.mode!=="disabled"&&policy.showSuggestion&&policy.channels.includes(channel)){
  const factor=10**identity.currencyDecimals;
  const suggested=Math.round(Math.max(0,Number(detail.order.total)-Number(detail.order.deliveryFee))*policy.percentage/100*factor)/factor;
  text(`Propina sugerida (${policy.percentage} %, voluntaria)`,{right:money(suggested)});
  text("Total si acepta la sugerencia",{right:money(Number(detail.order.total)+suggested)});
  text("Puede rechazarla o elegir otro monto. No es un cobro.",{size:8});
 }
 text("Pagado",{right:money(detail.paidAmount)});
 if(Number(detail.remainingAmount)>0)text("Saldo pendiente",{right:money(detail.remainingAmount),bold:true});
 if(payments.length){rule();text("PAGOS REGISTRADOS",{bold:true});}
 for(const payment of payments){
  text(payment.methodName||payment.method,{right:money(payment.netAmount),bold:true});
  if(detail.tip?.paymentId===payment.id&&!detail.tip.directToStaff){text("Propina incluida en este cobro",{right:money(tip)});text("Total recibido por este medio",{right:money(Number(payment.netAmount)+tip),bold:true});}
  text(date(payment.createdAt),{size:8});text(payment.createdByName?`Cobrado por: ${payment.createdByName}`:"",{size:8});
  text(payment.reference?`Referencia: ${payment.reference}`:"",{size:8});
  if(Number(payment.refundedAmount)>0)text(`Pago ajustado: ${money(payment.amount)} - ${money(payment.refundedAmount)}`,{size:8});
 }
 rule();text(mode==="prebill"?"CUENTA POR PAGAR":Number(detail.remainingAmount)>0?"PAGO PARCIAL":"CUENTA PAGADA",{center:true,bold:true});
 text("Gracias por su visita",{center:true});text("No es un comprobante fiscal",{center:true,size:8});text("Gestionado con Fudia",{center:true,size:8});
 return rows;
}
