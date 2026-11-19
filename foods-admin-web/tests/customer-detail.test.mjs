import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
const require=createRequire(import.meta.url);
const root=new URL("../",import.meta.url);
const read=file=>readFileSync(new URL(file,root),"utf8");
function compile(file){
 const exports={};
 vm.runInNewContext(ts.transpileModule(read(file),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports,require});
 return exports;
}
const display=compile("src/modules/customers/domain/customer-display.ts");
const manager=read("src/modules/customers/presentation/customers-manager.tsx");
const detail=manager.slice(manager.indexOf("function CustomerDetailContent"),manager.indexOf("function CustomerDetailSkeleton"));

test("el celular peruano se agrupa para leerlo y los demás números no se alteran",()=>{
 assert.equal(display.formatCustomerPhone("+51941452580"),"+51 941 452 580");
 assert.equal(display.formatCustomerPhone("941452580"),"941 452 580");
 assert.equal(display.formatCustomerPhone("+51 941-452-580"),"+51 941 452 580");
 assert.equal(display.formatCustomerPhone("014251234"),"014251234");
});

test("promociones: manda el permiso vigente y el último evento explica su ausencia",()=>{
 assert.equal(display.marketingConsentLabel({marketingConsent:true,marketingConsentStatus:"pending"}),"Acepta");
 assert.equal(display.marketingConsentLabel({marketingConsent:false}),"No acepta");
 assert.equal(display.marketingConsentLabel({marketingConsent:false,marketingConsentStatus:"withdrawn"}),"Retiró su permiso");
 assert.equal(display.marketingConsentLabel({marketingConsent:false,marketingConsentStatus:"granted"}),"No acepta");
});

test("sin compras, la relación se reduce a una línea en vez de tres ceros",()=>{
 assert.equal(display.hasPurchases({visitCount:0,totalSpent:"0.00",lastPurchaseAt:null}),false);
 assert.equal(display.hasPurchases({visitCount:3,totalSpent:"120.00",lastPurchaseAt:"2026-11-14T15:00:00Z"}),true);
 assert.match(detail,/hasPurchases\(customer\)\?<dl>[\s\S]*?:<p>[\s\S]*?Aún no registra compras\./);
});

test("el detalle no repite datos ni muestra vacíos como hechos",()=>{
 assert.doesNotMatch(detail,/No registrado/,"el documento vacío se omite; si existe va en la tarjeta de contacto");
 assert.match(detail,/const contactParts=\[document,/);
 assert.match(read("src/modules/customers/presentation/customers.css"),/\.customer-contact-meta>span\+span::before\{content:none\}/,"en móvil el punto medio no queda colgado");
 assert.match(detail,/<DetailContact icon=\{customer\.phone\?"phone":"user"\}/);
 assert.match(detail,/<Tag>\{segmentLabels\[customer\.segment\]/,"el segmento es un atributo: Tag neutro junto al estado");
 assert.doesNotMatch(detail,/meta=/,"el segmento ya no va como texto gris bajo el nombre");
 assert.match(detail,/channelOptions\.find/,"el canal preferido sale del catálogo del API");
 assert.match(detail,/className="modal-section-title modal-section-rule"/);
 assert.match(detail,/Sin dirección registrada\./);
});
