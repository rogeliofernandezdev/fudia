import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url),root=new URL("../src/",import.meta.url);
function compile(path){
 const exports={};
 vm.runInNewContext(ts.transpileModule(readFileSync(new URL(path,root),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports,require});
 return exports;
}
const regimes=compile("modules/organizations/domain/tax-regimes.ts");

test("los porcentajes del formulario se envían como fracción exacta de seis decimales",()=>{
 for(const [percent,rate] of [["8","0.080000"],["2.5","0.025000"],["18","0.180000"],["0.5","0.005000"],["100","1.000000"],["12.3456","0.123456"],["0","0.000000"]])assert.equal(regimes.percentToRate(percent),rate,percent);
 for(const [rate,percent] of [["0.080000","8"],["0.025000","2.5"],["0.123456","12.3456"],["1.000000","100"]])assert.equal(regimes.rateToPercent(rate),percent,rate);
 assert.equal(regimes.totalPercent([{percent:"8"},{percent:"2.5"}]),10.5);
 assert.equal(regimes.totalPercent([{percent:"0.1"},{percent:"0.2"}]),0.3);
});

test("«Hasta» es inclusiva en pantalla y exclusiva en el API",()=>{
 const input=regimes.regimeInput({name:" Régimen MYPE 2026 ",validFrom:"2026-01-01",validUntil:"2026-12-31",components:[{name:"IGV",percent:"8"},{name:"IPM",percent:"2.5"}]});
 assert.deepEqual(JSON.parse(JSON.stringify(input)),{name:"Régimen MYPE 2026",validFrom:"2026-01-01",validTo:"2027-01-01",components:[{code:"igv",name:"IGV",rate:"0.080000"},{code:"ipm",name:"IPM",rate:"0.025000"}]});
 assert.equal(regimes.regimeInput({name:"Sin fin",validFrom:"2028-02-28",validUntil:"",components:[{name:"IGV",percent:"18"}]}).validTo,null);
 const draft=regimes.draftFromRegime({id:"r1",name:"MYPE",validFrom:"2026-01-01",validTo:"2027-01-01",components:[{code:"igv",name:"IGV",rate:"0.080000"}],totalRate:"0.080000"});
 assert.deepEqual(JSON.parse(JSON.stringify(draft)),{id:"r1",name:"MYPE",validFrom:"2026-01-01",validUntil:"2026-12-31",components:[{name:"IGV",percent:"8"}]});
 assert.equal(regimes.shiftDate("2028-02-28",1),"2028-02-29");
 assert.equal(regimes.shiftDate("2027-03-01",-1),"2027-02-28");
});

test("el código del impuesto se deriva del nombre, sin pedirlo al usuario",()=>{
 const used=new Set();
 assert.equal(regimes.componentCode("Impuesto de Promoción Municipal",used),"impuesto_de_prom");
 assert.equal(regimes.componentCode("IGV",used),"igv");
 assert.equal(regimes.componentCode("igv",used),"igv_2");
 assert.equal(regimes.componentCode("10% servicio",used),"t_10_servicio");
 for(const code of used)assert.match(code,/^[a-z][a-z0-9_]{0,19}$/);
});

test("el estado usa la fecha de negocio del API, no el reloj del navegador",()=>{
 const regime={validFrom:"2026-01-01",validTo:"2027-01-01"};
 assert.equal(regimes.regimeStatus(regime,"2025-12-31"),"scheduled");
 assert.equal(regimes.regimeStatus(regime,"2026-01-01"),"current");
 assert.equal(regimes.regimeStatus(regime,"2027-01-01"),"ended");
 assert.equal(regimes.regimeStatus({validFrom:"2026-01-01",validTo:null},"2099-01-01"),"current");
 const source=readFileSync(new URL("modules/organizations/presentation/tax-regimes.tsx",root),"utf8");
 assert.ok(source.includes("regimeStatus(regime,view.effective.date)"));
 assert.ok(!source.includes("new Date()"));
});
