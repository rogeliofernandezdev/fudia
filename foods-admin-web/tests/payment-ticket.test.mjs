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
const builder=compile("modules/sales/infrastructure/payment-ticket-pdf.ts",name=>name.endsWith("payment-ticket")?domain:require(name));
function fixture(count=3){return{
 receiptContext:{organizationName:"La Panchita",legalName:"Panchita Restaurante SAC",taxId:"20123456789",locationName:"Local principal",address:"Av. Perú 123, Miraflores",phone:"999888777",country:"PE",timezone:"America/Lima",currency:"PEN",currencySymbol:"S/",currencyPosition:"before",currencyDecimals:2},
 order:{id:"order-1",code:"PED-8C26A1",channel:"salon",status:"entregado",customerName:"Familia Calderón",tableName:"03",waiterName:"José Carlos Calderón",createdAt:"2026-10-08T17:00:00Z",subtotal:"60.00",deliveryFee:"0",total:"60.00",items:Array.from({length:count},(_,index)=>({id:`item-${index}`,name:index===0?"Ají de gallina":index===1?"Lomo saltado con arroz y papas fritas":"Chicha morada",qty:"1",unitPrice:"20.00",selections:index===0?[{groupName:"Entrada",name:"Causa rellena"}]:[]}))},
 paidAmount:"60.00",remainingAmount:"0.00",paymentStatus:"paid",payments:[{id:"pay-1",method:"cash",methodName:"Efectivo",amount:"40",netAmount:"40",refundedAmount:"0",reference:"",createdByName:"Ana María",createdAt:"2026-10-08T18:05:00Z"},{id:"pay-2",method:"custom",methodName:"Yape del restaurante",amount:"20",netAmount:"20",refundedAmount:"0",reference:"YP-38294",createdByName:"Ana María",createdAt:"2026-10-08T18:06:00Z"}]};}

test("ticket muestra el consumo y los pagos reales sin simular un documento fiscal",()=>{
 const rows=domain.paymentTicketRows(fixture(),"pay-2"),text=rows.map(row=>`${row.text} ${row.right??""}`).join("\n");
 for(const value of ["La Panchita","20123456789","Av. Perú","José Carlos Calderón","1 x Ají de gallina","Causa rellena","S/ 60.00","Efectivo","Yape del restaurante","YP-38294","Ana María","CUENTA PAGADA","No es un comprobante fiscal"])assert.ok(text.includes(value),value);
 assert.ok(!text.includes("Saldo pendiente"));assert.ok(!/IGV|SUNAT|B001|F001/.test(text));
 assert.ok(text.includes("13:06")||text.includes("1:06"));
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
 const stress=fixture(160);stress.order.items[159].name="ÚLTIMO PRODUCTO - Piña y limón";stress.order.items[0].name="AJI".repeat(100);
 const big=builder.layoutPaymentTicket(stress);assert.ok(big.pages.length>2);
 assert.ok(big.pages.at(-1).texts.some(line=>line.text.includes("Gracias por su visita")));
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
  if(name==="@/shared/pdf-viewer")return{PdfViewer:"PdfViewer"};
  if(name.startsWith("../")||name.endsWith(".css"))return{};return require(name);
 });
 const tree=ui.PaymentTicketDialog({orderId:"order-1",close:()=>{}}),nodes=[];
 function visit(item){if(!item||typeof item!=="object")return;if(Array.isArray(item)){item.forEach(visit);return;}nodes.push(item);visit(item.props?.children);}visit(tree);
 assert.equal(tree.props.document,null);assert.equal(tree.props.loading,true);
});
