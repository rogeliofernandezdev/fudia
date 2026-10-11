// Render the actual category table and shared controls with illustrative API data.
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
const categories=[{id:"1",name:"Entradas",sortOrder:1,active:true,productCount:2,productScope:"prepared"},{id:"2",name:"Bebidas",sortOrder:2,active:true,productCount:4,productScope:"retail"},{id:"3",name:"Menús y combos",sortOrder:3,active:false,productCount:0,productScope:"both"}];
function load(path){
 const base=resolve(root,"src",path),file=[base,`${base}.tsx`,`${base}.ts`].find(candidate=>existsSync(candidate)&&/\.tsx?$/.test(candidate));
 assert.ok(file,path);if(cache.has(file))return cache.get(file);
 const exports={};cache.set(file,exports);
 const code=ts.transpileModule(readFileSync(file,"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,esModuleInterop:true}}).outputText;
 vm.runInNewContext(code,{exports,require:name=>{
  if(name.endsWith(".css"))return {};
  if(name==="next/dynamic")return ()=>()=>null;
  if(name==="react"&&file.endsWith("catalog-manager.tsx"))return {...React,useState:initial=>[initial==="products"?"categories":initial,()=>{}]};
  if(name==="@tanstack/react-query")return {useQueryClient:()=>({}),useMutation:()=>({}),useQuery:options=>({data:{items:options.queryKey[0]==="categories"?categories:[],total:options.queryKey[0]==="categories"?3:9},isLoading:false,isError:false})};
  if(name==="@/providers/settings-context")return {useSettings:()=>({})};
  if(name==="@/providers/session-context")return {useSession:()=>({can:()=>true,location:{id:"preview",country:"PE"}})};
  if(name==="@/providers/feedback-provider")return {useFeedback:()=>({notify:()=>{}})};
  if(name==="@/shared/hooks/use-debounced-value")return {useDebouncedValue:value=>value};
  if(name==="../infrastructure/products-api"||name==="./category-dialog")return {};
  if(name==="@/design-system")return {...load("design-system/page-header"),...load("design-system/search-field")};
  if(name==="@/design-system/confirm-dialog"||name==="@/design-system/remote-modal-skeleton")return {ConfirmDialog:()=>null,RemoteModalSkeleton:()=>null};
  if(name.startsWith("@/")||name.startsWith("."))return load(name.startsWith("@/")?name.slice(2):resolve(dirname(file),name));
  return require(name);
 }});return exports;
}
const {CatalogManager}=load("modules/menu/products/presentation/catalog-manager");
const css=readdirSync(resolve(root,".next/static/css")).filter(name=>name.endsWith(".css")).map(name=>readFileSync(resolve(root,".next/static/css",name),"utf8")).join("\n");
const font=css.match(/@font-face\{[^}]*font-family:Manrope;[^}]*unicode-range:u\+00\?\?[^}]*\}/)?.[0]?.match(/media\/([^/)]+\.woff2)/)?.[1];
assert.ok(font,"Ejecuta build antes de comprobar el CSS final.");
const output=resolve(process.argv[2]);mkdirSync(output,{recursive:true});
const server=createServer((req,res)=>{
 if(req.url==="/font.woff2"){res.setHeader("content-type","font/woff2");res.end(readFileSync(resolve(root,".next/static/media",font)));return;}
 const markup=renderToStaticMarkup(React.createElement(CatalogManager));
 res.setHeader("content-type","text/html; charset=utf-8");
 res.end(`<!doctype html><html lang="es"><head><meta name="viewport" content="width=device-width,initial-scale=1"><style>@font-face{font-family:Manrope;font-weight:200 800;src:url(/font.woff2)}:root{--font-manrope:Manrope}${css}</style></head><body><main style="padding:18px">${markup}</main></body></html>`);
});
await new Promise(resolve=>server.listen(0,"127.0.0.1",resolve));
const {chromium}=require(process.env.FOODS_PLAYWRIGHT_PATH??"playwright");let browser;
try{
 browser=await chromium.launch({executablePath:process.env.FOODS_BROWSER_PATH,headless:true});
 for(const width of [390,768,1440]){
  const page=await browser.newPage({viewport:{width,height:900},reducedMotion:"reduce"});await page.goto(`http://127.0.0.1:${server.address().port}`);await page.evaluate(()=>document.fonts.ready);
  const result=await page.locator(".toolbar").evaluate(el=>{const fields=[...el.querySelectorAll("input,select")],controls=fields.map(field=>field.closest(".ds-search-field")??field);return {count:fields.length,heights:controls.map(field=>field.getBoundingClientRect().height),overflow:document.documentElement.scrollWidth>innerWidth,rows:controls.map(field=>field.getBoundingClientRect().y),title:!!el.querySelector(".toolbar-title")};});
  assert.equal(result.count,3);assert.equal(result.title,false);assert.equal(result.overflow,false);assert.equal(new Set(result.heights).size,1);if(width<=600)assert.ok(result.heights.every(height=>height>=44));
  if(width===390)assert.equal(new Set(result.rows).size,3);else assert.equal(new Set(result.rows).size,1);
  await page.getByRole("searchbox").focus();await page.getByRole("searchbox").fill("Bebidas");await page.getByLabel("Filtrar categorías por uso").selectOption("retail");await page.getByLabel("Filtrar categorías por estado").selectOption("active");
  await page.screenshot({path:resolve(output,`categories-${width}.png`),fullPage:true});
  if(width===1440)await page.locator(".toolbar").screenshot({path:resolve(output,"categories-filters.png")});
  console.log(`${width}px: tres filtros accesibles, altura compartida, sin desbordes y sin título redundante`);await page.close();
 }
}finally{await browser?.close();await new Promise(resolve=>server.close(resolve));}
