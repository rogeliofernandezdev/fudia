import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url);
const source=readFileSync(new URL("../src/modules/dashboard/presentation/dashboard-view.tsx",import.meta.url),"utf8");
const base={salesNet:"140",paidOrders:4,averageTicket:"35",openOrders:3,criticalStock:1,purchasesToApprove:0,reservationsToday:2,kitchenPending:2,businessDate:"2026-10-07",
 operations:{pendingBalance:"80",unpaidOrders:1,partialOrders:2,tablesTotal:8,tablesOccupied:3,kitchenConfirmed:1,kitchenPreparing:1,readyOrders:1,deliveryPending:2,deliveryInTransit:1,activeCashRegisters:2,openCashShifts:1,cashBalance:"120",soldOutProducts:1},
 hourlySales:[],topProducts:[],
 period:{key:"today",days:1,from:"2026-10-07",to:"2026-10-07"},
 previous:{salesNet:"100",paidOrders:5,averageTicket:"35",from:"2026-09-30",to:"2026-09-30"},
 trend:{granularity:"hour",points:[{key:"12",current:"90",previous:"60",orders:2},{key:"13",current:"50",previous:"40",orders:2}]},
 salesByChannel:[],salesByPaymentMethod:[],salesByCategory:[]};
const modules={reportes:true,pedidos:true,mesas:true,reservas:true,cocina:true,delivery:true,caja:true,productos:true,inventario:true,compras:true};

