import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync,writeFileSync,mkdirSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url),root=new URL("../src/",import.meta.url);
const read=file=>readFileSync(new URL(file,root),"utf8");
function compile(file,resolve=require){const exports={};vm.runInNewContext(ts.transpileModule(read(file),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve,Blob,Uint8Array,Date});return exports;}
const regional=compile("shared/i18n/regional-format.ts");
const domain=compile("modules/sales/domain/payment-ticket.ts",name=>name==="@/shared/i18n/regional-format"?regional:require(name));
const ticket=compile("shared/ticket/ticket-80mm.ts");
const builder=compile("modules/sales/infrastructure/payment-ticket-pdf.ts",name=>name.endsWith("payment-ticket")?domain:name==="@/shared/ticket/ticket-80mm"?ticket:require(name));
function fixture(count=3){return{
 receiptContext:{organizationName:"La Panchita",legalName:"Panchita Restaurante SAC",taxId:"20123456789",locationName:"Local principal",address:"Av. Perú 123, Miraflores",phone:"999888777",country:"PE",timezone:"America/Lima",currency:"PEN",currencySymbol:"S/",currencyPosition:"before",currencyDecimals:2},
 order:{id:"order-1",code:"PED-8C26A1",channel:"salon",status:"entregado",customerName:"Familia Calderón",tableName:"03",waiterName:"José Carlos Calderón",createdAt:"2026-10-08T17:00:00Z",subtotal:"60.00",deliveryFee:"0",total:"60.00",items:Array.from({length:count},(_,index)=>({id:`item-${index}`,name:index===0?"Ají de gallina":index===1?"Lomo saltado con arroz y papas fritas":"Chicha morada",qty:"1",unitPrice:"20.00",selections:index===0?[{groupName:"Entrada",name:"Causa rellena"}]:[]}))},
 paidAmount:"60.00",remainingAmount:"0.00",paymentStatus:"paid",payments:[{id:"pay-1",method:"cash",methodName:"Efectivo",amount:"40",netAmount:"40",refundedAmount:"0",reference:"",createdByName:"Ana María",createdAt:"2026-10-08T18:05:00Z"},{id:"pay-2",method:"custom",methodName:"Yape del restaurante",amount:"20",netAmount:"20",refundedAmount:"0",reference:"YP-38294",createdByName:"Ana María",createdAt:"2026-10-08T18:06:00Z"}]};}

test("precuenta muestra sugerencia voluntaria sin inventar pagos ni sumar delivery a su base",()=>{
 const sample=fixture();sample.payments=[];sample.paidAmount="0";sample.remainingAmount="60";sample.order.deliveryFee="10";
 sample.tipPolicy={mode:"suggest",percentage:10,showSuggestion:true,channels:["salon"]};
 const rows=domain.paymentTicketRows(sample,undefined,"prebill");
 assert.ok(rows.some(v=>v.text==="PRECUENTA"));
 assert.ok(rows.some(v=>v.text.includes("Propina sugerida")&&v.right==="S/ 5.00"));
 assert.ok(rows.some(v=>v.text==="Total si acepta la sugerencia"&&v.right==="S/ 65.00"));
 assert.ok(!rows.some(v=>v.text==="PAGOS REGISTRADOS"||v.text==="CUENTA PAGADA"));
 assert.equal(sample.paidAmount,"0");assert.equal(sample.order.total,"60.00");
 sample.tipPolicy.mode="disabled";
 assert.ok(!domain.paymentTicketRows(sample,undefined,"prebill").some(v=>v.text.includes("Propina sugerida")));
});

test("el ticket no imprime lo que la boleta tampoco imprime: propina, pagos, mesa, mozo ni hora",()=>{
 const sample=fixture();sample.tip={paymentId:"pay-1",amount:"6",directToStaff:false};sample.order.deliveryFee="5";sample.order.total="65.00";
 const rows=domain.paymentTicketRows(sample,"pay-2"),text=rows.map(row=>`${row.text} ${row.right??""}`).join("\n");
 for(const absent of ["Propina","Efectivo","Yape del restaurante","YP-38294","Ana María","Cobrado por","Mesa","Mozo","José Carlos Calderón","Causa rellena","Subtotal","Gracias por su visita","Pagado"])assert.ok(!text.includes(absent),absent);
 assert.ok(!/\d{1,2}:\d{2}/.test(text),"la fecha de emisión no lleva hora, como en la boleta");
 assert.ok(rows.some(row=>row.text==="1 x Delivery"&&row.right==="S/ 5.00"),"el delivery es una línea más, como en el desglose fiscal");
});

