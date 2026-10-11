import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import {runInNewContext} from "node:vm";
const require=createRequire(import.meta.url);
const ts=require("typescript");
const source=readFileSync(new URL("../src/modules/operations/salon/domain/occupation-time.ts",import.meta.url),"utf8");
const compiled=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2022}}).outputText;
const sandboxModule={exports:{}};
runInNewContext(compiled,{module:sandboxModule,exports:sandboxModule.exports,Date,Number,Math});
const {formatTableOccupation}=sandboxModule.exports;

test("el tiempo parte de la apertura activa, no de la última edición del pedido",()=>{
 const opened="2026-10-11T10:00:00Z";
 assert.equal(formatTableOccupation(opened,Date.parse("2026-10-11T11:25:00Z")).label,"1 h 25 min");
 assert.equal(formatTableOccupation(opened,Date.parse("2026-10-11T10:00:00Z")).label,"0 min");
});
test("las mesas abiertas durante días se identifican para revisión sin liberarlas",()=>{
 const result=formatTableOccupation("2026-10-05T12:00:00Z",Date.parse("2026-10-11T10:00:00Z"));
 assert.equal(result.label,"5 d 22 h");
 assert.equal(result.needsReview,true);
 assert.match(result.description,/Revisar/);
});
test("no se inventa una antigüedad sin una hora válida",()=>{
 for(const opened of [undefined,"","invalida","2026-10-12T12:00:00Z"]){
  assert.equal(formatTableOccupation(opened,Date.parse("2026-10-11T10:00:00Z")).label,"—");
 }
});
test("la pantalla conserva el centrado, el polling parcial y estados semánticos",()=>{
 const page=readFileSync(new URL("../src/modules/operations/salon/presentation/salon-manager.tsx",import.meta.url),"utf8");
 const css=readFileSync(new URL("../src/modules/operations/styles/salon.css",import.meta.url),"utf8");
 assert.match(page,/refetchInterval:10000/);
 assert.match(page,/refetchIntervalInBackground:false/);
 assert.match(page,/formatTableOccupation\(t\.occupiedSince,clock\)/);
 assert.match(page,/setInterval\(\(\)=>setClock\(Date\.now\(\)\),30000\)/);
 assert.match(page,/floor\.isError&&!floor\.data/);
 assert.match(page,/<Status tone=\{meta\.tone\}>\{meta\.label\}<\/Status>/);
 assert.doesNotMatch(page,/<Status tone="red">Ocupada<\/Status>/);
 assert.doesNotMatch(page,/className="salon-table-order-status"/);
 assert.match(css,/\.salon-table-name-row\{[^\n]*align-items:center/);
 assert.match(css,/\.salon-table\.occupied\{[\s\S]*?border-color:var\(--danger-200\);[\s\S]*?background:var\(--surface\)/);
});

test("paleta de salon: rojo solo para ocupacion, reloj neutro e interaccion indigo",()=>{
 const css=readFileSync(new URL("../src/modules/operations/styles/salon.css",import.meta.url),"utf8");
 assert.match(css,/\.salon-table\.occupied\{[\s\S]*?background:var\(--surface\)/);
 assert.match(css,/\.salon-table\.occupied \.salon-table-icon\{background:var\(--primary-100\);color:var\(--primary-700\)\}/);
 assert.match(css,/\.salon-table-elapsed\{[^\n]*background:var\(--cloud-100\);color:var\(--ink-600\)/);
 assert.match(css,/\.salon-table-elapsed\.is-stale\{background:var\(--warning-100\);color:var\(--warning-800\)\}/);
 assert.match(css,/\.salon-table-order\{[^\n]*background:var\(--primary-600\);color:var\(--surface\)/);
 assert.match(css,/\.salon-table:hover \.salon-table-order\{background:var\(--primary-700\)\}/);
 assert.match(css,/\.salon-table\.free\{border-color:var\(--brand-200\)\}/);
});
