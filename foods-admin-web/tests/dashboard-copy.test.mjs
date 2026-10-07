import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url);
const source=readFileSync(new URL("../src/modules/dashboard/presentation/dashboard-view.tsx",import.meta.url),"utf8");
const base={salesNet:"140",paidOrders:4,averageTicket:"35",openOrders:3,criticalStock:1,purchasesToApprove:0,reservationsToday:2,kitchenPending:2,businessDate:"2026-10-07",operations:{pendingBalance:"80",unpaidOrders:1,partialOrders:2,tablesTotal:8,tablesOccupied:3,kitchenConfirmed:1,kitchenPreparing:1,readyOrders:1,deliveryPending:2,deliveryInTransit:1,activeCashRegisters:2,openCashShifts:1,cashBalance:"120",soldOutProducts:1},hourlySales:[],topProducts:[]};
const modules={reportes:true,pedidos:true,mesas:true,reservas:true,cocina:true,delivery:true,caja:true,productos:true,inventario:true,compras:true};

function render(data,sessionOverrides={},queryOverrides={}){
 const exports={};
 const resolve=name=>{
  if(name.endsWith(".css"))return {};
  if(name==="next/link")return {default:"Link"};
  if(name==="@tanstack/react-query")return {useQuery:()=>({data,isLoading:false,isError:false,...queryOverrides})};
  if(name==="@/design-system/icons")return {Icon:"Icon"};
  if(name==="@/design-system/page-header")return {PageHeader:"PageHeader"};
  if(name==="@/providers/session-context")return {useSession:()=>({user:{name:"Ana",platformAdmin:false},location:{id:"local",name:"Principal"},organization:{id:"company"},modules,menuAccess:["*"],permissions:["*"],setupRequired:false,...sessionOverrides})};
  if(name==="@/providers/settings-context")return {useSettings:()=>({currencyDecimals:2,currencySymbol:"S/",currencyPosition:"before"})};
  if(name==="@/shared/i18n/regional-format")return {formatRegionalNumber:value=>value.toFixed(2)};
  if(name==="@/shared/routing/page-routes"||name==="@/shell/navigation"){
   const file=name==="@/shell/navigation"?"shell/navigation.ts":"shared/routing/page-routes.ts";
   const result={};vm.runInNewContext(ts.transpileModule(readFileSync(new URL("../src/"+file,import.meta.url),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports:result,require:resolve});return result;
  }
  if(name.startsWith("."))return {};
  return require(name);
 };
 vm.runInNewContext(ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});
 const nodes=[];
 const visit=node=>{if(!node||typeof node!=="object")return;if(Array.isArray(node)){node.forEach(visit);return}nodes.push(node);if(typeof node.type==="function"&&node.type.name==="DashboardSkeleton")visit(node.type());visit(node.props?.children)};
 const root=exports.DashboardView();
 visit(root);return nodes;
}

test("dashboard usa términos del restaurante sin modificar importes ni cantidades",()=>{
 const view=render(base);
 const texts=view.filter(node=>typeof node.props?.children==="string").map(node=>node.props.children);
 for(const label of ["Ventas del día","Cobrado hoy","Pedidos cobrados","Promedio por pedido S/ 35.00","Pedidos en atención","2 en cocina","Pendientes por revisar","Productos vendidos hoy"]){assert.ok(texts.includes(label),label)}
 assert.equal(texts.some(text=>/devoluci[oó]n/i.test(text)),false);
 for(const label of ["Ventas netas","Cobros netos de hoy","Ticket promedio","Pedidos abiertos","Alertas operativas","Productos cobrados hoy"]){assert.equal(texts.some(text=>text.includes(label)),false,label)}
 const values=view.filter(node=>node.type==="strong").map(node=>node.props.children);
 assert.deepEqual(values,["S/ 140.00","4","S/ 80.00","3"]);
 assert.ok(source.includes("money(data.salesNet)"));assert.ok(source.includes("money(data.averageTicket)"));
});

test("dashboard conserva importes negativos y un vacío de alertas específico",()=>{
 const view=render({...base,salesNet:"-10",averageTicket:"-2.50",criticalStock:0,kitchenPending:0,operations:{...base.operations,soldOutProducts:0}});
 assert.ok(view.some(node=>node.type==="strong"&&node.props.children==="-S/ 10.00"));
 for(const text of ["Promedio por pedido -S/ 2.50","Sin pedidos en cocina","Sin pendientes por revisar","Sin alertas de disponibilidad o abastecimiento."]){assert.ok(view.some(node=>node.props?.children===text),text)}
});

test("dashboard consolida las áreas sin repetir cocina en pendientes",()=>{
 const nodes=render(base);
 for(const label of ["Atención","Cocina","Delivery","Caja","Mesas ocupadas","3 de 8","Por preparar","En preparación","Listos para entregar","Por despachar","En camino","Turnos abiertos","1 de 2","Efectivo en caja","S/ 120.00","Productos agotados"]){assert.ok(nodes.some(node=>node.props?.children===label),label)}
 assert.equal(nodes.some(node=>node.props?.children==="Comandas activas"),false);
 assert.ok(nodes.some(node=>node.props?.children==="3 pedidos pendientes · 2 con pago parcial"));
});

test("dashboard reserva efectivo ciego y no enlaza áreas sin acceso",()=>{
 const nodes=render({...base,operations:{...base.operations,cashBalance:null}},{menuAccess:["dashboard"],permissions:["dashboard.read"]});
 assert.ok(nodes.some(node=>node.props?.children==="Importe reservado"));
 assert.equal(nodes.some(node=>node.type==="Link"),false);
});

test("dashboard excluye paneles de módulos no contratados",()=>{
 const nodes=render(base,{modules:{reportes:true,pedidos:true}});
 for(const label of ["Cocina","Delivery","Caja","Mesas ocupadas","Reservas de hoy","Stock crítico","Productos agotados"]){assert.equal(nodes.some(node=>node.props?.children===label),false,label)}
});

test("dashboard muestra error recuperable cuando falta el contrato operativo",()=>{
 const nodes=render({...base,operations:undefined});
 assert.ok(nodes.some(node=>node.props?.children==="Resumen no disponible"));
 assert.equal(nodes.some(node=>node.type==="strong"),false);
});

test("dashboard carga sin valores simulados y reproduce los cuatro paneles",()=>{
 const nodes=render(undefined,{}, {isLoading:true});
 assert.equal(nodes.filter(node=>node.props?.className?.includes("dashboard-skeleton-operation")).length,4);
 assert.ok(nodes.some(node=>node.props?.["aria-busy"]==="true"));
 assert.equal(nodes.some(node=>node.type==="strong"),false);
});

test("dashboard conserva un error explícito y reintento si falla la petición",()=>{
 const nodes=render(base,{}, {isError:true,error:new Error("Sin conexión")});
 assert.ok(nodes.some(node=>node.props?.children==="Sin conexión"));
 assert.ok(nodes.some(node=>node.type==="button"&&node.props?.children==="Reintentar"));
 assert.equal(nodes.some(node=>node.type==="strong"),false);
});
