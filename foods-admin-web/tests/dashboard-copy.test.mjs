import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url);
const source=readFileSync(new URL("../src/modules/dashboard/presentation/dashboard-view.tsx",import.meta.url),"utf8");
const base={salesNet:"140",paidOrders:4,averageTicket:"35",openOrders:3,criticalStock:1,purchasesToApprove:0,reservationsToday:2,kitchenPending:2,businessDate:"2026-10-07",
 operations:{pendingBalance:"80",unpaidOrders:1,partialOrders:2,tablesTotal:8,tablesOccupied:3,kitchenConfirmed:1,kitchenPreparing:1,readyOrders:1,deliveryPending:2,deliveryInTransit:1,activeCashRegisters:2,openCashShifts:1,cashBalance:"120",soldOutProducts:1,tablesBillRequested:1,tableMinutesAverage:48,oldestTable:{name:"04",minutes:95}},
 hourlySales:[],topProducts:[],
 period:{key:"today",days:1,from:"2026-10-07",to:"2026-10-07"},
 previous:{salesNet:"100",paidOrders:5,averageTicket:"35",from:"2026-09-30",to:"2026-09-30"},
 trend:{granularity:"hour",now:"13",points:[{key:"12",current:"90",previous:"60",orders:2},{key:"13",current:"50",previous:"40",orders:2}]},
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
  if(name==="@/design-system/icon-tile")return {IconTile:"IconTile"};
  if(name==="@/design-system/page-header")return {PageHeader:"PageHeader"};
  if(name==="@/providers/session-context")return {useSession:()=>({user:{name:"Ana",platformAdmin:false},location:{id:"local",name:"Principal"},organization:{id:"company"},modules,menuAccess:["*"],permissions:["*"],setupRequired:false,...sessionOverrides})};
  if(name==="@/providers/settings-context")return {useSettings:()=>({currencyDecimals:2,currencySymbol:"S/",currencyPosition:"before"})};
  if(name==="@/shared/i18n/regional-format")return {formatRegionalNumber:value=>value.toFixed(2)};
  if(name==="@/design-system/tabs")return {Tabs:"Tabs"};
  if(name==="./dashboard-charts")return {BarList:"BarList",CountUp:"CountUp",DonutChart:"DonutChart",RankTrack:"RankTrack",TrendChart:"TrendChart"};
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
 assert.deepEqual(values,["S/ 140.00","4","S/ 35.00"]);
 for(const label of ["Ventas","Pedidos","Promedio","Pedidos activos","Por revisar","Lo más vendido","Ahora mismo","Saldo por cobrar","Antes: S/ 100.00"])assert.ok(has(view,label),label);
 assert.ok(has(view,"Comparado con el miércoles pasado a esta hora"),"Hoy se compara con el mismo día de la semana anterior hasta la misma hora, dicho una sola vez");
 for(const long of ["Pedidos cobrados","Promedio por pedido"])assert.equal(has(view,long),false,`Etiqueta corta en una línea: ${long}`);
 assert.equal(view.some(node=>node.props?.className?.startsWith("dash-change")&&/pasado/.test(text(node.props.children))),false,"La base de comparación no se repite en cada indicador");
 assert.ok(has(view,"+40%Antes: S/ 100.00"),"Ventas muestra su variación y el importe anterior");
 assert.ok(has(view,"-20%"),"Pedidos conserva su propia variación");
 for(const repeated of ["Ventas · Hoy","2 en cocina","Sin pagar","Ver ventas","HOY","Se actualiza cada 30 s","Productos más vendidos"])assert.equal(has(view,repeated),false,`Sin repetir: ${repeated}`);
 assert.equal(view.some(node=>/devoluci[oó]n/i.test(text(node.props?.children))),false);
 const chart=view.find(node=>node.type==="TrendChart");
 assert.deepEqual(JSON.parse(JSON.stringify(chart.props.points.map(point=>[point.tick,point.value,point.previous,point.orders,point.future]))),[["12",90,60,2,false],["13",50,40,2,false]]);
 assert.deepEqual(JSON.parse(JSON.stringify(chart.props.series)),{current:"Hoy",previous:"Miércoles pasado"},"La comparación se dibuja detrás y se nombra en la leyenda");
 assert.ok(source.includes("money(data.salesNet)"));assert.ok(source.includes("money(data.averageTicket)"));
});

