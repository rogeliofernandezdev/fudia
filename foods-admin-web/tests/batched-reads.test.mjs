import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url),root=new URL("../",import.meta.url);
const read=file=>readFileSync(new URL(file,root),"utf8");
function compile(file,resolve){
  const exports={};
  vm.runInNewContext(ts.transpileModule(read(file),{compilerOptions:{target:ts.ScriptTarget.ES2022,module:ts.ModuleKind.CommonJS}}).outputText,{exports,require:resolve,URLSearchParams});
  return exports;
}
const params=path=>new URLSearchParams(path.split("?")[1]);

test("los menús elegidos en la comanda se piden juntos, de a 20 por petición",async()=>{
  const calls=[];
  const api=compile("src/modules/operations/salon/infrastructure/catalog-api.ts",name=>name==="@/shared/api/client"?{apiFetch:async path=>{calls.push(path);return{items:params(path).get("ids").split(",").map(id=>({id}))}}}:require(name));
  const ids=Array.from({length:21},(_,index)=>`combo-${index}`);
  const items=await api.listOrderComboDetails(ids);
  assert.equal(JSON.stringify(items.map(item=>item.id)),JSON.stringify(ids),"conserva el orden de la comanda");
  assert.equal(calls.length,2);
  assert.ok(calls.every(path=>path.startsWith("order-combo-details?")));
  assert.equal(params(calls[0]).get("ids").split(",").length,20);
});

test("un menú que falla hace fallar el lote, como antes Promise.all",async()=>{
  const api=compile("src/modules/operations/salon/infrastructure/catalog-api.ts",name=>name==="@/shared/api/client"?{apiFetch:async()=>{throw new Error("combo_unavailable")}}:require(name));
  await assert.rejects(api.listOrderComboDetails(["a"]),/combo_unavailable/);
});

test("el editor de menús refresca solo las opciones elegidas, activas y de a 100",async()=>{
  const calls=[];
  const api=compile("src/modules/menu/combos/infrastructure/combos-api.ts",name=>name==="../domain/sale-window"?compile("src/modules/menu/combos/domain/sale-window.ts",require):name==="@/shared/api/client"?{apiFetch:async path=>{calls.push(path);return{items:params(path).get("productIds").split(",").map(id=>({id}))}}}:require(name));
  const ids=Array.from({length:101},(_,index)=>`p-${index}`);
  const result=await api.listComboProductsByIds(ids);
  assert.equal(result.items.length,101);assert.equal(JSON.stringify(result.ids),JSON.stringify(ids));
  assert.equal(calls.length,2);
  assert.ok(calls.every(path=>path.startsWith("products?")&&params(path).get("status")==="active"));
  assert.equal(params(calls[0]).get("productIds").split(",").length,100);
});

test("la comanda usa el lote de menús y el catálogo con varias páginas no las vuelve a pedir todas",()=>{
  const view=read("src/modules/operations/orders/presentation/comanda-view.tsx");
  assert.ok(view.includes("listOrderComboDetails(selectedComboIds)"));
  assert.ok(!view.includes("Promise.all(selectedComboIds.map"));
  const catalog=read("src/modules/operations/salon/presentation/comanda-catalog.tsx");
  assert.match(catalog,/refetchInterval:query=>\(query\.state\.data\?\.pages\.length\?\?1\)>1\?false:10000/);
  assert.ok(catalog.includes('queryKey:["order-catalog-availability"'));
});
