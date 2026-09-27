import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import vm from "node:vm";
import ts from "typescript";

const source=readFileSync(new URL("../src/modules/setup/presentation/setup-steps.ts",import.meta.url),"utf8");
const compiled=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText;
const context={exports:{}};
vm.runInNewContext(compiled,context);
const {buildSetupSteps,setupGuideHref,setupOverviewHref}=context.exports;
const empty={serviceMode:"",completedAt:null,coreReady:false,counts:{categories:0,products:0,tables:0,cashRegisters:0,users:1,inventoryItems:0,recipes:0,suppliers:0},modules:{}};
const keys=setup=>Array.from(buildSetupSteps(setup),step=>step.key);

test("mostrador no exige mesas; salón y operación mixta sí",()=>{
  assert.equal(keys({...empty,serviceMode:"counter"}).includes("tables"),false);
  for(const serviceMode of ["dine_in","mixed"]){
    const tableStep=buildSetupSteps({...empty,serviceMode}).find(step=>step.key==="tables");
    assert.equal(tableStep.required,true);
    assert.equal(tableStep.complete,false);
  }
});

test("la guía de un plan básico excluye módulos no contratados",()=>{
  assert.deepEqual(keys(empty),["operation","catalog","cash","team","review"]);
  assert.equal(keys({...empty,modules:{recetas:true}}).includes("recipes"),false);
  assert.deepEqual(keys({...empty,modules:{inventario:true,recetas:true,compras:true}}),["operation","catalog","cash","team","inventory","recipes","purchases","review"]);
});

test("la carta exige categoría y producto; recorrer pasos no altera requisitos",()=>{
  for(const counts of [{categories:1,products:0},{categories:0,products:1}]){
    const steps=buildSetupSteps({...empty,counts:{...empty.counts,...counts}});
    assert.equal(steps.find(step=>step.key==="catalog").complete,false);
  }
  const setup={...empty,counts:{...empty.counts,categories:1,products:1,cashRegisters:1}};
  const before=JSON.stringify(setup);
  const steps=buildSetupSteps(setup);
  assert.equal(steps.find(step=>step.key==="catalog").complete,true);
  assert.equal(steps.find(step=>step.key==="review").complete,false);
  assert.equal(JSON.stringify(setup),before);
});

test("los enlaces de guía abren la sección real y el resumen conserva el paso",()=>{
  const step=buildSetupSteps(empty).find(step=>step.key==="catalog");
  const href=setupGuideHref(step);
  assert.equal(href.pathname,"/productos");
  assert.equal(href.query.guia,"puesta-en-marcha");
  assert.equal(href.query.paso,"catalog");
  assert.equal(setupOverviewHref(step.key),"/configuracion/puesta-en-marcha?paso=catalog");
});