test("dashboard ofrece periodos Hoy, 7 y 30 días como filtro único",()=>{
 const view=render(base);
 const radios=view.filter(node=>node.props?.role==="radio");
 assert.deepEqual(radios.map(node=>text(node.props.children)),["Hoy","7 días","30 días"]);
 assert.deepEqual(radios.map(node=>node.props["aria-checked"]),[true,false,false]);
 assert.ok(source.includes("queryFn:()=>getDashboard(period)"));
 assert.ok(source.includes("placeholderData:keepPreviousData"),"Al cambiar de periodo conserva el render anterior");
 const week=render({...base,period:{key:"7d",days:7,from:"2026-10-01",to:"2026-10-07"},trend:{granularity:"day",now:"2026-10-07",points:[{key:"2026-10-06",current:"10",previous:"5",orders:1},{key:"2026-10-07",current:"20",previous:"10",orders:2}]}});
 assert.ok(has(week,"Comparado con los 7 días anteriores"));
 assert.deepEqual(week.find(node=>node.type==="TrendChart").props.points.map(point=>point.tick),["mar","Hoy"]);
 assert.deepEqual(JSON.parse(JSON.stringify(week.find(node=>node.type==="TrendChart").props.series)),{current:"Últimos 7 días",previous:"7 días anteriores"});
});

test("dashboard detiene la serie actual en la hora en curso y omite el promedio sin pedidos",()=>{
 const view=render({...base,paidOrders:0,averageTicket:"0",salesNet:"0",trend:{granularity:"hour",now:"11",points:[{key:"12",current:"0",previous:"60",orders:0},{key:"14",current:"0",previous:"40",orders:0}]}});
 const chart=view.find(node=>node.type==="TrendChart");
 assert.ok(chart,"La semana pasada se ve aunque hoy aún no haya cobros");
 assert.deepEqual(JSON.parse(JSON.stringify(chart.props.points.map(point=>[point.tick,point.future]))),[["11",false],["12",true],["13",true],["14",true]]);
 const changes=view.filter(node=>node.props?.className?.startsWith("dash-change"));
 assert.equal(changes.length,2,"Sin pedidos no hay variación de Promedio");
});

test("dashboard conserva importes negativos y estados vacíos específicos",()=>{
 const view=render({...base,salesNet:"-10",averageTicket:"-2.50",criticalStock:0,kitchenPending:0,previous:{...base.previous,salesNet:"0"},trend:{granularity:"hour",points:[]},operations:{...base.operations,soldOutProducts:0}});
 assert.ok(view.some(node=>node.type==="strong"&&text(node.props.children)==="-S/ 10.00"));
 for(const label of ["Aún no hay ventas cobradas hoy","Sin productos vendidos todavía"])assert.ok(has(view,label),label);
 assert.ok(view.some(node=>node.type==="p"&&text(node.props.children)==="Sin pendientes por revisar"),"Por revisar conserva su estado vacío");
 assert.equal(view.some(node=>/para comparar|Antes:/.test(text(node.props?.children))),false,"Sin base de comparación no se muestra variación ni importe anterior");
 assert.equal(view.some(node=>node.type==="TrendChart"),false);
});

test("dashboard consolida la operación en vivo en una columna",()=>{
 const nodes=render(base);
 for(const label of ["Atención","Pedidos activos","Mesas","Ocupadas","3 de 8","Libres","5","Por cobrar","Tiempo promedio","48 min","Más tiempo","Mesa 04 · 1 h 35 min","Cocina","Delivery","Caja","Por preparar","En preparación","Por entregar","Por despachar","En camino","Turnos abiertos","1 de 2","Efectivo en caja","S/ 120.00","Productos agotados","Stock crítico","S/ 80.00"])assert.ok(has(nodes,label),label);
 assert.ok(has(nodes,"3 pedidos · 2 con pago parcial"));
 const named=render({...base,operations:{...base.operations,oldestTable:{name:"Mesa 1",minutes:4371}}});
 assert.ok(has(named,"Mesa 1 · 72 h 51 min"),"Un nombre que ya dice «Mesa» no se repite");
 assert.equal(has(named,"Mesa Mesa 1 · 72 h 51 min"),false);
 const single=render({...base,operations:{...base.operations,unpaidOrders:1,partialOrders:0}});
 assert.ok(has(single,"1 pedido"),"Sin pagos parciales no se agrega «0 con pago parcial»");
 assert.equal(nodes.filter(node=>node.type==="aside").length,1,"Operación y pendientes por revisar comparten un solo panel");
});

