import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import {runInNewContext} from "node:vm";
const require=createRequire(import.meta.url);
const ts=require("typescript");
const source=readFileSync(new URL("../src/modules/operations/salon/domain/occupation-time.ts",import.meta.url),"utf8");
const compiled=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2022}}).outputText;
const module={exports:{}};
runInNewContext(compiled,{module,exports:module.exports,Date,Number,Math});
const {formatTableOccupation}=module.exports;

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
 assert.match(page,/formatTableOccupation\(t\.occupiedSince,Date\.now\(\)\)/);
 assert.match(page,/floor\.isError&&!floor\.data/);
 assert.match(page,/<Status tone="red">Ocupada<\/Status>/);
 assert.match(css,/\.salon-table-name-row\{[^\n]*align-items:center/);
 assert.match(css,/\.salon-table\.occupied\{border-color:var\(--danger-200\);background:var\(--danger-50\)/);
});
