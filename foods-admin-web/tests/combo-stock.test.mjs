import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import vm from "node:vm";
import ts from "typescript";
const exports={};vm.runInNewContext(ts.transpileModule(readFileSync(new URL("../src/modules/operations/orders/domain/draft-availability.ts",import.meta.url),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports});
const{lineAvailability,productUsage}=exports;
const option={productId:"aji",name:"Ají de gallina",available:true,remaining:3,quotaRemaining:3};
const combo=(id="menu",options=[option])=>({id,name:id,groups:[{id:"segundo",name:"Segundo",options}]});
const line=(id="menu",qty=1)=>({lineKey:id,itemType:"combo",productId:id,name:id,qty,selections:[{productId:"aji",groupId:"segundo",groupName:"Segundo",name:"Ají de gallina"}]});
const dish=(qty=1)=>({lineKey:"plato",itemType:"product",productId:"aji",name:"Ají de gallina",qty,selections:[]});
test("un menú con tres porciones no puede aumentarse a ocho aunque el menú no controle cantidad",()=>{
 const current=line("menu",1);assert.equal(lineAvailability(current,[current],[],{}, {menu:combo()}).max,3);
 current.qty=8;assert.equal(lineAvailability(current,[current],[],{}, {menu:combo()}).max,3);
});
test("platos sueltos y distintos combos comparten las mismas existencias",()=>{
 const a=line("a",2),b=line("b"),plain=dish(1),details={a:combo("a"),b:combo("b")};
 assert.equal(lineAvailability(a,[a,plain],[],{},details).max,2);
 assert.equal(lineAvailability(b,[a,plain,b],[],{},details).max,0);
 assert.equal(productUsage([a,b,plain]).aji,4);
 assert.equal(lineAvailability(plain,[a,plain],[],{aji:{remaining:3,availabilityStatus:"available"}},details).max,1);
});
test("cupo del menú se comparte entre configuraciones distintas del mismo combo",()=>{
 const a=line("menu",2),b={...line(),lineKey:"otra-configuracion"};
 assert.equal(lineAvailability(b,[a,b],[],{}, {menu:combo()}).max,1);
});
test("cero porciones, cuota cero y stock fraccionario inferior a una unidad bloquean agregar",()=>{
 for(const values of [{remaining:0},{quotaRemaining:0},{remaining:.75}]){
  const item=line();assert.equal(lineAvailability(item,[item],[],{}, {menu:combo("menu",[{...option,available:false,...values}])}).max,0);
 }
});
test("un producto elegido en dos partes consume dos unidades por cada menú",()=>{
 const item=line();item.selections.push({...item.selections[0],groupId:"extra",groupName:"Extra"});
 const detail=combo();detail.groups.push({id:"extra",name:"Extra",options:[{...option,quotaRemaining:null}]});
 assert.equal(lineAvailability(item,[item],[],{}, {menu:detail}).max,1);
});
test("editar suma solo lo ya reservado por ese pedido sin duplicar existencias",()=>{
 const initial={...line("menu",3),sourceItemId:"guardado"};const detail=combo("menu",[{...option,remaining:0,quotaRemaining:0,available:false}]);
 assert.equal(lineAvailability(initial,[initial],[initial],{}, {menu:detail}).max,3);
 const added={...line("menu",1),lineKey:"nuevo"};assert.equal(lineAvailability(added,[initial,added],[initial],{}, {menu:detail}).max,0);
});
test("sin control y sin cupo sigue ilimitado; sin metadata aún no se permite aumentar",()=>{
 const item=line();assert.equal(lineAvailability(item,[item],[],{}, {menu:combo("menu",[{...option,remaining:null,quotaRemaining:null}])}).max,undefined);
 assert.equal(lineAvailability(item,[item],[],{}, {}).pending,true);
});

test("un detalle fresco agotado bloquea aunque la carta todavía conserve existencias anteriores",()=>{
 const item=line();const products={aji:{remaining:3,availabilityStatus:"available"}};
 assert.equal(lineAvailability(item,[item],[],products,{menu:combo("menu",[{...option,available:false,remaining:0}])}).max,0);
});
