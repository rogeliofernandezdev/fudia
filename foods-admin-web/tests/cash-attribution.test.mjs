import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import vm from "node:vm";
import ts from "typescript";

const source=readFileSync(new URL("../src/modules/operations/cash/domain/shift-attribution.ts",import.meta.url),"utf8");
const context={exports:{}};
vm.runInNewContext(ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,context);
const {cashShiftAttribution}=context.exports;

test("retirar a Elmer deja el equipo vacío sin usar al abridor como cajero",()=>{
  const result=cashShiftAttribution({status:"open",openedByName:"Elmer",closedByName:"",activeUserNames:[]});
  assert.equal(result.label,"Equipo actual");
  assert.equal(result.name,"Sin usuarios asignados");
});
test("el equipo muestra todas las personas asignadas y refleja un reemplazo",()=>{
  const shift={status:"open",openedByName:"Jimena",closedByName:"",activeUserNames:["Jimena","Edith"]};
  assert.equal(cashShiftAttribution(shift).name,"Jimena, Edith");
  assert.equal(cashShiftAttribution({...shift,activeUserNames:["Edith"]}).name,"Edith");
});
test("si Jimena abre y Edith cierra, el historial atribuye el cierre a Edith",()=>{
  const result=cashShiftAttribution({status:"closed",openedByName:"Jimena",closedByName:"Edith",activeUserNames:[]});
  assert.equal(result.label,"Cerrado por");
  assert.equal(result.name,"Edith");
});
test("un cierre sin autor no se atribuye falsamente a quien abrió",()=>{
  assert.equal(cashShiftAttribution({status:"closed",openedByName:"Jimena",closedByName:"",activeUserNames:[]}).name,"Sin registro de cierre");
});
test("un API anterior sin equipo no se interpreta como equipo vacío",()=>{
  assert.equal(cashShiftAttribution({status:"open",openedByName:"Elmer",closedByName:""}).name,"Equipo no disponible");
});
