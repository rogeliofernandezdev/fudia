import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import React from "react";
import {renderToStaticMarkup} from "react-dom/server";

const read=path=>readFileSync(new URL(`../${path}`,import.meta.url),"utf8");
const require=createRequire(import.meta.url);
const exports={};
vm.runInNewContext(ts.transpileModule(read("src/design-system/page-header.tsx"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,esModuleInterop:true}}).outputText,{exports,require:name=>name.endsWith(".css")?{}:name==="next/link"?{}:name==="@/design-system/icons"?{Icon:({name})=>React.createElement("svg",{"data-icon":name})}:require(name)});
const {Button}=exports;
const render=props=>renderToStaticMarkup(React.createElement(Button,props,"Acción"));

test("la intención fija el color e icono, sin inferirlos de la etiqueta",()=>{
 for(const [action,kind,icon] of [["cancel","ghost","close"],["cancel-record","danger-outline","cancel"],["continue","primary","chevron"],["back","ghost","chevronLeft"]]){
  const html=render({action,kind:"secondary",icon:"plus"});
  assert.ok(html.includes(`class="button ${kind}"`));
  assert.ok(html.includes(`data-icon="${icon}"`));
  assert.ok(!html.includes(" action="));
 }
 assert.ok(render({kind:"secondary"}).includes('class="button secondary"'));
 assert.match(read("src/design-system/modal.tsx"),/<Button action="cancel"/);
});

test("Continuar lleva flecha al final; Cancelar conserva icono inicial",()=>{
 const next=render({action:"continue"}),cancel=render({action:"cancel"});
 assert.ok(next.indexOf("<span>")<next.indexOf("<svg"));
 assert.ok(cancel.indexOf("<svg")<cancel.indexOf("<span>"));
});

test("la intención no altera envío, accesibilidad ni bloqueo de peticiones",()=>{
 const onClick=()=>{};
 const element=Button({action:"continue",type:"submit",loading:true,onClick,"aria-label":"Continuar al catálogo",children:"Continuar"});
 assert.equal(element.props.type,"submit");
 assert.equal(element.props.disabled,true);
 assert.equal(element.props["aria-busy"],true);
 assert.equal(element.props["aria-label"],"Continuar al catálogo");
 assert.equal(element.props.onClick,onClick);
 assert.equal(element.props.action,undefined);
});

test("formularios y anulación de registros consumen intenciones distintas",()=>{
 const manual=read("src/modules/operations/orders/presentation/manual-order-dialog.tsx");
 assert.match(manual,/<Button action="cancel"/);
 assert.match(manual,/<Button type="submit" action="continue"/);
 assert.match(read("src/modules/operations/orders/presentation/order-detail-dialog.tsx"),/<Button action="cancel-record"/);
 assert.match(read("src/modules/supply/purchases/presentation/purchases-page.tsx"),/<Button action="cancel-record" onClick=\{cancel\}/);
 const confirm=read("src/design-system/confirm-dialog.tsx");
 assert.match(confirm,/<Button kind="ghost"[^>]*onClick=\{onCancel\}>Cancelar<\/Button>/);
 assert.match(confirm,/<Button kind=\{tone==="danger"\?"danger":"primary"\}/);
 const css=read("src/styles/globals.css");
 assert.match(css,/\.button\.danger-outline\{color:var\(--danger\);border-color:var\(--danger-200\);background:var\(--danger-50\)\}/);
 const local=read("src/modules/operations/orders/presentation/order-detail.css");
 assert.ok(!/\.order-detail-cancel[^}]*\{[^}]*\b(?:color|background|border-color):/.test(local));
});