test("ticket sigue la boleta sección por sección sin simular un documento fiscal",()=>{
 const rows=domain.paymentTicketRows(fixture(),"pay-2"),text=rows.map(row=>`${row.text} ${row.right??""}`).join("\n");
 for(const value of ["La Panchita","Panchita Restaurante SAC","RUC: 20123456789","Av. Perú","Cliente: Familia Calderón","1 x Ají de gallina","S/ 60.00","No es un comprobante fiscal","Gestionado con Fudia"])assert.ok(text.includes(value),value);
 assert.ok(!text.includes("Saldo pendiente"));assert.ok(!/IGV|SUNAT|B001|F001|ELECTRÓNIC|CUENTA PAGADA|SON:/.test(text),"no se presenta como comprobante electrónico");
 // Same order as the printed electronic document: issuer, rule, title and number, rule, data, rule, lines, rule, total.
 const title=rows.findIndex(row=>row.text==="TICKET DE VENTA");
 assert.ok(title>0&&rows[title-1].rule&&rows[title].bold&&rows[title+1].text==="PED-8C26A1"&&rows[title+1].bold&&rows[title+2].rule);
 assert.match(rows[title+3].text,/^Fecha de emisión: /);
 assert.ok(rows.some(row=>row.text==="1 x Ají de gallina"&&row.right==="S/ 20.00"),"cantidad, producto e importe en una línea");
 assert.ok(rows.some(row=>row.text==="TOTAL"&&row.right==="S/ 60.00"&&row.bold));
 const longName=fixture();longName.receiptContext.organizationName="Restaurante peruano con un nombre muy largo y atención familiar";longName.order.items[0].name="<script>alert('unsafe')</script> & Ají";
 const layout=builder.layoutPaymentTicket(longName);assert.ok(layout.pages[0].texts.some(line=>line.text.includes("<script>")),"el texto del documento no se interpreta como HTML");
});

test("pago parcial conserva saldo y no afirma pagado; un ticket exige cobros confirmados",()=>{
 const partial=fixture();partial.payments.pop();partial.paidAmount="40";partial.remainingAmount="20";partial.paymentStatus="partial";
 const rows=domain.paymentTicketRows(partial),text=rows.map(row=>`${row.text} ${row.right??""}`).join("\n");assert.match(text,/Saldo pendiente S\/ 20.00/);assert.match(text,/PAGO PARCIAL/);assert.ok(!text.includes("CUENTA PAGADA"));
 assert.throws(()=>domain.paymentTicketRows({...partial,receiptContext:undefined}),/datos del restaurante/);
 assert.throws(()=>domain.paymentTicketRows(partial,"pay-2"),/sin volver a cobrar/);
 assert.throws(()=>domain.paymentTicketRows({...partial,payments:[]}),/no tiene cobros/);
 const foreign=fixture();foreign.receiptContext.currencySymbol="€";foreign.receiptContext.currencyPosition="after";foreign.receiptContext.currencyDecimals=3;
 assert.ok(domain.paymentTicketRows(foreign).some(row=>row.right==="60.000 €"));
});

test("PDF real mide 80 mm y conserva todas las líneas en tickets grandes",async()=>{
 const sample=fixture(),layout=builder.layoutPaymentTicket(sample,"pay-2");
 const stress=fixture(400);stress.order.items[399].name="ÚLTIMO PRODUCTO - Piña y limón";stress.order.items[0].name="AJI".repeat(100);
 const big=builder.layoutPaymentTicket(stress);assert.ok(big.pages.length>2);
 assert.ok(big.pages.at(-1).texts.some(line=>line.text.includes("Gestionado con Fudia")));
 for(const plan of [layout,big]){
  for(const page of plan.pages)for(const line of page.texts){assert.ok(line.y>=6&&line.y<480);assert.ok(line.x>=5&&line.x<=75);}
  const blob=builder.buildPaymentTicketPdf(plan),buffer=Buffer.from(await blob.arrayBuffer());assert.equal(blob.type,"application/pdf");
  const boxes=[...buffer.toString("latin1").matchAll(/\/MediaBox \[0 0 ([\d.]+) ([\d.]+)\]/g)];assert.equal(boxes.length,plan.pages.length);for(const box of boxes)assert.ok(Math.abs(Number(box[1])-80*72/25.4)<.01);
  if(process.env.PAYMENT_TICKET_QA_DIR){mkdirSync(process.env.PAYMENT_TICKET_QA_DIR,{recursive:true});writeFileSync(`${process.env.PAYMENT_TICKET_QA_DIR}/${plan===layout?"sample":"stress"}.pdf`,buffer);}
 }
});