function render(data,sessionOverrides={},queryOverrides={}){
 const exports={};
 const resolve=name=>{
  if(name.endsWith(".css"))return {};
  if(name==="react")return {useState:initial=>[typeof initial==="function"?initial():initial,()=>{}]};
  if(name==="next/link")return {default:"Link"};
  if(name==="@tanstack/react-query")return {keepPreviousData:previous=>previous,useQuery:()=>({data,isLoading:false,isError:false,isFetching:false,isPlaceholderData:false,...queryOverrides})};
  if(name==="@/design-system/icons")return {Icon:"Icon"};
  if(name==="@/design-system/page-header")return {PageHeader:"PageHeader"};
  if(name==="@/providers/session-context")return {useSession:()=>({user:{name:"Ana",platformAdmin:false},location:{id:"local",name:"Principal"},organization:{id:"company"},modules,menuAccess:["*"],permissions:["*"],setupRequired:false,...sessionOverrides})};
  if(name==="@/providers/settings-context")return {useSettings:()=>({currencyDecimals:2,currencySymbol:"S/",currencyPosition:"before"})};
  if(name==="@/shared/i18n/regional-format")return {formatRegionalNumber:value=>value.toFixed(2)};
  if(name==="./dashboard-charts")return {BarList:"BarList",Sparkline:"Sparkline",TrendChart:"TrendChart"};
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
 visit(exports.DashboardView());
 return nodes;
}
const text=node=>typeof node==="string"||typeof node==="number"?String(node):Array.isArray(node)?node.map(text).join(""):node&&typeof node==="object"?text(node.props?.children):"";
const has=(nodes,value)=>nodes.some(node=>text(node.props?.children)===value);

test("dashboard presenta la cifra del periodo, indicadores y comparación con datos del backend",()=>{
 const view=render(base);
 const values=view.filter(node=>node.type==="strong").map(node=>text(node.props.children));
 assert.deepEqual(values,["S/ 140.00","4","S/ 35.00","3"]);
 for(const label of ["Ventas · Hoy","Pedidos cobrados","Promedio por pedido","Pedidos en atención","2 en cocina","Pendientes por revisar","Productos más vendidos","Ahora mismo","Saldo por cobrar","Antes: S/ 100.00"])assert.ok(has(view,label),label);
 assert.ok(has(view,"+40% vs. el miércoles pasado"),"Hoy se compara con el mismo día de la semana anterior");
 assert.ok(has(view,"-20% vs. el miércoles pasado"),"Pedidos cobrados conserva su propia variación");
 assert.equal(view.some(node=>/devoluci[oó]n/i.test(text(node.props?.children))),false);
 const chart=view.find(node=>node.type==="TrendChart");
 assert.deepEqual(JSON.parse(JSON.stringify(chart.props.points.map(point=>[point.tick,point.value,point.orders]))),[["12",90,2],["13",50,2]]);
 assert.equal("previousLabel" in chart.props,false,"El gráfico principal muestra una sola serie");
 assert.ok(source.includes("money(data.salesNet)"));assert.ok(source.includes("money(data.averageTicket)"));
});

test("dashboard ofrece periodos Hoy, 7 y 30 días como filtro único",()=>{
 const view=render(base);
 const radios=view.filter(node=>node.props?.role==="radio");
 assert.deepEqual(radios.map(node=>text(node.props.children)),["Hoy","7 días","30 días"]);
 assert.deepEqual(radios.map(node=>node.props["aria-checked"]),[true,false,false]);
 assert.ok(source.includes("queryFn:()=>getDashboard(period)"));
 assert.ok(source.includes("placeholderData:keepPreviousData"),"Al cambiar de periodo conserva el render anterior");
 const week=render({...base,period:{key:"7d",days:7,from:"2026-10-01",to:"2026-10-07"},trend:{granularity:"day",points:[{key:"2026-10-06",current:"10",previous:"5",orders:1},{key:"2026-10-07",current:"20",previous:"10",orders:2}]}});
 assert.ok(has(week,"+40% vs. los 7 días anteriores"));
 assert.deepEqual(week.find(node=>node.type==="TrendChart").props.points.map(point=>point.tick),["mar","Hoy"]);
});

test("dashboard conserva importes negativos y estados vacíos específicos",()=>{
 const view=render({...base,salesNet:"-10",averageTicket:"-2.50",criticalStock:0,kitchenPending:0,previous:{...base.previous,salesNet:"0"},trend:{granularity:"hour",points:[]},operations:{...base.operations,soldOutProducts:0}});
 assert.ok(view.some(node=>node.type==="strong"&&text(node.props.children)==="-S/ 10.00"));
 for(const label of ["Sin pedidos en cocina","Sin pendientes por revisar","Aún no hay ventas cobradas hoy","Sin ventas para comparar con el miércoles pasado","Sin productos vendidos todavía"])assert.ok(has(view,label),label);
 assert.equal(view.some(node=>node.type==="TrendChart"),false);
});

test("dashboard consolida la operación en vivo en una columna",()=>{
 const nodes=render(base);
 for(const label of ["Atención","Cocina","Delivery","Caja","Mesas ocupadas","3 de 8","Por preparar","En preparación","Listos para entregar","Por despachar","En camino","Turnos abiertos","1 de 2","Efectivo en caja","S/ 120.00","Productos agotados","S/ 80.00"])assert.ok(has(nodes,label),label);
 assert.ok(has(nodes,"3 pedidos pendientes · 2 con pago parcial"));
});

test("dashboard reserva efectivo ciego y no enlaza áreas sin acceso",()=>{
 const nodes=render({...base,operations:{...base.operations,cashBalance:null}},{menuAccess:["dashboard"],permissions:["dashboard.read"]});
 assert.ok(has(nodes,"Importe reservado"));
 assert.equal(nodes.some(node=>node.type==="Link"),false);
});

test("dashboard excluye áreas y alertas de módulos no contratados",()=>{
 const nodes=render(base,{modules:{reportes:true,pedidos:true}});
 for(const label of ["Cocina","Delivery","Caja","Mesas ocupadas","Reservas de hoy","Stock crítico","Productos agotados"])assert.equal(has(nodes,label),false,label);
});

test("dashboard muestra error recuperable cuando falta el contrato operativo",()=>{
 const nodes=render({...base,operations:undefined});
 assert.ok(has(nodes,"Resumen no disponible"));
 assert.equal(nodes.some(node=>node.type==="strong"),false);
});

test("dashboard carga sin valores simulados con la misma estructura",()=>{
 const nodes=render(undefined,{}, {isLoading:true});
 assert.equal(nodes.filter(node=>node.props?.className?.includes("dashboard-skeleton-operation")).length,4);
 assert.ok(nodes.some(node=>node.props?.["aria-busy"]==="true"));
 assert.equal(nodes.some(node=>node.type==="strong"),false);
});

test("dashboard conserva un error explícito y reintento si falla la petición",()=>{
 const nodes=render(base,{}, {isError:true,error:new Error("Sin conexión")});
 assert.ok(has(nodes,"Sin conexión"));
 assert.ok(nodes.some(node=>node.type==="button"&&node.props?.children==="Reintentar"));
 assert.equal(nodes.some(node=>node.type==="strong"),false);
});
