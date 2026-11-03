import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
const require=createRequire(import.meta.url);
const tipCalculation={};
vm.runInNewContext(ts.transpileModule(readFileSync(new URL("../src/modules/operations/tips/domain/calculation.ts",import.meta.url),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports:tipCalculation});
const exports={};
vm.runInNewContext(ts.transpileModule(readFileSync(new URL("../src/modules/operations/pos/domain/pos-schema.ts",import.meta.url),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports,require:name=>name.endsWith("tips/domain/calculation")?tipCalculation:require(name)});
const {cashChange,paymentResolverFor}=exports;
const methods=[{code:"bills",name:"Efectivo",affectsCash:true},{code:"card",name:"Tarjeta",affectsCash:false}];
const validate=(draft,remaining=10)=>paymentResolverFor(methods,remaining)({method:"bills",amount:"10",reference:"",...draft},{},{});

test("propina aceptada se suma al efectivo recibido, sin alterar el consumo",()=>{
 const policy={mode:"collect",percentage:10,allowFixed:true,decimals:2,channels:["salon"]};
 const resolve=draft=>paymentResolverFor(methods,100,policy,100)({method:"bills",amount:"50",reference:"",receivedAmount:"60",tipEnabled:true,tipKind:"percentage",...draft},{},{});
 assert.equal(Object.keys(resolve({}).errors).length,0);
 assert.equal(resolve({}).values.amount,"50");
 assert.ok(resolve({receivedAmount:"59.99"}).errors.receivedAmount);
 assert.equal(Object.keys(resolve({method:"card",receivedAmount:"",tipKind:"fixed",tipValue:"5"}).errors).length,0);
 assert.ok(resolve({tipKind:"fixed",tipValue:"-1"}).errors.tipValue);
 assert.ok(resolve({tipKind:"fixed",tipValue:"5.001"}).errors.tipValue);
 assert.equal(tipCalculation.canCollectTip(policy,"salon",true),false);
 assert.equal(tipCalculation.canCollectTip(policy,"delivery",false),false);
 assert.equal(tipCalculation.canCollectTip({...policy,mode:"suggest"},"salon",false),false);
 assert.equal(tipCalculation.canCollectTip({...policy,percentage:0,allowFixed:false},"salon",false),false);
});

test("el porcentaje de la propina sale de la configuración, no de lo escrito",()=>{
 const policy={mode:"collect",percentage:10,allowFixed:false,decimals:2,channels:["salon"]};
 const resolve=draft=>paymentResolverFor(methods,17,policy,17)({method:"bills",amount:"17",reference:"",tipEnabled:true,tipKind:"percentage",...draft},{},{});
 assert.equal(tipCalculation.tipInputValue("percentage",policy,"50"),10);
 assert.equal(tipCalculation.tipInputValue("fixed",policy,"3"),3);
 // 10 % de 17.00 = 1.70: el efectivo debe cubrir 18.70 aunque se envíe otro valor.
 assert.ok(resolve({receivedAmount:"18.69",tipValue:"0"}).errors.receivedAmount);
 assert.equal(Object.keys(resolve({receivedAmount:"18.70",tipValue:"0"}).errors).length,0);
 assert.ok(resolve({receivedAmount:"30",tipKind:"fixed",tipValue:"3"}).errors.tipValue);
 assert.ok(paymentResolverFor(methods,17,{...policy,percentage:0},17)({method:"card",amount:"17",reference:"",tipEnabled:true,tipKind:"percentage"},{},{}).errors.tipKind);
});

test("el cobro envía solo el tipo de propina porcentual y el monto libre",async()=>{
 const api={};let sent;
 vm.runInNewContext(ts.transpileModule(readFileSync(new URL("../src/modules/operations/pos/infrastructure/pos-api.ts",import.meta.url),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports:api,require:()=>({apiFetch:async(path,options)=>{sent=JSON.parse(options.body)}})});
 await api.createPayment("order",{method:"card",amount:"10",reference:"",tipEnabled:true,tipKind:"percentage",tipValue:"50"});
 assert.deepEqual(sent.tip,{kind:"percentage"});
 await api.createPayment("order",{method:"card",amount:"10",reference:"",tipEnabled:true,tipKind:"fixed",tipValue:"3"});
 assert.deepEqual(sent.tip,{kind:"fixed",value:3});
});

test("el formulario de cobro no ofrece propina entregada en mano y muestra al mozo",()=>{
 const source=readFileSync(new URL("../src/modules/operations/pos/presentation/pos-dialogs.tsx",import.meta.url),"utf8");
 assert.doesNotMatch(source,/Entregada directamente|tipDirectToStaff|Porcentaje \(%\)/);
 assert.match(source,/Atendió: <\/span><b>\{order\.waiterName\}<\/b>/);
 assert.match(source,/<div className="pos-checkout-meta">[\s\S]*Atendió: <\/span><b>\{order\.waiterName\}<\/b>[\s\S]*Caja: <\/span><b>\{shiftName\}<\/b>/,"el mozo queda en el visor fijo junto a la caja");
 assert.equal((source.match(/Atendió:/g)??[]).length,1,"el mozo no se repite dentro de la propina");
});

test("propinas en moneda sin centavos solo aceptan montos fijos enteros",()=>{
 const policy={mode:"collect",allowFixed:true,decimals:0};
 const resolve=value=>paymentResolverFor(methods,100,policy,100)({method:"card",amount:"100",reference:"",tipEnabled:true,tipKind:"fixed",tipValue:value},{},{});
 assert.ok(resolve("5.5").errors.tipValue);
 assert.equal(Object.keys(resolve("5").errors).length,0);
});

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
