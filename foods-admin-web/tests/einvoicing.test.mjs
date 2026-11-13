import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url),root=new URL("../src/",import.meta.url);
const read=path=>readFileSync(new URL(path,root),"utf8");
function compile(path){
 const exports={};
 vm.runInNewContext(ts.transpileModule(read(path),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports,require});
 return exports;
}
const fiscal=compile("modules/einvoicing/domain/fiscal.ts");
const options={enabled:true,defaultDocument:"receipt",documentTypes:[{value:"none",label:"Sin comprobante"},{value:"receipt",label:"Boleta"},{value:"invoice",label:"Factura"}],
 customerDocumentTypes:[{value:"DNI",label:"DNI",pattern:"^\\d{8}$"},{value:"RUC",label:"RUC",pattern:"^\\d{11}$"}],identityThreshold:"700.00",invoiceCustomerDocumentType:"RUC",currency:"PEN"};
const draft=patch=>({documentType:"receipt",customerDocumentType:"",customerDocumentNumber:"",customerName:"",customerAddress:"",...patch});

test("el comprobante inicial sigue la configuración del local y sin opciones no se emite",()=>{
 assert.equal(fiscal.blankFiscalDraft(options).documentType,"receipt");
 assert.equal(fiscal.blankFiscalDraft(undefined).documentType,"none");
 assert.equal(fiscal.blankFiscalDraft({...options,enabled:false}).documentType,"none");
});

test("la boleta exige identificar al cliente solo sobre el umbral del país",()=>{
 assert.deepEqual({...fiscal.validateFiscalDraft(options,draft(),85)},{});
 assert.ok(fiscal.validateFiscalDraft(options,draft(),700.01).customerDocumentNumber);
 assert.ok(fiscal.validateFiscalDraft(options,draft({customerDocumentType:"DNI",customerDocumentNumber:"4567"}),20).customerDocumentNumber);
 const missingName=fiscal.validateFiscalDraft(options,draft({customerDocumentType:"DNI",customerDocumentNumber:"45678901"}),20);
 assert.ok(missingName.customerName&&!missingName.customerDocumentNumber);
});

test("la factura usa el documento fiscal que indica el API",()=>{
 const invoice=draft({documentType:"invoice",customerDocumentNumber:"20100047218",customerName:" Empresa SAC ",customerAddress:" Av. Larco 123 "});
 assert.deepEqual({...fiscal.validateFiscalDraft(options,invoice,50)},{});
 assert.ok(fiscal.validateFiscalDraft(options,{...invoice,customerDocumentNumber:"123"},50).customerDocumentNumber);
 assert.ok(fiscal.validateFiscalDraft(options,{...invoice,customerName:""},50).customerName);
 assert.deepEqual(JSON.parse(JSON.stringify(fiscal.fiscalRequest(options,invoice))),{documentType:"invoice",customer:{documentType:"RUC",documentNumber:"20100047218",name:"Empresa SAC",address:"Av. Larco 123"}});
});

test("Sin comprobante no envía nada y una boleta anónima no envía documento",()=>{
 assert.equal(fiscal.fiscalRequest(options,draft({documentType:"none"})),undefined);
 assert.equal(fiscal.fiscalRequest({...options,enabled:false},draft()),undefined);
 assert.deepEqual(JSON.parse(JSON.stringify(fiscal.fiscalRequest(options,draft({customerDocumentNumber:"99"})))),{documentType:"receipt",customer:{documentType:"",documentNumber:"",name:"",address:""}});
});

test("todos los estados fiscales tienen etiqueta y tono central",()=>{
 for(const status of ["PENDING","SENDING","ACCEPTED","OBSERVED","REJECTED","ERROR","CANCELLED"]){
  const view=fiscal.fiscalStatusView[status];
  assert.ok(view.label&&["sky","green","orange","red","gray"].includes(view.tone),status);
 }
});

