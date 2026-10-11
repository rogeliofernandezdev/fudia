// Production listing with isolated illustrative API data; never persists changes.
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
let mode="normal";
const base={imageUrl:null,categoryName:"Entradas",quantityControl:"portions",status:"available",source:"portions",manualStatus:"available",portionQuantity:24,soldQuantity:7,remaining:17,inventoryUnit:null,note:"",businessDate:"2026-10-11"};
const items=[
 {...base,productId:"causa",name:"Causa Rellena de Pollo",status:"sold_out",portionQuantity:0,soldQuantity:0,remaining:0},
 {...base,productId:"tequenos",name:"Tequeños de Queso",quantityControl:"none",source:"manual",portionQuantity:null,remaining:null},
 {...base,productId:"arroz",name:"Arroz con Mariscos",categoryName:"Segundos"},
 {...base,productId:"lomo",name:"Lomo Saltado Criollo",categoryName:"Segundos",status:"low",portionQuantity:12,soldQuantity:10,remaining:2},
 {...base,productId:"caldo",name:"Caldo de Gallina Tradicional",categoryName:"Sopas",manualStatus:"sold_out",status:"sold_out",source:"manual"},
 {...base,productId:"bebida",name:"Chicha Morada",categoryName:"Bebidas",quantityControl:"inventory",source:"inventory",portionQuantity:null,soldQuantity:4,remaining:3.125,inventoryUnit:"L"},
 {...base,productId:"combo",name:"Menú del día",categoryName:"Menús y combos",quantityControl:"none",source:"combo_components",status:"sold_out",portionQuantity:null,remaining:null},
];
function load(path){
 const basePath=resolve(root,"src",path),file=[basePath,`${basePath}.tsx`,`${basePath}.ts`].find(candidate=>existsSync(candidate)&&/\.tsx?$/.test(candidate));
 assert.ok(file,path);if(cache.has(file))return cache.get(file);
 const exports={};cache.set(file,exports);
 const code=ts.transpileModule(readFileSync(file,"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,esModuleInterop:true}}).outputText;
 vm.runInNewContext(code,{exports,require:name=>{
  if(name.endsWith(".css"))return {};
  if(name==="react")return {...React,useState:initial=>[typeof initial==="object"&&initial!==null&&mode==="edited"?{arroz:"30"}:initial,()=>{}]};
  if(name==="@tanstack/react-query")return {
   useQuery:options=>options.queryKey[0]!=="product-availability"?{data:{items:[{id:"entradas",name:"Entradas"}]}}:mode==="loading"?{isLoading:true}:mode==="error"?{isError:true,error:{message:"No pudimos conectar con el servidor. Inténtalo de nuevo."},refetch(){}}:{data:{items:mode==="empty"?[]:mode==="long"?items.map(item=>({...item,name:`${item.name} — preparación especial para el servicio de la tarde con ingredientes de temporada`,soldQuantity:2000000000,remaining:2100000000})):items,total:mode==="empty"?0:items.length,businessDate:base.businessDate}},
   useQueryClient:()=>({invalidateQueries(){}}),useMutation:()=>({isPending:false}),
  };
  if(name==="@/providers")return {useFeedback:()=>({notify(){}})};
  if(name==="@/providers/session-context")return {useSession:()=>({organization:{id:"preview"},location:{id:"preview",country:"PE",timezone:"America/Lima"},can:permission=>mode!=="readonly"||permission==="menu.read",isLoading:false})};
  if(name==="../infrastructure/availability-api")return {};
  if(name==="./availability-reason-dialog")return {AvailabilityReasonDialog:()=>null};
  if(name==="./availability-history-dialog")return {AvailabilityHistoryDialog:()=>null};
  if(name==="@/design-system")return {...load("design-system/icons"),...load("design-system/page-header"),...load("design-system/search-field")};
  if(name.startsWith("@/")||name.startsWith("."))return load(name.startsWith("@/")?name.slice(2):resolve(dirname(file),name));
  return require(name);
 }});return exports;
}
const {ProductAvailabilityManager}=load("modules/menu/availability/presentation/product-availability-manager");
const css=readdirSync(resolve(root,".next/static/css")).filter(name=>name.endsWith(".css")).map(name=>readFileSync(resolve(root,".next/static/css",name),"utf8")).join("\n");
assert.match(css,/availability-quota-editor/);
const font=css.match(/@font-face\{[^}]*font-family:Manrope;[^}]*unicode-range:u\+00\?\?[^}]*\}/)?.[0]?.match(/media\/([^/)]+\.woff2)/)?.[1];assert.ok(font);
const output=resolve(process.argv[2]);mkdirSync(output,{recursive:true});
const server=createServer((req,res)=>{
 if(req.url==="/font.woff2"){res.setHeader("content-type","font/woff2");res.end(readFileSync(resolve(root,".next/static/media",font)));return;}
 mode=new URL(req.url,"http://localhost").searchParams.get("mode")??"normal";
 const markup=renderToStaticMarkup(React.createElement(ProductAvailabilityManager));
 res.setHeader("content-type","text/html; charset=utf-8");res.end(`<!doctype html><html lang="es"><head><meta name="viewport" content="width=device-width,initial-scale=1"><style>@font-face{font-family:Manrope;font-weight:200 800;src:url(/font.woff2)}:root{--font-manrope:Manrope}${css}.availability-preview{padding:32px;max-width:1600px;margin:auto}@media(width<=600px){.availability-preview{padding:16px}}</style></head><body><main class="availability-preview">${markup}</main></body></html>`);
});
await new Promise(resolve=>server.listen(0,"127.0.0.1",resolve));
const {chromium}=require(process.env.FOODS_PLAYWRIGHT_PATH??"playwright");let browser;
const issues=[];
try{
 browser=await chromium.launch({executablePath:process.env.FOODS_BROWSER_PATH,headless:true});
 for(const width of [390,768,1440,1546])for(const state of ["normal","edited","long","readonly","loading","error","empty"]){
  const page=await browser.newPage({viewport:{width,height:900},reducedMotion:"reduce"});await page.goto(`http://127.0.0.1:${server.address().port}/?mode=${state}`);await page.evaluate(()=>document.fonts.ready);
  if(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth))issues.push(`${state} ${width}: page overflow`);
  const smallTargets=await page.locator("input,select,button").evaluateAll(elements=>elements.flatMap(el=>{const r=(el.closest(".ds-input-shell")??el).getBoundingClientRect();return r.width>=44&&r.height>=44?[]:[`${el.className}: ${r.width}×${r.height}`]}));
  if(smallTargets.length)issues.push(`${state} ${width}: touch targets ${smallTargets.join("; ")}`);
  assert.equal(await page.locator(".availability-panel").evaluate(el=>getComputedStyle(el).boxShadow),"none","Border-only panel survives compiled cascade");
  if(width===390)assert.ok(await page.locator(".availability-toolbar select").evaluate(el=>parseFloat(getComputedStyle(el).fontSize)>=16),"Touch select typography");
  if(!["loading","error","empty","readonly"].includes(state))assert.ok(await page.locator(".availability-soldout-action").evaluateAll(elements=>{
   const probe=document.createElement("span");probe.style.color="var(--warning)";document.body.append(probe);const warning=getComputedStyle(probe).color;probe.remove();
   return elements.every(el=>{const style=getComputedStyle(el);return style.color===warning&&style.borderTopColor===warning});
  }),"Warning action survives compiled cascade");
  if(!["loading","error","empty"].includes(state)){
   assert.equal(await page.locator(".availability-list-head>span").count(),5);
   if(!await page.locator(".availability-card").evaluateAll(elements=>elements.every(el=>el.scrollWidth<=el.clientWidth+1)))issues.push(`${state} ${width}: card overflow`);
   if(!await page.locator(".availability-cell,.availability-product,.availability-balance span,.availability-balance strong").evaluateAll(elements=>elements.every(el=>el.scrollWidth<=el.clientWidth+1)))issues.push(`${state} ${width}: content overflow`);
   if(width<=768)assert.equal(await page.locator(".availability-list-head").isVisible(),false);
   if(state==="edited")assert.equal(await page.getByRole("button",{name:"Guardar cupo de Arroz con Mariscos",exact:true}).count(),1);
   if(state==="readonly")assert.equal(await page.getByRole("button",{name:/Marcar .* como/}).count(),0);
  }
  if(state==="normal"||width===390)await page.screenshot({path:resolve(output,`availability-${state}-${width}.png`),fullPage:true});
  if(state==="normal"&&width===1440){
   const action=page.locator(".availability-soldout-action").first();await action.hover();
   assert.ok(await action.evaluate(async el=>{
    await new Promise(requestAnimationFrame);await Promise.all(el.getAnimations().map(animation=>animation.finished));
    const probe=document.createElement("span");probe.style.background="var(--warning-50)";document.body.append(probe);const expected=getComputedStyle(probe).backgroundColor;probe.remove();
    return getComputedStyle(el).backgroundColor===expected;
   }),"Warning hover remains soft, never indigo");
  }
  console.log(`${width}px ${state}: verificado`);await page.close();
 }
 console.log(issues.join("\n"));assert.equal(issues.length,0,"Visual geometry checks");
}finally{await browser?.close();await new Promise(resolve=>server.close(resolve));}
