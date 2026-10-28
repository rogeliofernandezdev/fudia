import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
const require=createRequire(import.meta.url);
const exports={};
vm.runInNewContext(ts.transpileModule(readFileSync(new URL("../src/modules/operations/pos/domain/pos-schema.ts",import.meta.url),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports,require});
const {cashChange,paymentResolverFor}=exports;
const methods=[{code:"bills",name:"Efectivo",affectsCash:true},{code:"card",name:"Tarjeta",affectsCash:false}];
const validate=(draft,remaining=10)=>paymentResolverFor(methods,remaining)({method:"bills",amount:"10",reference:"",...draft},{},{});

test("el vuelto se calcula sobre el cobro actual y redondea a centavos",()=>{
 assert.equal(cashChange("10","20"),10);
 assert.equal(cashChange("10","10"),0);
 assert.equal(cashChange("5","20"),15);
 assert.equal(cashChange("0.10","0.30"),0.2);
 for(const [amount,received] of [["10","9"],["10",""],["10","invalid"],["","20"],["-1","20"]])assert.equal(cashChange(amount,received),null);
});

test("efectivo requiere un importe recibido válido que cubra el cobro",()=>{
 for(const receivedAmount of ["",undefined," ","0","-1","NaN","Infinity","9.99"]){
  assert.ok(validate({receivedAmount}).errors.receivedAmount,receivedAmount);
 }
 for(const receivedAmount of ["10","20"]){
  const result=validate({receivedAmount});
  assert.equal(Object.keys(result.errors).length,0);
  assert.equal(result.values.amount,"10");
 }
 assert.ok(validate({receivedAmount:"20",amount:"11"}).errors.amount);
 assert.equal(Object.keys(validate({receivedAmount:"20",amount:"5"}).errors).length,0);
});

test("medios sin impacto en caja no requieren efectivo ni aplican el vuelto",()=>{
 assert.equal(Object.keys(validate({method:"card"}).errors).length,0);
 assert.equal(Object.keys(validate({method:"card",receivedAmount:"invalid"}).errors).length,0);
});

test("el pago envía el monto cobrado y excluye efectivo recibido y vuelto del API",async()=>{
 const api={};let sent;
 vm.runInNewContext(ts.transpileModule(readFileSync(new URL("../src/modules/operations/pos/infrastructure/pos-api.ts",import.meta.url),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports:api,require:()=>({apiFetch:async(path,options)=>{sent={path,body:JSON.parse(options.body)}}})});
 await api.createPayment("order",{method:"bills",amount:"10",receivedAmount:"20",reference:""});
 assert.equal(sent.path,"payments");
 assert.deepEqual(sent.body,{orderId:"order",method:"bills",amount:10,reference:""});
});