test("el cobro solo ofrece comprobante en el pago que completa la cuenta y la web no fija catálogos fiscales",()=>{
 const form=read("modules/operations/pos/presentation/pos-dialogs.tsx");
 assert.ok(form.includes("const offerFiscal=Boolean(fiscalOptions?.enabled)&&settles"));
 assert.ok(form.includes("fiscal:offerFiscal?fiscalRequest(fiscalOptions,fiscal):undefined"));
 for(const path of ["modules/einvoicing/presentation/checkout-fiscal-section.tsx","modules/einvoicing/domain/fiscal.ts"]){
  const source=read(path);
  assert.equal(/"RUC"|"DNI"|"B001"|"F001"/.test(source),false,`${path} no fija documentos ni series`);
 }
 assert.ok(read("modules/operations/pos/presentation/pos-page.tsx").includes("fiscalOptions={fiscal.data?.enabled?fiscal.data:undefined}"));
});

test("las credenciales se arman con los campos del proveedor y nunca se leen de vuelta",()=>{
 const page=read("modules/einvoicing/presentation/einvoicing-settings-page.tsx");
 assert.ok(page.includes("profile.credentials.fields??[]"),"los campos vienen del API");
 assert.ok(page.includes('autoComplete={field.type==="secret"?"new-password":"off"}'));
 assert.ok(page.includes("readAsDataURL"),"el certificado viaja en base64");
 assert.ok(!/values\[field\.key\]\?\?profile/.test(page),"no hay valores guardados que mostrar");
 assert.ok(!page.includes("MODDATOS")&&!page.includes('"solUser",type'),"sin catálogos del proveedor en la web");
});

test("el detalle descarga XML y CDR por el BFF sin corromper binarios",()=>{
 const api=read("modules/einvoicing/infrastructure/einvoicing-api.ts");
 assert.ok(api.includes("/api/admin/fiscal-documents/${encodeURIComponent(id)}/files/${kind}"));
 const proxy=read("app/api/admin/[...path]/route.ts");
 assert.ok(proxy.includes("await response.arrayBuffer()")&&proxy.includes("Content-Disposition"));
});

test("el módulo Facturación es la única activación: sin segundo interruptor y con aviso en el cobro",()=>{
 const page=read("modules/einvoicing/presentation/einvoicing-settings-page.tsx");
 assert.equal(page.includes("Emitir comprobantes electrónicos"),false,"no existe un interruptor por perfil fiscal");
 assert.equal(/draft\.enabled|profile\.enabled/.test(page),false,"el perfil solo guarda proveedor y ambiente");
 assert.ok(page.includes('profile.ready?"Lista para emitir":"Falta configurar"'),"el estado se deriva de la configuración real");
 const types=read("modules/einvoicing/domain/types.ts");
 assert.ok(types.includes('export type ProfileSettingsInput={provider:string;environment:"test"|"production"};'));
 const form=read("modules/operations/pos/presentation/pos-dialogs.tsx");
 assert.ok(form.includes('{!offerFiscal&&settles&&fiscalIssue&&<InlineNotice tone="warning" title="Boleta y factura no disponibles">{fiscalIssue}</InlineNotice>}'),"el cobro explica qué falta en vez de ocultarlo");
 assert.ok(read("modules/operations/pos/presentation/pos-page.tsx").includes("fiscalIssue={fiscal.data&&!fiscal.data.enabled?fiscal.data.issue:undefined}"));
});

test("el cobro consulta la facturación vigente al abrirse y Mi restaurante la invalida al guardar",()=>{
 const hook=read("modules/einvoicing/presentation/checkout-fiscal-section.tsx");
 assert.ok(hook.includes('staleTime:0,refetchOnMount:"always"'),"un cambio de RUC, credenciales o series se ve en el siguiente cobro");
 assert.ok(read("modules/operations/pos/presentation/pos-page.tsx").includes("(fiscal.isFetchedAfterMount||fiscal.isError)"),"el formulario espera la respuesta fresca, no la caché");
 const company=read("modules/organizations/presentation/organization-admin.tsx");
 assert.ok(company.includes('client.invalidateQueries({queryKey:["einvoicing-checkout"]})')&&company.includes('client.invalidateQueries({queryKey:["einvoicing-settings"]})'));
});

