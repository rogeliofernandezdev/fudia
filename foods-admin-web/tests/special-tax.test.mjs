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
const tax=compile("shared/fiscal/special-tax.ts");
const format=value=>String(value).replace(".",",");
const mype={code:"pe_mype_restaurants",country:"PE",name:"IGV reducido MYPE (restaurantes y hoteles)",rates:[
 {id:"a",validFrom:"2025-01-01",validTo:"2026-01-01",rate:"0.100000",legalBasis:"Ley 32219"},
 {id:"b",validFrom:"2026-01-01",validTo:"2027-01-01",rate:"0.105000",legalBasis:"Leyes 32219 y 32387"},
 {id:"c",validFrom:"2027-01-01",validTo:"2028-01-01",rate:"0.150000",legalBasis:"Leyes 32219 y 32387"},
]};
const plain=value=>JSON.parse(JSON.stringify(value));

test("el IGV reducido se muestra como un solo porcentaje legible",()=>{
 assert.equal(tax.ratePercentLabel("0.105000",format),"10,5 %");
 assert.equal(tax.ratePercentLabel("0.180000",format),"18 %");
 for(const [percent,rate] of [["10.5","0.105000"],["15","0.150000"],["0","0.000000"],["100","1.000000"],["12.3456","0.123456"]])assert.equal(tax.percentToRate(percent),rate,percent);
 assert.equal(tax.rateToPercent("0.105000"),"10.5");
});

test("el estado avisa 30 días antes de que terminen las vigencias registradas",()=>{
 assert.deepEqual(plain(tax.specialTaxStatusOn(mype,"2026-11-01")),{state:"current",rate:"0.105000",rateValidTo:"2027-01-01",coverageEnd:"2028-01-01"});
 assert.equal(tax.specialTaxStatusOn(mype,"2027-12-01").state,"current");
 assert.equal(tax.specialTaxStatusOn(mype,"2027-12-02").state,"expiring");
 assert.deepEqual(plain(tax.specialTaxStatusOn(mype,"2028-01-01")),{state:"expired",rate:null,rateValidTo:null,coverageEnd:null});
 const gap={...mype,rates:mype.rates.slice(0,2)};
 assert.equal(tax.specialTaxStatusOn(gap,"2026-12-15").coverageEnd,"2027-01-01");
 assert.equal(tax.lastDay("2028-01-01"),"2027-12-31");
});

test("cada vigencia se clasifica como vigente, programada o finalizada",()=>{
 assert.deepEqual(mype.rates.map(rate=>tax.ratePeriodState(rate,"2026-11-01")),["ended","current","scheduled"]);
 assert.equal(tax.regimesForCountry([mype],"PE").length,1);
 assert.equal(tax.regimesForCountry([mype],"CO").length,0);
});

test("un periodo de año completo se nombra por su año",()=>{
 const day=iso=>iso.split("-").reverse().join("/");
 assert.equal(tax.ratePeriodLabel(mype.rates[1],day),"2026");
 assert.equal(tax.ratePeriodLabel({validFrom:"2028-03-01",validTo:"2029-01-01"},day),"01/03/2028 – 31/12/2028");
 assert.equal(tax.ratePeriodLabel({validFrom:"2028-01-01",validTo:null},day),"Desde 01/01/2028");
});
