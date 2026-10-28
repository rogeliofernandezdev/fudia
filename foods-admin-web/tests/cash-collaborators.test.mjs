import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import vm from "node:vm";
import ts from "typescript";

function compile(file,imports={}){
 const exports={};
 vm.runInNewContext(ts.transpileModule(readFileSync(new URL(`../${file}`,import.meta.url),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports,require:name=>imports[name]});
 return exports;
}
const {availableCashCollaborators}=compile("src/modules/operations/cash/domain/collaborators.ts");
const {canCloseCashShift}=compile("src/modules/operations/cash/domain/shift-ownership.ts");
test("solo el abridor puede cerrar; permiso administrativo no sustituye la identidad",()=>{
 const shift={status:"open",openedById:"opener"};
 assert.equal(canCloseCashShift(shift,"opener",true),true);
 assert.equal(canCloseCashShift(shift,"other-cashier",true),false);
 assert.equal(canCloseCashShift(shift,"administrator",true),false);
 assert.equal(canCloseCashShift(shift,"opener",false),false);
 assert.equal(canCloseCashShift({...shift,status:"closed"},"opener",true),false);
 assert.equal(canCloseCashShift({status:"open"},"opener",true),false);
 assert.equal(canCloseCashShift(shift,undefined,true),false);
});
test("el selector excluye al actor, duplicados y personas ocupadas; un turno cerrado no asigna",()=>{
 const options=[{id:"self"},{id:"assigned"},{id:"busy",assignedShiftId:"other-shift"},{id:"available"}];
 const users=[{userId:"assigned"}];
 assert.deepEqual(Array.from(availableCashCollaborators(options,users,"self","open"),user=>user.id),["available"]);
 assert.equal(availableCashCollaborators(options,users,"self","closed").length,0);
});
const routes=compile("src/shared/routing/page-routes.ts");
const navigation=compile("src/shell/navigation.ts",{"@/shared/routing/page-routes":routes});
test("la capacidad temporal abre solo POS sin conceder Caja ni modificar permisos",()=>{
 const context={user:{platformAdmin:false},modules:{pos:true,caja:true,pedidos:true},menuAccess:["pedidos"],permissions:["orders.read"],cashCollection:true};
 const items=navigation.visibleNavigation(context).flatMap(group=>group.items);
 assert.ok(items.some(item=>item.module==="pos"));
 assert.equal(items.some(item=>item.module==="caja"),false);
 assert.deepEqual(context.permissions,["orders.read"]);
 assert.equal(navigation.visibleNavigation({...context,cashCollection:false}).flatMap(group=>group.items).some(item=>item.module==="pos"),false);
 assert.equal(navigation.visibleNavigation({...context,modules:{...context.modules,pos:false}}).flatMap(group=>group.items).some(item=>item.module==="pos"),false);
});