const printout=compile("modules/einvoicing/domain/fiscal-printout.ts");
const signed={title:"BOLETA DE VENTA ELECTRÓNICA",issuer:{taxId:"20000000001",legalName:"A Mi Manera E.I.R.L.",tradeName:"La Rinconada",address:"San Juan Bautista, Ayacucho"},issuerTaxIdLabel:"RUC",
 amountInWords:"SON: DIEZ CON 00/100 SOLES",qrData:"20000000001|03|B001|1|1.53|10.00|2026-11-03|1|46351741|abc=|",legends:["Representación impresa de la Boleta de Venta Electrónica.","Consulte su validez en www.sunat.gob.pe"],notice:"Ambiente de pruebas: sin validez tributaria."};

test("el cobro espera a la autoridad unos segundos y luego entrega el comprobante firmado",()=>{
 assert.equal(printout.AUTHORITY_WAIT_MS<=10000,true,"la espera no bloquea la caja");
 assert.equal(printout.fiscalPrintPhase({status:"SENDING",printout:null},false),"waiting");
 assert.equal(printout.fiscalPrintPhase({status:"SENDING",printout:signed},false),"waiting","dentro de la espera aún se intenta mostrarlo aceptado");
 assert.equal(printout.fiscalPrintPhase({status:"ACCEPTED",printout:signed},false),"ready");
 assert.equal(printout.fiscalPrintPhase({status:"OBSERVED",printout:signed},false),"ready");
 assert.equal(printout.fiscalPrintPhase({status:"SENDING",printout:signed},true),"pending","firmado pero sin respuesta se imprime igual");
 assert.equal(printout.fiscalPrintPhase({status:"ERROR",printout:signed},false),"pending","un error temporal no sigue esperando");
 assert.equal(printout.fiscalPrintPhase({status:"ERROR",printout:null},false),"unsigned");
 assert.equal(printout.fiscalPrintPhase({status:"PENDING",printout:null},true),"unsigned");
 assert.equal(printout.fiscalPrintPhase({status:"REJECTED",printout:null},false),"rejected","un rechazo nunca se imprime");
});

test("la representación impresa usa los textos legales del API, el número fiscal y el QR",()=>{
 const detail={id:"doc-1",documentType:"receipt",label:"Boleta",number:"B001-1",issueDate:"2026-11-03",orderCode:"PED-75157E9E",digestValue:"abc=",status:"ACCEPTED",
  customer:{documentType:"DNI",documentNumber:"46351741",name:"Rogelio Fernandez",address:""},
  breakdown:{lines:[{no:1,description:"Papa a la Huancaína",quantity:"1",unitPrice:"10",netAmount:"8.47",taxAmount:"1.53",total:"10.00"}],taxes:[{code:"general",name:"IGV",rate:"0.18",amount:"1.53"}],netTotal:"8.47",taxTotal:"1.53",total:"10.00"},
  printout:signed};
 const format={money:value=>`S/ ${Number(value).toFixed(2)}`,date:value=>value,quantity:value=>String(Number(value)),percent:rate=>`${Number(rate)*100} %`};
 const rows=printout.fiscalPrintoutRows(detail,format),texts=rows.map(row=>row.text);
 for(const expected of ["La Rinconada","A Mi Manera E.I.R.L.","RUC: 20000000001","BOLETA DE VENTA ELECTRÓNICA","B001-1","Cliente: Rogelio Fernandez","DNI: 46351741","Pedido: PED-75157E9E","SON: DIEZ CON 00/100 SOLES","Valor resumen: abc=","Representación impresa de la Boleta de Venta Electrónica.","AMBIENTE DE PRUEBAS: SIN VALIDEZ TRIBUTARIA."])
  assert.ok(texts.includes(expected),expected);
 assert.ok(rows.some(row=>row.text==="IGV 18 %"&&row.right==="S/ 1.53"));
 assert.ok(rows.some(row=>row.text==="TOTAL"&&row.right==="S/ 10.00"&&row.bold));
 assert.equal(rows.filter(row=>row.qr).length,1,"un único QR");
 assert.ok(!texts.some(text=>/no es un comprobante fiscal/i.test(text)),"no se confunde con el ticket de pago");
 const anonymous=printout.fiscalPrintoutRows({...detail,customer:{documentType:"",documentNumber:"",name:"",address:""}},format).map(row=>row.text);
 assert.ok(!anonymous.some(text=>text.startsWith("Cliente:")),"la boleta anónima no inventa un cliente");
 assert.throws(()=>printout.fiscalPrintoutRows({...detail,printout:null},format),/firmado/);
});

