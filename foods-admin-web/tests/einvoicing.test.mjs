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
