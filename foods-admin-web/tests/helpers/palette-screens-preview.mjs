// Read-only specimens of the production Dashboard indicators and CashRegisters.
// node tests/helpers/palette-screens-preview.mjs ; no API, authentication or writes.
import {readFileSync,readdirSync,existsSync} from "node:fs";
import {createRequire} from "node:module";
import {resolve,dirname} from "node:path";
import {fileURLToPath} from "node:url";
import {createServer} from "node:http";
import vm from "node:vm";
import ts from "typescript";
import React from "react";
import {renderToStaticMarkup} from "react-dom/server";
import {cashReportFixture} from "./cash-report-fixture.mjs";

const root=resolve(dirname(fileURLToPath(import.meta.url)),"../.."),require=createRequire(import.meta.url),h=React.createElement,cache=new Map();
const dashboard={salesNet:"1840",paidOrders:46,averageTicket:"40",openOrders:8,criticalStock:2,purchasesToApprove:1,reservationsToday:3,kitchenPending:4,businessDate:"2026-10-26",topProducts:[],hourlySales:[],salesByChannel:[],salesByPaymentMethod:[],salesByCategory:[],period:{key:"today",days:1},previous:{salesNet:"1600",paidOrders:50,averageTicket:"32",from:"2026-10-19"},trend:{granularity:"hour",points:[{key:"12",current:"520",orders:13},{key:"13",current:"800",orders:20},{key:"14",current:"520",orders:13}]},operations:{pendingBalance:"320",unpaidOrders:3,partialOrders:1,tablesOccupied:5,tablesTotal:12,kitchenConfirmed:2,kitchenPreparing:2,readyOrders:1,deliveryPending:0,deliveryInTransit:0,openCashShifts:1,activeCashRegisters:2,cashBalance:"450",soldOutProducts:0}};
function load(file){
 if(cache.has(file))return cache.get(file);
 const exports={};cache.set(file,exports);
 let text=readFileSync(file,"utf8");
 // Freeze only animation frames for SSR: graphs show their final production geometry.
 if(file.endsWith("dashboard-charts.tsx"))text=text.replace("values:values.map(()=>base),animating:values.length>0","values,animating:false");
 const code=ts.transpileModule(text,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,esModuleInterop:true}}).outputText;
 vm.runInNewContext(code,{exports,require:name=>{
  if(name.endsWith(".css"))return {};
  if(name==="react"&&file.endsWith("dashboard-view.tsx"))return {...React,useState:initial=>[initial,()=>{}]};
  if(name==="next/link")return {__esModule:true,default:({children,...props})=>h("a",props,children)};
  if(name==="@tanstack/react-query")return {keepPreviousData:value=>value,useQuery:()=>({data:dashboard,isLoading:false,isError:false,isFetching:false})};
  if(name==="@/providers/session-context")return {useSession:()=>({user:{name:"Ana",platformAdmin:true},location:{id:"preview",name:"Local de prueba",country:"PE"},organization:{id:"preview"},modules:{},permissions:["*"],menuAccess:["*"]})};
  if(name==="@/providers/settings-context")return {useSettings:()=>({currencyDecimals:2,currencySymbol:"S/",currencyPosition:"before"})};
  if(name==="../infrastructure/dashboard-api")return {getDashboard:()=>{throw new Error("API disabled in visual fixture")}};
  if(name==="./dashboard-charts")return {...load(resolve(dirname(file),"dashboard-charts.tsx")),CountUp:({children})=>h("span",{className:"dash-count"},children)};
  if(name.startsWith(".")||name.startsWith("@/")){
   const base=name.startsWith("@/")?resolve(root,"src",name.slice(2)):resolve(dirname(file),name);
   const target=[base,base+".tsx",base+".ts"].find(path=>existsSync(path));
   if(!target)throw new Error(`Missing fixture dependency: ${name}`);
   return load(target);
  }
  return require(name);
 }});return exports;
}
const primitives={...load(resolve(root,"src/design-system/page-header.tsx")),...load(resolve(root,"src/design-system/icons.tsx")),...load(resolve(root,"src/design-system/icon-tile.tsx")),...load(resolve(root,"src/design-system/states.tsx")),...load(resolve(root,"src/modules/operations/cash/domain/shift-attribution.ts"))};
const cashSource=readFileSync(resolve(root,"src/modules/operations/cash/presentation/cash-page.tsx"),"utf8");
const ast=ts.createSourceFile("cash-page.tsx",cashSource,ts.ScriptTarget.Latest,true,ts.ScriptKind.TSX);
const cashFunction=ast.statements.find(node=>ts.isFunctionDeclaration(node)&&node.name?.text==="CashRegisters");
const cashModule={exports:{}};
vm.runInNewContext(ts.transpileModule("export "+cashFunction.getText(ast),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{...primitives,exports:cashModule.exports,require});
const {CashRegisters}=cashModule.exports;
const dashboardTree=load(resolve(root,"src/modules/dashboard/presentation/dashboard-view.tsx")).DashboardView();
function find(node,className){if(Array.isArray(node)){for(const child of node){const match=find(child,className);if(match)return match;}}else if(node&&typeof node==="object"){if(node.props?.className===className)return node;return find(node.props?.children,className);}return null;}
const css=["styles/variables.css","styles/globals.css","design-system/styles/icon-tile.css","design-system/styles/stat-card.css","design-system/styles/table-actions.css","modules/dashboard/presentation/dashboard.css","modules/operations/cash/presentation/cash.css"].map(file=>readFileSync(resolve(root,"src",file),"utf8").replace(/^@import[^;]+;/gm,"")).join("\n");
const fontCss=readdirSync(resolve(root,".next/static/css")).filter(file=>file.endsWith(".css")).map(file=>readFileSync(resolve(root,".next/static/css",file),"utf8")).join("\n");
const font=fontCss.match(/@font-face\{[^}]*font-family:Plus Jakarta Sans;[^}]*unicode-range:u\+00\?\?[^}]*\}/)?.[0].match(/media\/([^/)]+\.woff2)/)?.[1];
if(!font)throw new Error("Run a build once to provide the production font.");
const noop=()=>{};
function specimen(params){
 const long=params.has("long"),blind=params.has("blind");
 const shift={...cashReportFixture().shift,status:"open",expectedVisible:!blind,closedByName:"",closedAt:null,openedByName:"Personal de prueba",activeUserNames:["Personal de prueba"],businessDate:"2026-10-26",openedAt:"2026-10-26T14:00:00.000Z",incomeAmount:long?"1234567890.55":"430",expectedAmount:long?"1234567910.55":"450"};
 const items=[{id:"one",name:long?"Caja principal del salón y terraza con un nombre muy extenso":"Caja principal",active:true,openShift:shift},{id:"two",name:"Caja de barra",active:true,openShift:null}];
 const cash=h(CashRegisters,{items,loading:false,error:null,canManage:true,money:value=>`S/ ${Number(value).toLocaleString("es-PE",{minimumFractionDigits:2,maximumFractionDigits:2})}`,dateTime:()=>"26 oct. 2026, 9:00 a. m.",businessDate:()=>"26 oct. 2026",retry:noop,create:noop,edit:noop,toggleStatus:noop,start:noop,view:noop,movement:noop,team:noop,operation:noop,close:noop});
 return renderToStaticMarkup(h("main",null,h("p",{className:"preview-note"},"Muestra visual · datos ilustrativos · sin operaciones"),h(primitives.PageHeader,{title:"Dashboard",description:"Indicadores del periodo"}),find(dashboardTree,"dash-kpis"),h(primitives.PageHeader,{title:"Caja y turnos",description:"Fondo inicial, ingresos y egresos diferenciados por intención"}),h("section",{className:"panel cash-panel"},cash)));
}
createServer((req,res)=>{
 if(req.url==="/font.woff2"){res.setHeader("content-type","font/woff2");res.end(readFileSync(resolve(root,".next/static/media",font)));return;}
 const params=new URL(req.url,"http://localhost").searchParams;
 res.setHeader("content-type","text/html; charset=utf-8");res.end(`<!doctype html><html lang="es"><head><meta name="viewport" content="width=device-width,initial-scale=1"><title>FudIA · acentos en Dashboard y Caja</title><style>${css}@font-face{font-family:Plus Jakarta Sans;font-weight:200 800;src:url(/font.woff2)}:root{--font-jakarta:"Plus Jakarta Sans"}main{max-width:var(--size-1440);margin:auto;padding:var(--space-6)}.preview-note{margin:0 0 var(--space-5);font-size:var(--font-size-xs);color:var(--ink-500)}main>.dash-kpis{margin-bottom:var(--space-8)}.dash-enter{animation:none}@media(width<=600px){main{padding:var(--space-3)}}</style></head><body>${specimen(params)}</body></html>`);
}).listen(5198,"localhost",()=>console.log("Palette screens QA: http://localhost:5198/ (?long, ?blind)"));
