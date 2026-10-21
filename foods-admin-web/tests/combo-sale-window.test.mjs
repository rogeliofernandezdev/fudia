import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url);
const root=new URL("../src/modules/menu/combos/",import.meta.url);
function compile(file,resolve=require){
 const exports={};
 vm.runInNewContext(ts.transpileModule(readFileSync(new URL(file,root),"utf8"),{compilerOptions:{target:ts.ScriptTarget.ES2022,module:ts.ModuleKind.CommonJS}}).outputText,{exports,require:resolve});
 return exports;
}
const window=compile("domain/sale-window.ts");
const blank=()=>({availableFrom:"",availableUntil:"",availableDays:[]});
const plain=value=>JSON.parse(JSON.stringify(value));

test("seleccionar un día desde cualquiera de los calendarios completa Desde y Hasta automáticamente",()=>{
 for(const field of ["availableFrom","availableUntil"]){
  const selected=window.changeSaleDate(blank(),field,"2026-10-21");
  assert.deepEqual(plain(selected),{availableFrom:"2026-10-21T00:00",availableUntil:"2026-10-21T23:59:59.999",availableDays:[]});
 }
});

test("cambiar la fecha de un menú vencido reemplaza el rango anterior sin conservar sus horas",()=>{
 const old={availableFrom:"2026-10-20T12:30",availableUntil:"2026-10-20T15:00",availableDays:[1,2,3,4,5]};
 const selected=window.changeSaleDate(old,"availableFrom","2026-10-21");
 assert.equal(selected.availableFrom,"2026-10-21T00:00");
 assert.equal(selected.availableUntil,"2026-10-21T23:59:59.999");
 assert.deepEqual(plain(selected.availableDays),old.availableDays);
 assert.equal(old.availableFrom,"2026-10-20T12:30","no modifica el dato original");
});

test("el día completo incluye la última fracción del día de Lima, pero no el día siguiente",()=>{
 const selected=window.changeSaleDate(blank(),"availableFrom","2026-10-21");
 const from=window.zonedInputToIso(selected.availableFrom,"America/Lima");
 const until=window.zonedInputToIso(selected.availableUntil,"America/Lima");
 assert.equal(from,"2026-10-21T05:00:00.000Z");
 assert.equal(until,"2026-10-22T04:59:59.999Z");
 for(const instant of [Date.parse(from),Date.parse("2026-10-21T08:00:00Z"),Date.parse(until)])assert.equal(window.isOutsideDates(from,until,instant),false);
 assert.equal(window.isOutsideDates(from,until,Date.parse(from)-1),true);
 assert.equal(window.isOutsideDates(from,until,Date.parse(until)+1),true);
});

test("días completos en una zona con cambio de horario duran 23 o 25 horas, no siempre 24",()=>{
 for(const [date,hours] of [["2026-03-29",23],["2026-10-25",25]]){
  const selected=window.changeSaleDate(blank(),"availableFrom",date);
  const from=window.zonedInputToIso(selected.availableFrom,"Europe/Madrid");
  const until=window.zonedInputToIso(selected.availableUntil,"Europe/Madrid");
  assert.equal(Date.parse(until)-Date.parse(from)+1,hours*60*60*1000);
  assert.equal(window.isoToZonedInput(from,"Europe/Madrid"),`${date}T00:00`);
  assert.equal(window.isoToZonedInput(until,"Europe/Madrid"),`${date}T23:59`);
 }
});

test("un rango de varios días, límites vacíos y Siempre siguen disponibles",()=>{
 const one=window.changeSaleDate(blank(),"availableFrom","2026-10-21");
 const range=window.changeSaleDate(one,"availableUntil","2026-10-24");
 assert.equal(range.availableFrom,"2026-10-21T00:00");
 assert.equal(range.availableUntil,"2026-10-24T23:59:59.999");
 assert.equal(window.changeSaleDate(range,"availableFrom","2026-10-22").availableUntil,range.availableUntil);
 assert.equal(window.changeSaleDate(range,"availableUntil","").availableUntil,"");
 assert.deepEqual(plain(window.applySaleWindow(range,"always")),blank());
 assert.deepEqual(plain(window.fullDaySchedule(blank())),blank());
});

test("editar convierte la fecha persistida a la zona del local antes de completar el día",()=>{
 const selected=window.fullDaySchedule({
  availableFrom:window.isoToZonedInput("2026-10-22T02:00:00Z","America/Lima"),
  availableUntil:window.isoToZonedInput("2026-10-22T03:00:00Z","America/Lima"),
  availableDays:[],
 });
 assert.equal(selected.availableFrom,"2026-10-21T00:00");
 assert.equal(selected.availableUntil,"2026-10-21T23:59:59.999");
});

test("guardar conserva los límites inclusivos completos en el contrato ISO del API",async()=>{
 let request;
 const api=compile("infrastructure/combos-api.ts",name=>{
  if(name==="../domain/sale-window")return window;
  if(name==="@/shared/api/client")return{apiFetch:async(path,options)=>{request={path,...options};return{id:"menu"}}};
  return require(name);
 });
 const schedule=window.changeSaleDate(blank(),"availableUntil","2026-10-21");
 await api.saveCombo({name:"Menú",price:"15",description:"",groups:[],...schedule},"menu","America/Lima");
 assert.equal(request.method,"PATCH");
 const body=JSON.parse(request.body);
 assert.equal(body.availableFrom,"2026-10-21T05:00:00.000Z");
 assert.equal(body.availableUntil,"2026-10-22T04:59:59.999Z");
});
