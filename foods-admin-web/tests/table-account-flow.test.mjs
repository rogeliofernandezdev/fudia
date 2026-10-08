import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url),root=new URL("../src/modules/operations/orders/",import.meta.url);
function compile(file,resolve=require){const exports={};vm.runInNewContext(ts.transpileModule(readFileSync(new URL(file,root),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});return exports}
const flow=compile("domain/service-flow.ts"),actions=compile("domain/order-actions.ts");
let userId="waiter",manage=true;const mutations=[],feedback=[],invalidations=[];
const controls=compile("presentation/order-service-controls.tsx",name=>{
 if(name.endsWith("service-flow"))return flow;
 if(name.endsWith("order-actions"))return actions;
 if(name==="@/design-system")return{Button:"Button",Status:"Status"};
 if(name==="@/providers")return{useSession:()=>({user:{id:userId},can:()=>manage}),useFeedback:()=>({notify:v=>feedback.push(v)})};
 if(name==="@tanstack/react-query")return{useQueryClient:()=>({setQueryData:()=>{},invalidateQueries:q=>invalidations.push(q.queryKey)}),useMutation:config=>{mutations.push(config);return{isPending:false,mutate:v=>mutations.push(v)}}};
 if(name.endsWith("service-api"))return{closeOrderAccount:id=>({id})};
 if(name.endsWith(".css"))return{};
 return require(name);
});
function nodes(tree){const all=[];function visit(n){if(!n||typeof n!=="object")return;if(Array.isArray(n)){n.forEach(visit);return}all.push(n);visit(n.props?.children)}visit(tree);return all}
const order={id:"order",channel:"salon",status:"entregado",waiterId:"waiter",total:"40",paidAmount:"0",remainingAmount:"40",serviceItems:[{id:"service",orderItemId:"item",name:"Gaseosa",destination:"direct",destinationLabel:"Entrega directa",status:"listo",statusLabel:"Pendiente de entrega"}]};

test("cerrar y cobrar requieren entrega completa; agregar requiere cuenta abierta",()=>{
 assert.equal(flow.canAddConsumption(order),true);
 assert.equal(flow.canChargeAccount(order),false);
 for(const status of ["nuevo","confirmado","preparando","listo","entregado"]){const closed={...order,status,billClosedAt:"2026-10-07T10:00:00Z"};assert.equal(flow.canCloseAccount({...order,status}),false);assert.equal(flow.canChargeAccount(closed),false);assert.equal(flow.canAddConsumption({...closed,paidAmount:"10",remainingAmount:"30"}),false)}
 const served={...order,serviceItems:order.serviceItems.map(item=>({...item,status:"entregado"}))};
 assert.equal(flow.canCloseAccount(served),true);
 assert.equal(flow.canChargeAccount({...served,billClosedAt:"date"}),true);
 for(const patch of [{status:"confirmado"},{status:"preparando"},{status:"listo"},{billClosedAt:"date"},{completedAt:"date"},{channel:"delivery"}])assert.equal(flow.canCloseAccount({...served,...patch}),false);
 for(const destination of ["kitchen","bar","direct"])for(const status of ["confirmado","preparando","listo"]){const pending={...served,serviceItems:[...served.serviceItems,{id:"pending",destination,status}]};assert.equal(flow.canCloseAccount(pending),false);assert.equal(flow.canChargeAccount({...pending,billClosedAt:"date"}),false)}
 assert.equal(flow.canCloseAccount({...served,serviceItems:undefined}),true); // historical served order
 for(const patch of [{status:"nuevo"},{status:"cancelado"},{completedAt:"2026-10-07T12:00:00Z"},{accountState:"paid"},{paidAmount:"40",remainingAmount:"0"}])assert.equal(flow.canAddConsumption({...order,...patch}),false);
 assert.equal(actions.nextOrderAction(order),null);
});
test("el producto muestra solo destino y estado; la entrega no se repite por fila",()=>{
 userId="waiter";manage=true;
 const ready=nodes(controls.OrderItemService({order,itemId:"item"}));
 assert.equal(ready.some(n=>n.type==="Button"),false);
 assert.ok(ready.some(n=>n.type==="Status"&&n.props.children==="Pendiente de entrega"));
 for(const [viewer,permission] of [["other",true],["waiter",false]]){userId=viewer;manage=permission;assert.equal(nodes(controls.OrderItemService({order,itemId:"item"})).some(n=>n.type==="Button"),false)}
 userId="waiter";manage=true;
 for(const status of ["confirmado","preparando","entregado"]){assert.equal(nodes(controls.OrderItemService({order:{...order,serviceItems:[{...order.serviceItems[0],status}]},itemId:"item"})).some(n=>n.type==="Button"),false)}
});
test("la cuenta muestra agregar y cerrar, nunca reabrir ni liberar manualmente",()=>{
 const buttons=o=>nodes(controls.OrderAccountActions({order:o,onAdd:()=>{}})).filter(n=>n.type==="Button");
 assert.deepEqual(buttons(order).map(n=>n.props.children),["Agregar productos"]);
 const served={...order,serviceItems:order.serviceItems.map(item=>({...item,status:"entregado"}))};
 assert.deepEqual(buttons(served).map(n=>n.props.children),["Agregar productos","Cerrar cuenta"]);
 assert.ok(nodes(controls.OrderAccountActions({order:served,onAdd:()=>{},busy:true})).filter(n=>n.type==="Button").every(n=>n.props.disabled));
 assert.deepEqual(buttons({...served,billClosedAt:"date"}).map(n=>n.props.children),[]);
 assert.equal(buttons({...order,billClosedAt:"date",accountState:"paid"}).length,0);
 assert.equal(controls.accountLabel(order),"Cuenta abierta");assert.equal(controls.accountLabel({...order,billClosedAt:"date"}),"Por cobrar");assert.equal(controls.accountLabel({...order,accountState:"paid"}),"Pagada");
 userId="other";assert.equal(controls.OrderAccountActions({order}),null);userId="waiter";
});
test("transiciones operativas invalidan las vistas sin modal exitoso; errores explícitos",()=>{
 const config=mutations.find(m=>m.onSuccess);config.onSuccess({id:"order"});assert.equal(feedback.length,0);
 for(const key of ["orders","salon-floor","pos-orders","kitchen-tickets","dashboard","sales"])assert.ok(invalidations.some(q=>q[0]===key));
 config.onError(new Error("No disponible"));assert.equal(feedback.at(-1).tone,"danger");assert.equal(feedback.at(-1).message,"No disponible");
});
test("agregar envía únicamente productos nuevos y la clave de reintento",()=>{
 const calls=[];const api=compile("infrastructure/service-api.ts",name=>name==="@/shared/api/client"?{apiFetch:(url,options)=>calls.push({url,options})}:require(name));
 api.addOrderConsumption("order",{lines:[{productId:"product",qty:2,note:"Sin hielo",selections:[],unitPrice:999,lineKey:"local"}]},"request-key");
 assert.equal(calls[0].url,"orders/order/items");assert.equal(calls[0].options.method,"POST");assert.deepEqual(JSON.parse(calls[0].options.body),{requestKey:"request-key",items:[{productId:"product",qty:2,note:"Sin hielo",selections:[]}]});
});
