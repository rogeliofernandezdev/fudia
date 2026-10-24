// Production history modal; illustrative remote data, no API or persisted writes.
import {readFileSync,readdirSync,existsSync,mkdirSync} from "node:fs";
import {createRequire} from "node:module";
import {dirname,resolve} from "node:path";
import {fileURLToPath} from "node:url";
import {createServer} from "node:http";
import vm from "node:vm";
import ts from "typescript";
import React from "react";
import {renderToStaticMarkup} from "react-dom/server";
import assert from "node:assert/strict";

const require=createRequire(import.meta.url),root=resolve(dirname(fileURLToPath(import.meta.url)),"../.."),cache=new Map();
const item={productId:"preview",name:"Causa Rellena de Pollo",quantityControl:"portions"};
const entry={id:"event",createdAt:"2026-10-11T22:13:00Z",businessDate:"2026-10-11",previousPortionQuantity:15,portionQuantity:20,soldQuantity:5,previousManualStatus:"available",manualStatus:"available",reason:"Producción adicional para el servicio de la tarde",userName:"Luis Angel Quispe Sulca"};
let result={};
function load(path){
 const base=resolve(root,"src",path),file=[base,`${base}.tsx`,`${base}.ts`].find(candidate=>existsSync(candidate)&&/\.tsx?$/.test(candidate));
 assert.ok(file,path);if(cache.has(file))return cache.get(file);
 const exports={};cache.set(file,exports);
 const code=ts.transpileModule(readFileSync(file,"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,esModuleInterop:true}}).outputText;
 vm.runInNewContext(code,{exports,require:name=>{
  if(name.endsWith(".css"))return {};
  if(name==="@tanstack/react-query")return {useQuery:()=>({...result,refetch:()=>{}})};
  if(name==="@/providers/session-context")return {useSession:()=>({organization:{id:"preview"},location:{id:"preview",country:"PE",timezone:"America/Lima"},can:()=>true})};
  if(name==="../infrastructure/availability-api")return {};
  if(name==="@/design-system")return {...load("design-system/icons"),...load("design-system/page-header"),...load("design-system/modal")};
  if(name.startsWith("@/")||name.startsWith("."))return load(name.startsWith("@/")?name.slice(2):resolve(dirname(file),name));
  return require(name);
 }});return exports;
}
const {AvailabilityHistoryDialog}=load("modules/menu/availability/presentation/availability-history-dialog");
const css=readdirSync(resolve(root,".next/static/css")).filter(name=>name.endsWith(".css")).map(name=>readFileSync(resolve(root,".next/static/css",name),"utf8")).join("\n");
assert.match(css,/availability-history-quantity[^}]*flex-wrap:wrap/);
const font=css.match(/@font-face\{[^}]*font-family:Plus Jakarta Sans;[^}]*unicode-range:u\+00\?\?[^}]*\}/)?.[0]?.match(/media\/([^/)]+\.woff2)/)?.[1];assert.ok(font);
const output=resolve(process.argv[2]);mkdirSync(output,{recursive:true});
const server=createServer((req,res)=>{
 if(req.url==="/font.woff2"){res.setHeader("content-type","font/woff2");res.end(readFileSync(resolve(root,".next/static/media",font)));return;}
 const mode=new URL(req.url,"http://localhost").searchParams.get("mode")??"normal";
 const legacy={...entry,id:"legacy",businessDate:null,previousPortionQuantity:null,portionQuantity:null,soldQuantity:null,previousManualStatus:null,manualStatus:null,reason:null};
 const records=mode==="legacy"?[legacy]:mode==="long"?Array.from({length:10},(_,index)=>({...entry,id:String(index),reason:"Producción adicional para atender una reserva de grupo; se confirma la cantidad preparada y el motivo del ajuste con el responsable de cocina.",userName:"Luis Angel Quispe Sulca Fernández de los Ríos",previousPortionQuantity:2000000000,portionQuantity:2100000000})):mode==="empty"?[]:[entry,{...entry,id:"status",previousPortionQuantity:20,portionQuantity:20,previousManualStatus:"available",manualStatus:"sold_out",reason:"Cierre de producción"},legacy];
 result=mode==="loading"?{isLoading:true}:mode==="error"?{isError:true,error:{message:"No pudimos conectar con el servidor. Inténtalo de nuevo."}}:{data:{items:records,total:records.length,page:1,pageSize:10}};
 const markup=renderToStaticMarkup(React.createElement(AvailabilityHistoryDialog,{item,close:()=>{}}));
 res.setHeader("content-type","text/html; charset=utf-8");res.end(`<!doctype html><html lang="es"><head><meta name="viewport" content="width=device-width,initial-scale=1"><style>@font-face{font-family:Plus Jakarta Sans;font-weight:200 800;src:url(/font.woff2)}:root{--font-jakarta:"Plus Jakarta Sans"}${css}</style></head><body>${markup}</body></html>`);
});
await new Promise(resolve=>server.listen(0,"127.0.0.1",resolve));
const {chromium}=require(process.env.FOODS_PLAYWRIGHT_PATH??"playwright");let browser;
try{
 browser=await chromium.launch({executablePath:process.env.FOODS_BROWSER_PATH,headless:true});
 for(const width of [390,768,1440])for(const mode of ["normal","legacy","long","loading","error","empty"]){
  const page=await browser.newPage({viewport:{width,height:844},reducedMotion:"reduce"});await page.goto(`http://127.0.0.1:${server.address().port}/?mode=${mode}`);await page.evaluate(()=>document.fonts.ready);
  const geometry=await page.locator(".crud-modal").evaluate(el=>{const r=el.getBoundingClientRect(),close=el.querySelector(".modal-close").getBoundingClientRect();return {right:r.right,bottom:r.bottom,width:r.width,overflow:document.documentElement.scrollWidth>innerWidth,close:Math.min(close.width,close.height),bodyScroll:el.querySelector(".modal-body").scrollHeight>el.querySelector(".modal-body").clientHeight};});
  assert.equal(geometry.overflow,false);assert.ok(geometry.right<=width&&geometry.bottom<=844);assert.ok(geometry.close>=44);assert.equal(await page.locator(".modal-header small,.modal-header p").count(),0);
  if(["normal","legacy","long"].includes(mode)){
   assert.equal(await page.locator("th").count(),5);
   const scroll=page.getByRole("region");await scroll.focus();assert.equal(await scroll.evaluate(el=>document.activeElement===el),true);
   if(width===390)assert.ok(await scroll.evaluate(el=>el.scrollWidth>el.clientWidth));
   assert.ok(await page.locator(".availability-history-table time").evaluateAll(elements=>elements.every(el=>getComputedStyle(el).whiteSpace==="nowrap")));
   if(mode==="legacy")assert.equal(await page.locator("tbody td:nth-child(3)").textContent(),"Sin registrar");
   if(mode==="long")assert.equal(geometry.bodyScroll,true);
   if(width===1440)assert.ok(await page.locator("tbody td").evaluateAll(elements=>elements.every(el=>el.scrollWidth<=el.clientWidth+1)));
  }
  await page.evaluate(()=>document.activeElement?.blur());
  await page.screenshot({path:resolve(output,`history-${mode}-${width}.png`),fullPage:true});
  if(mode==="normal"&&width===1440)await page.locator(".crud-modal").screenshot({path:resolve(output,"history-detail.png")});
  if(width===390&&mode==="normal"){await page.getByRole("region").evaluate(el=>el.scrollLeft=el.scrollWidth);await page.screenshot({path:resolve(output,"history-mobile-end.png")});}
  console.log(`${width}px ${mode}: panel contenido, cierre 44px${["normal","legacy","long"].includes(mode)?", cinco columnas":", estado explícito"}`);await page.close();
 }
}finally{await browser?.close();await new Promise(resolve=>server.close(resolve));}
