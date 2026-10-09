import {jsPDF} from "jspdf";
import {autoTable,type RowInput} from "jspdf-autotable";
import {cashReportFormat,cashMovementSource} from "../domain/report-format";
import type {CashShiftReport} from "../domain/types";

export type CashReportPalette={primary:string;ink:string;muted:string;surface:string;soft:string;line:string;brand:string};

// Uses selectable text, not a screenshot of the interface. Tables repeat their
// header, wrap long names and continue on new A4 pages without dropping rows.
export function buildCashReportPdf(report:CashShiftReport,logo:Uint8Array,palette:CashReportPalette){
 const doc=new jsPDF({orientation:"landscape",unit:"mm",format:"a4",compress:true});
 const f=cashReportFormat(report),s=report.shift,w=297,margin=16,width=w-margin*2;
 doc.setProperties({title:`Cierre de caja - ${s.code}`,subject:`${report.organizationName} / ${report.locationName}`,author:"Fudia",creator:"Fudia"});
 doc.setCreationDate(new Date(report.generatedAt));
 let y=42;
 function section(title:string,headers:string[],rows:RowInput[],empty:string,widths?:number[]){
  if(y>165){doc.addPage();y=36;}
  doc.setFont("helvetica","bold");doc.setFontSize(12);doc.setTextColor(palette.ink);doc.text(title,margin,y);y+=5;
  autoTable(doc,{startY:y,margin:{top:36,bottom:20,left:margin,right:margin},tableWidth:width,
   head:[headers],body:rows.length?rows:[[{content:empty,colSpan:headers.length}]],theme:"striped",showHead:"everyPage",rowPageBreak:"avoid",
   styles:{font:"helvetica",fontSize:9,cellPadding:2,overflow:"linebreak",textColor:palette.ink,lineColor:palette.line,lineWidth:.1},
   headStyles:{fillColor:palette.primary,textColor:palette.surface,fontStyle:"bold",fontSize:9},alternateRowStyles:{fillColor:palette.soft},
   columnStyles:Object.fromEntries((widths??[]).map((value,index)=>[index,{cellWidth:value,...(index===headers.length-1?{halign:"right" as const}:{})}])),
  });
  y=(doc as jsPDF&{lastAutoTable:{finalY:number}}).lastAutoTable.finalY+6;
 }
 section("Empresa y local",["Empresa","Local"],[[report.organizationName,report.locationName]],"",[132,133]);
 section("Datos del turno",["Caja / turno","Encargado","Apertura","Cierre / responsable"],[[`${s.cashRegisterName}\n${s.code}`,s.openedByName,f.date(s.openedAt),`${s.closedAt?f.date(s.closedAt):"Turno abierto"}\n${s.closedByName||"Sin cierre"}`]],"",[65,65,65,70]);
 if(s.openingNote||s.closingNote)section("Observaciones del turno",["Apertura","Cierre"],[[s.openingNote||"Sin observación",s.closingNote||"Sin observación"]],"");
 const hasRefunds=Number(report.refundedAmount)>0;
 section("Resumen de cobros",["Cobrado en el turno",...(hasRefunds?["Reversos registrados","Cobrado menos reversos"]:[])],[[f.money(report.collectedAmount),...(hasRefunds?[f.money(report.refundedAmount),f.money(report.netCollectedAmount)]:[])]],"");
 section("Arqueo de efectivo",["Fondo inicial","Ingresos de efectivo","Egresos de efectivo","Esperado","Contado","Diferencia"],[[f.money(s.openingAmount),f.money(s.incomeAmount),f.money(s.expenseAmount),f.money(s.closingExpectedAmount??s.expectedAmount),f.money(s.closingCountedAmount),f.money(s.varianceAmount)]],"");
 section("Cobros por medio de pago",["Medio de pago","Importe"],report.methods.map(item=>[item.name,f.money(item.amount)]),"No se registraron cobros.",[185,80]);
 section("Productos vendidos - pedidos con cobros en el turno",["Fecha y hora","Pedido / mesa / mozo","Producto / detalle","Cantidad","Precio unitario","Importe"],report.sales.map(item=>[f.date(item.createdAt),`${item.orderCode}\n${item.customer}${item.waiterName?`\nMozo: ${item.waiterName}`:""}`,`${item.name}${item.details?`\n${item.details}`:""}`,f.quantity(item.quantity),f.money(item.unitPrice),f.money(item.total)]),"No hay productos de pedidos cobrados en este turno.",[32,46,88,25,37,37]);
 section("Alcance de los productos",["Criterio del informe"],[["Cada línea comercial aparece una sola vez. En pagos parciales o en turnos distintos, el valor del pedido no equivale al importe cobrado en este turno. Los cobros y el arqueo se detallan por separado."]],"");
 section("Detalle de cada cobro",["Fecha y hora","Pedido","Medio de pago","Referencia","Cobrado por","Importe"],report.payments.map(item=>[f.date(item.createdAt),item.orderCode,item.methodName,item.reference||"-",item.createdByName,f.money(item.amount)]),"No se registraron cobros.",[32,37,43,58,58,37]);
 const expenses=s.movements?.filter(item=>item.movementType==="expense")??[];
 const incomes=s.movements?.filter(item=>item.movementType==="income"&&item.sourceType!=="cash_sale")??[];
 for(const [title,items] of [["Egresos de efectivo",expenses],["Otros ingresos de efectivo",incomes]] as const){
  section(title,["Fecha y hora","Origen","Motivo / observación","Registrado por","Importe"],items.map(item=>[f.date(item.createdAt),cashMovementSource(item.sourceType),`${item.reason}${item.note?`\n${item.note}`:""}`,item.createdByName,f.money(item.amount)]),`No se registraron ${title.toLowerCase()}.`,[32,43,95,58,37]);
 }
 if(report.refunds.length)section("Reversos de cobros registrados en el turno",["Fecha y hora","Pedido","Medio de pago","Motivo","Registrado por","Importe"],report.refunds.map(item=>[f.date(item.createdAt),item.orderCode,item.methodName,item.reference,item.createdByName,f.money(item.amount)]),"",[32,37,43,58,58,37]);
 if(report.counts.length)section("Conteo por denominaciones",["Denominación","Cantidad","Importe"],report.counts.map(item=>[f.money(item.denomination),String(item.quantity),f.money(item.total)]),"",[100,65,100]);
 const pages=doc.getNumberOfPages();
 for(let page=1;page<=pages;page++){
  doc.setPage(page);
  doc.setFillColor(palette.primary);doc.rect(0,0,w,3,"F");doc.setFillColor(palette.brand);doc.rect(0,0,w/3,3,"F");
  doc.addImage(logo,"PNG",margin,9,12,12,"fudia-logo","FAST");doc.setFont("helvetica","bold");doc.setFontSize(18);doc.setTextColor(palette.ink);doc.text("fudIA",margin+16,17);
  doc.setFontSize(10);doc.text(s.status==="closed"?"CIERRE DE CAJA":"RESUMEN DEL TURNO",w-margin,14,{align:"right"});
  doc.setFont("helvetica","normal");doc.setFontSize(9);doc.setTextColor(palette.muted);
  // Avoid letting long organization names collide with the right-hand title.
  const identity=doc.splitTextToSize(`${report.organizationName} / ${report.locationName}`,160) as string[];
  doc.text(identity.slice(0,2),margin+16,23);doc.text(s.code,w-margin,21,{align:"right"});
  doc.setDrawColor(palette.line);doc.line(margin,31,w-margin,31);doc.line(margin,193,w-margin,193);
  doc.setFontSize(8);doc.text(`${f.day(s.businessDate)} · ${report.timezone}`,margin,198);
  doc.text(`${report.persisted?"Copia del cierre registrado":"Detalle reconstruido / consulta"} · ${page} / ${pages}`,w-margin,198,{align:"right"});
  doc.text(`Generado: ${f.date(report.generatedAt)} · Moneda: ${report.currency} · Documento de control interno`,margin,203);
 }
 return doc.output("blob");
}

export async function loadFudiaReportLogo(){
 const response=await fetch("/assets/images/logo.png",{cache:"force-cache"});
 if(!response.ok)throw new Error("No pudimos cargar el logo de Fudia. Reintenta la vista previa.");
 return new Uint8Array(await response.arrayBuffer());
}
