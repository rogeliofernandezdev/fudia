import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import vm from "node:vm";
import ts from "typescript";

const source=readFileSync(new URL("../src/modules/operations/cash/domain/shift-attribution.ts",import.meta.url),"utf8");
const context={exports:{}};
vm.runInNewContext(ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,context);
const {cashShiftAttribution,cashShiftTeamName}=context.exports;

test("el cajero del turno activo es quien lo abrió, separado del equipo",()=>{
  const shift={status:"open",openedByName:"Elmer",closedByName:"",activeUserNames:["Elmer","Edith"]};
  assert.equal(cashShiftAttribution(shift).label,"Cajero");
  assert.equal(cashShiftAttribution(shift).name,"Elmer");
  assert.equal(cashShiftTeamName(shift),"Elmer, Edith");
});
test("cerrar el turno de Elmer y abrir el de Edith muestra a Edith como cajero",()=>{
  const previous={status:"closed",openedByName:"Elmer",closedByName:"Elmer",activeUserNames:[]};
  const next={status:"open",openedByName:"Edith",closedByName:"",activeUserNames:["Edith"]};
  assert.equal(cashShiftAttribution(previous).label,"Cerrado por");
  assert.equal(cashShiftAttribution(previous).name,"Elmer");
  assert.equal(cashShiftAttribution(next).label,"Cajero");
  assert.equal(cashShiftAttribution(next).name,"Edith");
});
test("retirar integrantes actualiza el equipo sin transferir la responsabilidad del turno",()=>{
  const shift={status:"open",openedByName:"Jimena",closedByName:"",activeUserNames:["Edith"]};
  assert.equal(cashShiftAttribution(shift).name,"Jimena");
  assert.equal(cashShiftTeamName(shift),"Edith");
  assert.equal(cashShiftTeamName({...shift,activeUserNames:[]}),"Sin usuarios asignados");
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
  assert.equal(cashShiftTeamName({}),"Equipo no disponible");
});