test("ticket usa PdfViewer sin visor nativo, variante Android ni impresión fuera del visor",()=>{
 const source=read("modules/sales/presentation/payment-ticket-dialog.tsx");
 assert.doesNotMatch(source,/Imprime al/);
 assert.doesNotMatch(source,/no reemplaza una boleta o factura/,"sin aviso sobre el visor");
 assert.doesNotMatch(source,/Imprimir ticket|contentWindow\.print|srcDoc/,"el visor del PDF ya ofrece imprimir");
 assert.match(source,/<PdfViewer/);assert.match(source,/cancelled=true/);assert.ok(!source.includes("createPayment"));assert.match(source,/organization\?\.id,location\?\.id,orderId/);
 assert.doesNotMatch(source,/iframe|pdfViewerEnabled|TicketPaper|ModalFooter|URL\.createObjectURL|window\.print/);
});

test("ticket entrega el Blob real y su nombre al visor compartido",()=>{
 const data=fixture(),document=new Blob(["%PDF-fixture"],{type:"application/pdf"});
 let state=0;const values=[{document,filename:"ticket.pdf",source:data,paymentId:"pay-2"},null,0];
 const ui=compile("modules/sales/presentation/payment-ticket-dialog.tsx",name=>{
  if(name==="react")return{useEffect:()=>{},useState:()=>[values[state++],()=>{}]};
  if(name==="@tanstack/react-query")return{useQuery:()=>({data,isFetching:false,isPending:false,isError:false})};
  if(name==="@/providers")return{useSession:()=>({organization:{id:"org"},location:{id:"loc"}})};
  if(name==="@/shared/api/restaurant-branding")return {loadRestaurantReportLogo:async()=>null};
  if(name==="@/shared/pdf-viewer")return{PdfViewer:"PdfViewer"};
  if(name.startsWith("../")||name.endsWith(".css"))return{};return require(name);
 });
 const tree=ui.PaymentTicketDialog({orderId:"order-1",paymentId:"pay-2",close:()=>{}}),nodes=[];
 function visit(item){if(!item||typeof item!=="object")return;if(Array.isArray(item)){item.forEach(visit);return;}nodes.push(item);visit(item.props?.children);}visit(tree);
 assert.equal(tree.type,"PdfViewer");assert.equal(tree.props.document,document);assert.equal(tree.props.filename,"ticket.pdf");assert.equal(tree.props.loading,false);
});
test("cambiar el documento o pago nunca muestra el PDF anterior",()=>{
 let state=0;
 const values=[{document:new Blob(["otro"]),filename:"anterior.pdf",source:fixture(),paymentId:"pay-1"},null,0];
 const ui=compile("modules/sales/presentation/payment-ticket-dialog.tsx",name=>{
  if(name==="react")return{useEffect:()=>{},useState:()=>[values[state++],()=>{}]};
  if(name==="@tanstack/react-query")return{useQuery:()=>({data:fixture(),isFetching:false,isPending:false,isError:false})};
  if(name==="@/providers")return{useSession:()=>({organization:{id:"org"},location:{id:"loc"}})};
  if(name==="@/shared/api/restaurant-branding")return {loadRestaurantReportLogo:async()=>null};
  if(name==="@/shared/pdf-viewer")return{PdfViewer:"PdfViewer"};
  if(name.startsWith("../")||name.endsWith(".css"))return{};return require(name);
 });
 const tree=ui.PaymentTicketDialog({orderId:"order-1",close:()=>{}}),nodes=[];
 function visit(item){if(!item||typeof item!=="object")return;if(Array.isArray(item)){item.forEach(visit);return;}nodes.push(item);visit(item.props?.children);}visit(tree);
 assert.equal(tree.props.document,null);assert.equal(tree.props.loading,true);
});

test("el ticket de 80 mm reserva el espacio del QR sin superponer texto",()=>{
 const layout=ticket.layoutTicket([{text:"Antes"},{text:"",qr:30},{text:"Después"}],"B001-1");
 const [image]=layout.pages[0].images,[before,after]=layout.pages[0].texts;
 assert.equal(image.size,30);
 assert.ok(image.y>before.y&&after.y>=image.y+image.size,"el texto continúa debajo del QR");
 assert.ok(ticket.buildTicketPdf(layout,{title:"Boleta B001-1",subject:"Comprobante"})instanceof Blob,"sin imagen de QR también genera el PDF");
});
