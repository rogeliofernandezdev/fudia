// Real ProductDialog layout with illustrative data; no API or persisted writes.
// Only the rich-text editor internals are substituted: this checks modal geometry.
import {readFileSync,readdirSync,existsSync,mkdirSync,writeFileSync} from "node:fs";
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
const {QueryClient,QueryClientProvider}=require("@tanstack/react-query");
function load(path){
 const base=resolve(root,"src",path),file=[base,`${base}.tsx`,`${base}.ts`].find(candidate=>existsSync(candidate)&&/\.tsx?$/.test(candidate));
 assert.ok(file,path);if(cache.has(file))return cache.get(file);
 const exports={};cache.set(file,exports);
 const code=ts.transpileModule(readFileSync(file,"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,esModuleInterop:true}}).outputText;
 vm.runInNewContext(code,{exports,require:name=>{
  if(name.endsWith(".css"))return {};
  if(name==="@/design-system")return {...load("design-system/page-header"),...load("design-system/modal"),...load("design-system/form-field")};
  if(name==="@/design-system/rich-text-editor")return {RichTextEditor:()=>React.createElement("div",{className:"rich-text-editor"},"Editor ilustrativo para la prueba de distribución.")};
  if(name.startsWith("@/")||name.startsWith("."))return load(name.startsWith("@/")?name.slice(2):resolve(dirname(file),name));
  return require(name);
 }});return exports;
}
const {ProductDialog}=load("modules/menu/products/presentation/product-dialog");
const draft={id:"preview",name:"Arroz con Mariscos",price:"28",costPrice:"",prepMinutes:"",categoryId:"preview",description:"",quantityControl:"none",allergens:[],featured:false,active:true,productType:"prepared",imageUrl:null};
const css=readdirSync(resolve(root,".next/static/css")).filter(name=>name.endsWith(".css")).map(name=>readFileSync(resolve(root,".next/static/css",name),"utf8")).join("\n");
const font=css.match(/@font-face\{[^}]*font-family:Manrope;[^}]*unicode-range:u\+00\?\?[^}]*\}/)?.[0]?.match(/media\/([^/)]+\.woff2)/)?.[1];
assert.ok(font,"Ejecuta build antes de revisar el CSS final.");
const output=resolve(process.argv[2]??resolve(root,".next/product-modal-preview"));mkdirSync(output,{recursive:true});
const server=createServer((req,res)=>{
 if(req.url==="/font.woff2"){res.setHeader("content-type","font/woff2");res.end(readFileSync(resolve(root,".next/static/media",font)));return;}
 const busy=new URL(req.url,"http://localhost").searchParams.has("busy");
 const markup=renderToStaticMarkup(React.createElement(QueryClientProvider,{client:new QueryClient({defaultOptions:{queries:{retry:false}}})},React.createElement(ProductDialog,{draft,categories:[{id:"preview",name:"Segundos",active:true,productScope:"both"}],destinations:[{value:"kitchen",label:"Cocina"}],loadAllergens:async()=>[],currencySymbol:"S/",busy,close:()=>{},save:async()=>{}})));
 res.setHeader("content-type","text/html; charset=utf-8");
 res.end(`<!doctype html><html lang="es"><head><meta name="viewport" content="width=device-width,initial-scale=1,viewport-fit=cover"><style>@font-face{font-family:Manrope;font-weight:200 800;src:url(/font.woff2)}:root{--font-manrope:Manrope}${css}</style></head><body><main>Prueba visual: datos ilustrativos, sin guardar.</main>${markup}</body></html>`);
});
await new Promise(resolve=>server.listen(0,"127.0.0.1",resolve));
const {chromium}=require(process.env.FOODS_PLAYWRIGHT_PATH??"playwright");let browser;
try{
 browser=await chromium.launch({executablePath:process.env.FOODS_BROWSER_PATH,headless:true});const results=[];
 for(const [name,width,height] of [["desktop",1440,900],["mobile",390,844],["short",390,480]]){
  let idle;
  for(const busy of [false,true]){
   const context=await browser.newContext({viewport:{width,height},hasTouch:width===390,reducedMotion:"reduce"});
   const page=await context.newPage();await page.goto(`http://127.0.0.1:${server.address().port}/${busy?"?busy":""}`);await page.evaluate(()=>document.fonts.ready);
   const result=await page.evaluate(busy=>{
    const modal=document.querySelector(".crud-modal"),header=modal.querySelector("header"),body=modal.querySelector(".modal-body"),footer=modal.querySelector(".modal-footer"),close=modal.querySelector(".modal-close"),buttons=[...footer.querySelectorAll("button")];
    const rect=el=>{const r=el.getBoundingClientRect();return {x:r.x,y:r.y,width:r.width,height:r.height,right:r.right,bottom:r.bottom};},style=el=>getComputedStyle(el);
    const inset=innerWidth<=600?16:24;
    if(!modal.querySelector(".modal-form"))throw new Error("Producto debe usar ModalForm");
    if([header,body,footer].some(el=>parseFloat(style(el).paddingLeft)!==inset||parseFloat(style(el).paddingRight)!==inset))throw new Error("Cabecera, cuerpo y pie deben compartir 24/16 px");
    if(Math.abs(rect(footer).width-rect(modal).width)>.5||rect(footer).x<rect(modal).x||rect(footer).right>rect(modal).right+.5)throw new Error("Pie sin márgenes negativos ni recortes");
    if(Math.abs(rect(modal).right-rect(buttons[1]).right-inset)>.5||Math.abs(rect(buttons[1]).x-rect(buttons[0]).right-10)>.5)throw new Error("Acciones contiguas: 10 px entre ellas y margen final estándar");
    if(Math.abs(rect(buttons[0]).y-rect(buttons[1]).y)>.5||rect(buttons[0]).height!==rect(buttons[1]).height||Math.abs(rect(footer).bottom-rect(buttons[1]).bottom-16)>.5)throw new Error("Altura y espacio inferior constantes");
    if(widthIsMobile()&&(Math.abs(rect(buttons[0]).width-rect(buttons[1]).width)>.5||buttons.some(button=>rect(button).height<44||parseFloat(style(button).borderRadius)<22)))throw new Error("Dos acciones móviles iguales, redondeadas y táctiles");
    function widthIsMobile(){return innerWidth<=600;}
    if(close.disabled!==busy||buttons.some(button=>button.disabled!==busy))throw new Error("Cierres y guardado bloqueados durante la petición");
    const appearance={color:style(close).color,opacity:style(close).opacity,border:getComputedStyle(close,"::before").borderTopColor,background:style(close).backgroundColor};
    if(appearance.color!=="rgb(255, 255, 255)"||appearance.border!==appearance.color||appearance.opacity!=="1"||appearance.background!=="rgba(0, 0, 0, 0)")throw new Error("X y contorno blancos, opacidad completa incluso guardando");
    const before=rect(footer);body.scrollTop=body.scrollHeight;if(rect(footer).y!==before.y)throw new Error("El pie no se desplaza con el cuerpo");
    if(rect(modal).bottom>innerHeight+.5||document.documentElement.scrollWidth>innerWidth)throw new Error("Modal fuera del viewport");
    return {footer:rect(footer),buttons:buttons.map(rect),appearance,scrolls:body.scrollHeight>body.clientHeight};
   },busy);
   if(!busy)idle=result;else {assert.deepEqual(result.footer,idle.footer);assert.deepEqual(result.appearance,idle.appearance);}
   await page.screenshot({path:resolve(output,`${name}-${busy?"saving":"idle"}.png`),fullPage:true});
   results.push({name,busy,...result});console.log(`${name} ${busy?"guardando":"normal"}: márgenes, acciones y X correctos`);await context.close();
  }
 }
 writeFileSync(resolve(output,"geometry.json"),JSON.stringify(results,null,2));
}finally{await browser?.close();await new Promise(resolve=>server.close(resolve));}