test("con comprobante el cobro abre la representación impresa en lugar del ticket de pago",()=>{
 const pos=read("modules/operations/pos/presentation/pos-page.tsx");
 assert.ok(pos.includes("if(fiscal){setFiscalTarget({documentId:fiscal.id,orderId:variables.orderId,paymentId:item.id,wait:true});return;}"));
 assert.ok(pos.includes("<FiscalPrintDialog documentId={fiscalTarget.documentId} wait={fiscalTarget.wait}"),"solo el cobro espera la respuesta; la reimpresión no");
 assert.ok(pos.includes("setTicketTarget({orderId:fiscalTarget.orderId,paymentId:fiscalTarget.paymentId})"),"si no se puede imprimir queda el ticket de pago");
 const dialog=read("modules/einvoicing/presentation/fiscal-print-dialog.tsx");
 assert.ok(dialog.includes("prepared?.key===key"),"las consultas de estado no regeneran el PDF");
 assert.ok(dialog.includes('else if(data.status!=="ACCEPTED")notice='),"aceptado es el caso normal: sin franja de estado");
 for(const viewer of ["modules/einvoicing/presentation/fiscal-print-dialog.tsx","modules/sales/presentation/payment-ticket-dialog.tsx","modules/operations/cash/presentation/cash-report-dialog.tsx"])
  assert.ok(!/<PdfViewer[^>]*subtitle=/.test(read(viewer)),`${viewer}: la cabecera del visor solo lleva título y X`);
 assert.ok(read("modules/einvoicing/presentation/fiscal-documents-page.tsx").includes("Ver comprobante"),"se reimprime desde Comprobantes");
 assert.ok(!read("modules/einvoicing/infrastructure/fiscal-printout-pdf.ts").includes("SUNAT"),"el contenido legal viene del API");
});

test("la reimpresión muestra lo que se entregó: comprobante con valor fiscal o ticket de pago",()=>{
 const ref=status=>({id:"doc-1",documentType:"receipt",label:"Boleta",number:"B001-1",status});
 for(const status of ["ACCEPTED","OBSERVED","PENDING","SENDING","ERROR"])assert.equal(fiscal.deliveredDocument(ref(status))?.id,"doc-1",status);
 for(const status of ["REJECTED","CANCELLED"])assert.equal(fiscal.deliveredDocument(ref(status)),null,status);
 assert.equal(fiscal.deliveredDocument(undefined),null,"sin comprobante: ticket de pago");
 assert.equal(fiscal.ticketNoticeFor(ref("REJECTED")),"Boleta B001-1 · Rechazado. Este ticket no es un comprobante fiscal.");
 assert.equal(fiscal.ticketNoticeFor(ref("ACCEPTED")),undefined);
 assert.equal(fiscal.ticketNoticeFor(undefined),undefined);
 for(const page of ["modules/sales/presentation/sales-page.tsx","modules/operations/pos/presentation/pos-page.tsx"]){
  const source=read(page);
  assert.ok(source.includes("deliveredDocument(detail.data?.fiscalDocument)")&&source.includes("ticketNoticeFor(detail.data?.fiscalDocument)"),page);
 }
 for(const dialog of ["modules/sales/presentation/sale-detail-dialog.tsx","modules/operations/pos/presentation/pos-page.tsx"])
  assert.ok(read(dialog).includes('{data.fiscalDocument.label} {data.fiscalDocument.number}'),`${dialog} identifica el comprobante entregado`);
});
