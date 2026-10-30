import {jsPDF} from "jspdf";
import {paymentTicketRows} from "../domain/payment-ticket";
import type {SaleDetail} from "../domain/types";

type TicketText={text:string;x:number;y:number;width:number;size:number;bold:boolean;align:"left"|"right"|"center"};
export type PaymentTicketLayout={height:number;pages:{texts:TicketText[];rules:number[]}[];code:string};

// 80 mm roll with a conservative 70 mm content area. Large tickets continue,
// rather than exceeding PDF page limits or truncating consumption lines.
export function layoutPaymentTicket(detail:SaleDetail,paymentId?:string,mode:"payment"|"prebill"="payment",withLogo=false):PaymentTicketLayout{
 const measure=new jsPDF({unit:"mm",format:[80,500],orientation:"portrait"});
 const pages:PaymentTicketLayout["pages"]=[{texts:[],rules:[]}];let y=withLogo?28:6;
 for(const row of paymentTicketRows(detail,paymentId,mode)){
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

export function buildPaymentTicketPdf(layout:PaymentTicketLayout,logo?:Uint8Array|null){
 const doc=new jsPDF({unit:"mm",format:[80,layout.height],orientation:"portrait",compress:true});
 doc.setProperties({title:`Ticket de pago - ${layout.code}`,author:"Fudia",subject:"Ticket operativo no fiscal de 80 mm"});
 layout.pages.forEach((page,index)=>{
  if(index)doc.addPage([80,layout.height],"portrait");
  if(index===0&&logo){const properties=doc.getImageProperties(logo),ratio=Math.min(30/properties.width,18/properties.height);const width=properties.width*ratio,height=properties.height*ratio;doc.addImage(logo,"PNG",(80-width)/2,5,width,height);}
  doc.setTextColor(0);doc.setDrawColor(0);doc.setLineWidth(.15);
  for(const y of page.rules)doc.line(5,y,75,y);
  for(const item of page.texts){doc.setFont("helvetica",item.bold?"bold":"normal");doc.setFontSize(item.size);doc.text(item.text,item.x,item.y,{align:item.align,baseline:"top"});}
 });
 return doc.output("blob");
}
