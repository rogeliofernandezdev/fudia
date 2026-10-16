import {formatRegionalCalendarDate,formatRegionalDateTime,formatRegionalNumber} from "@/shared/i18n/regional-format";
import type {CashShiftReport} from "./types";

export function cashReportFormat(report:CashShiftReport){
 return {
  money:(value:string|number|null|undefined)=>{
   if(value===null||value===undefined||value==="")return "—";
   const amount=Number(value);if(!Number.isFinite(amount))return "—";
   const formatted=formatRegionalNumber(Math.abs(amount),report.country,{minimumFractionDigits:report.currencyDecimals,maximumFractionDigits:report.currencyDecimals});
   return `${amount<0?"-":""}${report.currencyPosition==="before"?`${report.currencySymbol} ${formatted}`:`${formatted} ${report.currencySymbol}`}`;
  },
  date:(value:string)=>formatRegionalDateTime(value,{country:report.country,timeZone:report.timezone},{dateStyle:"short",timeStyle:"short"}),
  day:(value:string)=>formatRegionalCalendarDate(value,report.country,{dateStyle:"long"}),
  quantity:(value:string|number)=>formatRegionalNumber(Number(value),report.country,{maximumFractionDigits:3}),
 };
}

export function cashMovementSource(source:string){
 const labels:Record<string,string>={cash_sale:"Venta en efectivo",cash_refund:"Reverso de cobro",cash_pull:"Retiro",transfer_in:"Transferencia recibida",transfer_out:"Transferencia enviada",deposit:"Depósito",adjustment:"Ajuste",manual:"Manual"};
 return labels[source]??source;
}
