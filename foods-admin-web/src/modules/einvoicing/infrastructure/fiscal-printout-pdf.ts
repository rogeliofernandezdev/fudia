import QRCode from "qrcode";
import {buildTicketPdf,layoutTicket} from "@/shared/ticket/ticket-80mm";
import {fiscalPrintoutRows,type FiscalPrintFormat} from "../domain/fiscal-printout";
import type {FiscalDocumentDetail} from "../domain/types";

/** 80 mm printed representation with the QR the country defines (library default black on white, for thermal printers). */
export async function buildFiscalPrintoutPdf(detail:FiscalDocumentDetail,format:FiscalPrintFormat,logo?:Uint8Array|null){
 const layout=layoutTicket(fiscalPrintoutRows(detail,format),detail.number,Boolean(logo));
 const qr=await QRCode.toDataURL(detail.printout!.qrData,{errorCorrectionLevel:"Q",margin:0,width:360});
 return buildTicketPdf(layout,{title:`${detail.label} ${detail.number}`,subject:detail.printout!.title,logo,qr});
}
