import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url);
const compile=(name,imports={})=>{
 const exports={};
 const source=readFileSync(new URL(`../src/modules/delivery-panel/domain/${name}.ts`,import.meta.url),"utf8");
 vm.runInNewContext(ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2022}}).outputText,{exports,require:path=>imports[path]??require(path)});
 return exports;
};
const dispatch=compile("dispatch",{"./panel":compile("panel")});
const order=(token,changes={})=>({token,stage:"accepted",courierName:"",orderCode:`PED-${token}`,customerName:"Rosa Quispe",organizationName:"Restaurante",locationName:"San Miguel",address:"Av. La Marina",pickupAlert:"",pickupAlertAcknowledged:false,...changes});

test("despacho prioriza incidencias, ofertas y asignaciones pendientes sin mutar el API",()=>{
 const orders=[order("assigned",{courierName:"Luis"}),order("unassigned"),order("new",{stage:"offer"}),order("alert",{pickupAlert:"urgent"}),order("delivered",{stage:"delivered"})];
 assert.deepEqual(Array.from(dispatch.dispatchOrders(orders,"all",""),o=>o.token),["alert","new","unassigned","assigned"]);
 assert.equal(orders[0].token,"assigned");
 assert.deepEqual(Array.from(dispatch.dispatchOrders(orders,"unassigned",""),o=>o.token),["alert","unassigned"]);
 assert.deepEqual(Array.from(dispatch.dispatchOrders(orders,"all","",true),o=>o.token),["alert"]);
});

test("búsqueda por nombre, local, dirección o pedido ignora tildes y espacios",()=>{
 const orders=[order("1",{courierName:"José   Pérez"}),order("2",{locationName:"Jesús María"}),order("3",{address:"Av. Perú"})];
 for(const [search,token] of [["jose perez","1"],[" JESUS MARIA ","2"],["peru","3"],["PED-1","1"]]){
  assert.deepEqual(Array.from(dispatch.dispatchOrders(orders,"all",search),o=>o.token),[token]);
 }
 assert.equal(dispatch.dispatchOrders(orders,"all","no existe").length,0);
});

test("en camino e historial excluyen otras etapas; historial comienza por la última entrega",()=>{
 const orders=[order("a"),order("b",{stage:"picked_up",issueReportedAt:"2026-11-14T12:00:00Z"}),order("c",{stage:"delivered",deliveredAt:"2026-11-14T12:00:00Z"}),order("d",{stage:"delivered",deliveredAt:"2026-11-14T13:00:00Z"})];
 assert.deepEqual(Array.from(dispatch.dispatchOrders(orders,"on_route",""),o=>o.token),["b"]);
 assert.deepEqual(Array.from(dispatch.dispatchOrders(orders,"delivered",""),o=>o.token),["d","c"]);
 assert.equal(dispatch.needsAttention(orders[1]),true);
 assert.equal(dispatch.needsAssignment({...orders[0],courierName:"  "}),true);
});

test("carga del motorizado cuenta solo repartos activos y agrupa nombres normalizados",()=>{
 const result=dispatch.courierWorkload([order("1",{courierName:"José Pérez"}),order("2",{stage:"picked_up",courierName:" JOSE  PEREZ "}),order("3",{stage:"delivered",courierName:"Ana"}),order("4",{stage:"offer",courierName:"Ana"}),order("5"),order("6",{courierName:"Ana"})]);
 assert.deepEqual(Array.from(result,c=>({...c})),[{name:"Ana",collecting:1,onRoute:0},{name:"José Pérez",collecting:1,onRoute:1}]);
});

test("filtrar por responsable no incluye pedidos cuyo cliente tiene el mismo nombre",()=>{
 const orders=[order("1",{courierName:"José Pérez"}),order("2",{courierName:"Ana",customerName:"José Pérez"})];
 assert.deepEqual(Array.from(dispatch.dispatchOrders(orders,"all","",false," JOSE PEREZ "),o=>o.token),["1"]);
});

test("carga progresiva agrega lotes sin sustituir ni repetir los pedidos ya visibles",()=>{
 const items=Array.from({length:23},(_,index)=>order(String(index)));
 const first=dispatch.dispatchWindow(items),second=dispatch.dispatchWindow(items,20),last=dispatch.dispatchWindow(items,30);
 assert.equal(first.total,23);
 assert.equal(first.items.length,10);
 assert.equal(second.items.length,20);
 assert.equal(last.items.length,23);
 assert.equal(first.hasMore,true);assert.equal(second.hasMore,true);assert.equal(last.hasMore,false);
 assert.deepEqual(Array.from(second.items.slice(0,10),o=>o.token),Array.from(first.items,o=>o.token));
 assert.equal(new Set(last.items.map(o=>o.token)).size,23);
 assert.equal(items.length,23,"mostrar más no modifica la respuesta remota");
});

test("carga progresiva ajusta el final y aplica filtros a toda la colección",()=>{
 const empty=dispatch.dispatchWindow([],20);
 assert.equal(empty.total,0);assert.equal(empty.items.length,0);assert.equal(empty.hasMore,false);
 const items=Array.from({length:25},(_,index)=>order(String(index),{stage:index<12?"picked_up":"accepted"}));
 const filtered=dispatch.dispatchOrders(items,"on_route","");
 const expanded=dispatch.dispatchWindow(filtered,20);
 assert.equal(expanded.total,12);assert.equal(expanded.items.length,12);assert.equal(expanded.hasMore,false);
 assert.equal(dispatch.dispatchWindow(items,Number.NaN).items.length,10);
 assert.equal(dispatch.dispatchWindow(items,-5).items.length,10);
 assert.equal(dispatch.dispatchWindow(items.slice(0,3),20).items.length,3);
});