test("dashboard suma ritmo del día y pedidos por canal",()=>{
 const view=render({...base,salesByChannel:[{value:"delivery",label:"Delivery",total:"100",count:3,orders:3,averageTicket:"33.3333"},{value:"salon",label:"Salón",total:"40",count:1,orders:1,averageTicket:"40"}]});
 const pace=view.find(node=>node.props?.className==="dash-pace");
 assert.ok(pace,"Hoy muestra el ritmo frente al día completo de la semana pasada");
 assert.equal(text(pace.props.children[0]),"140% de lo vendido el miércoles completo");
 const bars=view.find(node=>node.type==="BarList"&&node.props.label==="Ventas por canal");
 assert.deepEqual(bars.props.items.map(item=>item.detail),["3 pedidos · promedio S/ 33.33","1 pedido · promedio S/ 40.00"]);
 assert.deepEqual(bars.props.items.map(item=>item.tone),["indigo","indigo"],"Todos los canales en índigo");
 const methods=render({...base,salesByPaymentMethod:[{value:"cash",label:"Efectivo",total:"80",count:2,affectsCash:true},{value:"card",label:"Tarjeta",total:"40",count:1,affectsCash:false,digitalIndex:0},{value:"old",label:"Antiguo",total:"20",count:1}]});
 assert.deepEqual(methods.find(node=>node.type==="DonutChart").props.items.map(item=>item.tone),["violet","teal","gray"],"Efectivo violeta, digitales por su orden en el catálogo; fuera del catálogo, gris");
 const week=render({...base,period:{key:"7d",days:7,from:"2026-10-01",to:"2026-10-07"},trend:{granularity:"day",now:"2026-10-07",points:[{key:"2026-10-07",current:"20",previous:"10",orders:2}]}});
 assert.equal(week.some(node=>node.props?.className==="dash-pace"),false,"El ritmo del día solo aplica a «Hoy»");
 const free=render({...base,operations:{...base.operations,tablesOccupied:0,tablesBillRequested:0,tableMinutesAverage:null,oldestTable:null}});
 assert.equal(has(free,"Tiempo promedio")||has(free,"Más tiempo"),false,"Sin mesas ocupadas no hay tiempos");
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

test("dashboard muestra tres indicadores del periodo con Ventas como protagonista",()=>{
 const view=render({...base,topProducts:[{name:"Lomo",qty:"2",revenue:"60"}]});
 const kpis=view.filter(node=>node.type==="article"&&node.props?.className?.includes("dash-kpi"));
 assert.equal(kpis.length,3,"Pedidos activos es operación en vivo: vive en «Ahora mismo», no entre los indicadores del periodo");
 assert.ok(kpis[0].props.className.includes("lead"),"Ventas abre el mosaico como única protagonista");
 assert.equal(kpis.filter(node=>node.props.className.includes("lead")).length,1);
 const counters=view.filter(node=>node.type==="CountUp");
 assert.deepEqual(counters.map(node=>node.props.children),["S/ 140.00","4","S/ 35.00","S/ 80.00"],"Las cifras cuentan hacia el valor final que reciben los lectores de pantalla");
 assert.deepEqual(counters.map(node=>node.props.value),[140,4,35,80]);
 const change=view.find(node=>node.type==="p"&&node.props?.className==="dash-change up");
 assert.equal(text(change.props.children[0]),"+40%","El porcentaje va en su propia pastilla");
 assert.ok(view.some(node=>node.type==="DonutChart"&&node.props.label==="Cobros por medio de pago"),"Medios de pago se muestra como anillo");
 assert.equal(view.filter(node=>node.type==="BarList").length,1,"Solo el canal usa barras");
 const tabs=view.find(node=>node.type==="Tabs");
 assert.deepEqual(JSON.parse(JSON.stringify(tabs.props.items.map(item=>item.label))),["Productos","Categorías"],"Productos y categorías comparten un solo ranking");
 assert.ok(view.some(node=>node.type==="RankTrack"),"El ranking conserva su barra");
 assert.equal(view.filter(node=>node.type==="Link"&&node.props.href==="/sales").length,1,"Un solo acceso a Ventas");
 assert.equal(view.filter(node=>node.type==="IconTile").length,3,"Los iconos quedan solo en los indicadores");
 const delays=view.filter(node=>node.props?.className?.includes("dash-enter")).map(node=>node.props.style.animationDelay);
 assert.deepEqual(delays.slice(0,4),["0ms","50ms","100ms","150ms"],"La entrada se escalona en orden de lectura");
});

test("el skeleton del dashboard reproduce los tres indicadores",()=>{
 const nodes=render(undefined,{}, {isLoading:true});
 assert.equal(nodes.filter(node=>node.props?.className?.includes("dash-kpi dashboard-skeleton-kpi")).length,3);
});
