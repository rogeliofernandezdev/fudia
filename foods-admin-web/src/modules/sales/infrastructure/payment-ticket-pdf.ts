import {jsPDF} from "jspdf";
import {paymentTicketRows} from "../domain/payment-ticket";
import type {SaleDetail} from "../domain/types";

type TicketText={text:string;x:number;y:number;width:number;size:number;bold:boolean;align:"left"|"right"|"center"};
export type PaymentTicketLayout={height:number;pages:{texts:TicketText[];rules:number[]}[];code:string};

// 80 mm roll with a conservative 70 mm content area. Large tickets continue,
// rather than exceeding PDF page limits or truncating consumption lines.
export function layoutPaymentTicket(detail:SaleDetail,paymentId?:string):PaymentTicketLayout{
 const measure=new jsPDF({unit:"mm",format:[80,500],orientation:"portrait"});
 const pages:PaymentTicketLayout["pages"]=[{texts:[],rules:[]}];let y=6;
 for(const row of paymentTicketRows(detail,paymentId)){
  if(row.rule){if(y>475){pages.push({texts:[],rules:[]});y=6;}pages.at(-1)!.rules.push(y+1);y+=5;continue;}
  const size=row.size??9,lineHeight=size*.352778*1.35;
  measure.setFont("helvetica",row.bold?"bold":"normal");measure.setFontSize(size);
  const rightWidth=row.right?Math.min(34,Math.max(22,measure.getTextWidth(row.right)+2)):0;
  const leftWidth=70-rightWidth;
  const left=measure.splitTextToSize(row.text,leftWidth) as string[];
  const right=row.right?measure.splitTextToSize(row.right,rightWidth-1) as string[]:[];
  const lines=Math.max(left.length,right.length);
  for(let index=0;index<lines;index++){
   if(y+lineHeight>480){pages.push({texts:[],rules:[]});y=6;}
   const page=pages.at(-1)!;
   if(left[index])page.texts.push({text:left[index],x:row.center?40:5,y,width:leftWidth,size,bold:Boolean(row.bold),align:row.center?"center":"left"});
   if(right[index])page.texts.push({text:right[index],x:75,y,width:rightWidth,size,bold:Boolean(row.bold),align:"right"});
   y+=lineHeight;
  }
  y+=.6;
 }
 return{height:pages.length>1?486:Math.max(100,Math.ceil(y+6)),pages,code:detail.order.code};
}

export function buildPaymentTicketPdf(layout:PaymentTicketLayout){
 const doc=new jsPDF({unit:"mm",format:[80,layout.height],orientation:"portrait",compress:true});
 doc.setProperties({title:`Ticket de pago - ${layout.code}`,author:"Fudia",subject:"Ticket operativo no fiscal de 80 mm"});
 layout.pages.forEach((page,index)=>{
  if(index)doc.addPage([80,layout.height],"portrait");
  doc.setTextColor(0);doc.setDrawColor(0);doc.setLineWidth(.15);
  for(const y of page.rules)doc.line(5,y,75,y);
  for(const item of page.texts){doc.setFont("helvetica",item.bold?"bold":"normal");doc.setFontSize(item.size);doc.text(item.text,item.x,item.y,{align:item.align,baseline:"top"});}
 });
 return doc.output("blob");
}

const escapeHtml=(value:string)=>value.replace(/[&<>"']/g,char=>({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"}[char]!));

// Same measured lines as the PDF. No scripts, external assets or raw API HTML.
// Printing this isolated document cannot include the shell, modals or buttons.
export function paymentTicketPrintHtml(layout:PaymentTicketLayout){
 return`<!doctype html><html lang="es"><head><meta charset="utf-8"><title>Ticket ${escapeHtml(layout.code)}</title><style>
 @page{size:80mm ${layout.height}mm;margin:0}*{box-sizing:border-box}html,body{margin:0;padding:0;background:white;color:black;font-family:Arial,Helvetica,sans-serif}
 section{position:relative;width:80mm;height:${layout.height}mm;margin:0 auto;break-after:page;overflow:hidden}section:last-child{break-after:auto}
 p{position:absolute;margin:0;line-height:1.35;white-space:pre}hr{position:absolute;left:5mm;width:70mm;margin:0;border:0;border-top:.15mm solid black}
 </style></head><body>${layout.pages.map(page=>`<section>${page.rules.map(y=>`<hr style="top:${y}mm">`).join("")}${page.texts.map(item=>`<p style="top:${item.y}mm;left:${item.align==="right"?item.x-item.width:item.align==="center"?5:item.x}mm;width:${item.align==="center"?70:item.width}mm;font-size:${item.size}pt;font-weight:${item.bold?700:400};text-align:${item.align}">${escapeHtml(item.text)}</p>`).join("")}</section>`).join("")}</body></html>`;
}
