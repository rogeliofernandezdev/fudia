import type {TicketRow} from "@/shared/ticket/ticket-80mm";
import type {FiscalDocumentDetail,FiscalStatus} from "./types";

/** How long checkout waits for the tax authority before offering the signed document anyway. */
export const AUTHORITY_WAIT_MS=8000;

/**
 * What the print dialog shows for a document:
 * waiting  → still sending, inside the wait window;
 * ready    → accepted or observed, printable;
 * pending  → signed but not confirmed yet (slow authority or temporary error), printable;
 * rejected → final rejection, nothing fiscal to print;
 * unsigned → not signed yet (setup problem or still queued after the wait).
 */
export type FiscalPrintPhase="waiting"|"ready"|"pending"|"rejected"|"unsigned";

export function fiscalPrintPhase(document:{status:FiscalStatus;printout:FiscalDocumentDetail["printout"]},waited:boolean):FiscalPrintPhase{
 const{status,printout}=document;
 if(status==="REJECTED"||status==="CANCELLED")return "rejected";
 if(status==="ACCEPTED"||status==="OBSERVED")return printout?"ready":"unsigned";
 if(status!=="ERROR"&&!waited)return "waiting";
 return printout?"pending":"unsigned";
}

export type FiscalPrintFormat={money:(value:string)=>string;date:(value:string)=>string;quantity:(value:string)=>string;percent:(rate:string)=>string};

/** Printed representation on an 80 mm roll; legal texts come from the API printout. */
export function fiscalPrintoutRows(detail:FiscalDocumentDetail,format:FiscalPrintFormat):TicketRow[]{
 const printout=detail.printout;
 if(!printout)throw new Error("El comprobante aún no está firmado. Reintenta en unos segundos.");
 const rows:TicketRow[]=[];
 const text=(value:string,options:Omit<TicketRow,"text">={})=>{if(value)rows.push({text:value,...options});};
 const rule=()=>rows.push({text:"",rule:true});
 const{issuer}=printout;
 text(issuer.tradeName||issuer.legalName,{bold:true,center:true,size:13});
 if(issuer.legalName!==issuer.tradeName)text(issuer.legalName,{center:true});
 text(`${printout.issuerTaxIdLabel}: ${issuer.taxId}`,{center:true});
 text(issuer.address,{center:true});rule();
 text(printout.title,{bold:true,center:true,size:11});
 text(detail.number,{bold:true,center:true,size:11});rule();
 text(`Fecha de emisión: ${format.date(detail.issueDate)}`);
 if(detail.customer.documentType){
  text(`Cliente: ${detail.customer.name}`);
  text(`${detail.customer.documentType}: ${detail.customer.documentNumber}`);
  text(detail.customer.address?`Dirección: ${detail.customer.address}`:"");
 }
 text(`Pedido: ${detail.orderCode}`);rule();
 for(const line of detail.breakdown.lines)text(`${format.quantity(line.quantity)} x ${line.description}`,{right:format.money(line.total)});
 rule();
 text("Valor de venta",{right:format.money(detail.breakdown.netTotal)});
 for(const tax of detail.breakdown.taxes)text(`${tax.name} ${format.percent(tax.rate)}`,{right:format.money(tax.amount)});
 text("TOTAL",{right:format.money(detail.breakdown.total),bold:true,size:12});
 text(printout.amountInWords,{size:8});rule();
 rows.push({text:"",qr:30});
 text(`Valor resumen: ${detail.digestValue}`,{center:true,size:7});
 for(const legend of printout.legends)text(legend,{center:true,size:8});
 if(printout.notice){rule();text(printout.notice.toUpperCase(),{bold:true,center:true,size:9});}
 text("Gestionado con Fudia",{center:true,size:8});
 return rows;
}
