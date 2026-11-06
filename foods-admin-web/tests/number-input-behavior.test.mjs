import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
const require=createRequire(import.meta.url),exports={};
const source=readFileSync(new URL("../src/design-system/number-input-behavior.tsx",import.meta.url),"utf8");
vm.runInNewContext(ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require});
const {stepNumberValue}=exports;
test("las flechas parten de 1.0 y conservan la fracción al avanzar una unidad",()=>{
 assert.equal(stepNumberValue("",1,{min:"0.001",decimal:true}),"1.0");
 assert.equal(stepNumberValue("0",1,{decimal:true}),"1.0");
 assert.equal(stepNumberValue("0.004",1,{decimal:true}),"1.004");
 assert.equal(stepNumberValue("0.0005",1,{decimal:true}),"1.0005");
 assert.equal(stepNumberValue("1.25",-1,{decimal:true}),"0.25");
 assert.equal(stepNumberValue("2",1),"3");
});
test("respeta mínimos, máximos y valores inválidos sin errores de coma flotante",()=>{
 assert.equal(stepNumberValue("0.5",-1,{min:"0.001",decimal:true}),"0.001");
 assert.equal(stepNumberValue("99.5",1,{max:"100",decimal:true}),"100.0");
 assert.equal(stepNumberValue("0.1",1,{decimal:true}),"1.1");
 assert.equal(stepNumberValue("1.1",-1,{decimal:true}),"0.1");
 assert.equal(stepNumberValue("1e-3",1,{decimal:true}),"1.001");
 assert.equal(stepNumberValue("2147483646",1),"2147483647");
 assert.equal(stepNumberValue("invalid",1),"invalid");
 assert.equal(stepNumberValue("1",1,{min:"5",max:"3"}),"1");
});
