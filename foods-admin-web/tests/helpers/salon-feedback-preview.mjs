// Real production components with illustrative data. Read-only visual QA, no API.
// node tests/helpers/salon-feedback-preview.mjs → localhost:5199/{salon,success,confirm,danger}
import {readFileSync,readdirSync,existsSync} from "node:fs";
import {createRequire} from "node:module";
import {resolve,dirname} from "node:path";
import {fileURLToPath} from "node:url";
import {createServer} from "node:http";
import vm from "node:vm";
import ts from "typescript";
import React from "react";
import {renderToStaticMarkup} from "react-dom/server";

const root=resolve(dirname(fileURLToPath(import.meta.url)),"../.."),require=createRequire(import.meta.url),cache=new Map(),h=React.createElement;
let feedbackSlot=0;
const noop=()=>{},disabled=()=>{throw new Error("Operations disabled in visual specimen")};
const money=value=>`S/ ${Number(value).toLocaleString("es-PE",{minimumFractionDigits:2,maximumFractionDigits:2})}`;
const tables=Array.from({length:8},(_,i)=>({id:String(i),name:String(i+1).padStart(2,"0"),zone:i<4?"Salón":"Terraza",seats:4,occupiedSince:new Date(Date.now()-35*60000).toISOString(),order:i%3?null:{id:`order-${i}`,code:`PED-DEMO${i}`,channel:i===3?"whatsapp":"salon",status:i===0?"listo":"preparando",total:"52",customerName:i===0?"Familia Torres":"",waiterName:"Ana García"}}));
function load(path){
 const base=resolve(root,"src",path),file=[base+".tsx",base+".ts"].find(existsSync);
 if(!file)throw new Error(`Missing specimen dependency: ${path}`);
 if(cache.has(file))return cache.get(file);
 const exports={};cache.set(file,exports);
 const code=ts.transpileModule(readFileSync(file,"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,esModuleInterop:true}}).outputText;
 vm.runInNewContext(code,{exports,require:name=>{
  if(name.endsWith(".css"))return {};
  if(name==="react"&&file.endsWith("feedback-provider.tsx"))return {...React,useState:()=>[feedbackSlot++===0?{id:1,tone:"success",title:"Producto actualizado",message:"Los cambios ya están disponibles en el catálogo."}:false,noop]};
  if(name==="next/link")return {__esModule:true,default:({children,...props})=>h("a",props,children)};
  if(name==="@tanstack/react-query")return {useQueryClient:()=>({invalidateQueries:noop}),useMutation:()=>({mutate:disabled,mutateAsync:disabled}),useQuery:options=>({data:options.queryKey[0]==="salon-floor"?{items:tables}:undefined,isLoading:false,isError:false,refetch:disabled})};
  if(name==="@/providers/feedback-provider")return {useFeedback:()=>({notify:disabled})};
  if(name==="@/providers/session-context")return {useSession:()=>({can:()=>true,user:{id:"preview"}})};
  if(name==="@/providers")return {useMoney:()=>money};
  if(name==="@/design-system")return Object.assign({},...dsFiles.map(load));
  if(name.endsWith("/comanda-view"))return {ComandaView:()=>null};
  if(name.endsWith("/order-detail-dialog"))return {};
  if(name.endsWith("/infrastructure/salon-api")||name.endsWith("/infrastructure/service-api"))return {};
  if(name.startsWith("@/")||name.startsWith("."))return load(name.startsWith("@/")?name.slice(2):resolve(dirname(file),name));
  return require(name);
 }});return exports;
}
const dsFiles=["design-system/page-header","design-system/icons","design-system/icon-tile","design-system/states","design-system/search-field","design-system/confirm-dialog"];
const {SalonManager}=load("modules/operations/salon/presentation/salon-manager"),{FeedbackProvider}=load("providers/feedback-provider"),{ConfirmDialog}=load("design-system/confirm-dialog");
const css=["styles/variables.css","styles/globals.css","design-system/styles/table-actions.css","design-system/styles/icon-tile.css","design-system/styles/modal.css","design-system/styles/confirm-dialog.css","providers/styles/feedback.css","modules/operations/styles/salon-comanda.css","modules/operations/styles/salon.css"].map(path=>readFileSync(resolve(root,"src",path),"utf8").replace(/^@import[^;]+;/gm,"")).join("\n");
const fontCss=readdirSync(resolve(root,".next/static/css")).filter(file=>file.endsWith(".css")).map(file=>readFileSync(resolve(root,".next/static/css",file),"utf8")).join("\n");
const font=fontCss.match(/@font-face\{[^}]*font-family:Plus Jakarta Sans;[^}]*unicode-range:u\+00\?\?[^}]*\}/)?.[0].match(/media\/([^/)]+\.woff2)/)?.[1];
createServer((req,res)=>{
 if(req.url==="/font.woff2"){res.setHeader("content-type","font/woff2");res.end(readFileSync(resolve(root,".next/static/media",font)));return;}
 const path=new URL(req.url,"http://localhost").pathname;
 feedbackSlot=0;
 const component=path==="/success"?h(FeedbackProvider):["/confirm","/danger"].includes(path)?h(ConfirmDialog,{open:true,tone:path==="/danger"?"danger":"success",title:path==="/danger"?"Desactivar mesa":"Activar mesa",subject:{label:"Mesa",name:"Terraza 04"},description:"¿Deseas continuar con esta acción?",confirmLabel:path==="/danger"?"Desactivar":"Activar",onCancel:disabled,onConfirm:disabled}):h(SalonManager);
 res.setHeader("content-type","text/html; charset=utf-8");res.end(`<!doctype html><html lang="es"><head><meta name="viewport" content="width=device-width,initial-scale=1"><title>FudIA · Mesas y confirmaciones</title><style>${css}@font-face{font-family:Plus Jakarta Sans;font-weight:200 800;src:url(/font.woff2)}:root{--font-jakarta:"Plus Jakarta Sans"}main{max-width:var(--size-1440);margin:auto;padding:var(--space-6)}.specimen-note{margin:0 0 var(--space-5);color:var(--ink-500);font-size:var(--font-size-xs)}@media(width<=600px){main{padding:var(--space-4)}}</style></head><body><main><p class="specimen-note">Muestra visual · datos ilustrativos · sin operaciones</p>${renderToStaticMarkup(component)}</main></body></html>`);
}).listen(5199,"localhost",()=>console.log("Salon and feedback QA: http://localhost:5199/salon"));
