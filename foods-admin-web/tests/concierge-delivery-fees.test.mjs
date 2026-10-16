import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
const require=createRequire(import.meta.url);
const root=new URL("../src/modules/configuration/",import.meta.url);
function compile(path,resolve=require){const exports={};vm.runInNewContext(ts.transpileModule(readFileSync(new URL(path,root),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});return exports}
const fees=compile("domain/delivery-fees.ts");
const row=(key,maxDistanceKm,fee)=>({key,maxDistanceKm,fee});

test("tarifas de delivery: valida límites, decimales y distancias repetidas",()=>{
 assert.deepEqual(JSON.parse(JSON.stringify(fees.deliveryFeeErrors([row("a","2","5"),row("b","4,5","0")]))),{});
 const errors=fees.deliveryFeeErrors([row("a","0","5"),row("b","51","5"),row("c","2.555","5"),row("d","3","-1"),row("e","3","1000"),row("f","3","x")]);
 assert.ok(errors.a.maxDistanceKm);assert.ok(errors.b.maxDistanceKm);assert.ok(errors.c.maxDistanceKm);
 assert.equal(errors.d.maxDistanceKm,undefined);assert.ok(errors.d.fee);
 assert.match(errors.e.maxDistanceKm,/ya existe/);assert.ok(errors.e.fee);assert.ok(errors.f.fee);
});

test("tarifas de delivery: payload numérico ordenado por distancia y sin datos del local",()=>{
 const payload=fees.deliveryFeesPayload([row("a","6","12"),row("b","2","5"),row("c","4","8.50")]);
 assert.deepEqual(JSON.parse(JSON.stringify(payload)),{tiers:[{maxDistanceKm:2,fee:5},{maxDistanceKm:4,fee:8.5},{maxDistanceKm:6,fee:12}]});
 assert.deepEqual(JSON.parse(JSON.stringify(fees.deliveryFeesPayload([]))),{tiers:[]});
 assert.throws(()=>fees.deliveryFeesPayload([row("a","0","5")]),/Revisa/);
 assert.throws(()=>fees.deliveryFeesPayload(Array.from({length:11},(_,i)=>row(String(i),String(i+1),"1"))),/máximo 10/);
});

test("tarifas de delivery: filas editables desde la respuesta del backend",()=>{
 const rows=fees.deliveryFeeRows([{maxDistanceKm:"2.00",fee:"5.00"},{maxDistanceKm:"4.50",fee:"8.50"}]);
 assert.deepEqual(rows.map(item=>[item.maxDistanceKm,item.fee]),[["2","5.00"],["4.5","8.50"]]);
 assert.equal(new Set(rows.map(item=>item.key)).size,2);
});

test("tarifas de delivery: la API usa el endpoint de Concierge con GET y PUT",async()=>{
 const calls=[];
 const api=compile("infrastructure/concierge-settings-api.ts",name=>name==="@/shared/api/client"?{apiFetch:async(path,options)=>{calls.push({path,options});return{tiers:[]}}}:require(name));
 await api.getDeliveryFees();
 await api.saveDeliveryFees({tiers:[{maxDistanceKm:2,fee:5}]});
 assert.equal(calls[0].path,"concierge-settings/delivery-fees");assert.equal(calls[0].options,undefined);
 assert.equal(calls[1].path,"concierge-settings/delivery-fees");assert.equal(calls[1].options.method,"PUT");
 assert.deepEqual(JSON.parse(calls[1].options.body),{tiers:[{maxDistanceKm:2,fee:5}]});
});

test("tarifas de delivery: el proxy de administración reenvía PUT",()=>{
 const route=readFileSync(new URL("../src/app/api/admin/[...path]/route.ts",import.meta.url),"utf8");
 assert.match(route,/export const PUT=proxy/);
});

test("tarifas de delivery: la pantalla tiene carga, error, vacío y permisos",()=>{
 const panel=readFileSync(new URL("presentation/delivery-fees-panel.tsx",root),"utf8");
 for(const fragment of ["DeliveryFeesSkeleton","ErrorState","Sin rangos configurados","can(\"organizations.manage\")","Falta la ubicación del local","pageRoutes.locations"])assert.ok(panel.includes(fragment),fragment);
 const page=readFileSync(new URL("presentation/concierge-settings-page.tsx",root),"utf8");
 assert.match(page,/<DeliveryFeesPanel\/>/);
});
