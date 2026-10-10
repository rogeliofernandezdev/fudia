// Render production primitives without authentication or business mutations.
// Usage: node tests/helpers/shared-ui-preview.mjs <artifact-directory>
import {readFileSync, mkdirSync, readdirSync, writeFileSync} from "node:fs";
import {createRequire} from "node:module";
import {dirname, resolve} from "node:path";
import {fileURLToPath} from "node:url";
import {createServer} from "node:http";
import vm from "node:vm";
import ts from "typescript";
import React from "react";
import {renderToStaticMarkup} from "react-dom/server";
import assert from "node:assert/strict";

const require=createRequire(import.meta.url);
const root=resolve(dirname(fileURLToPath(import.meta.url)),"../..");
const cache=new Map();
function load(path){
 const file=resolve(root,"src",path);
 if(cache.has(file))return cache.get(file);
 const exports={};cache.set(file,exports);
 const source=ts.transpileModule(readFileSync(file,"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,esModuleInterop:true}}).outputText;
 vm.runInNewContext(source,{exports,require:name=>{
  if(name.endsWith(".css"))return {};
  if(name.startsWith(".")||name.startsWith("@/")){
   const target=name.startsWith("@/")?resolve(root,"src",name.slice(2)):resolve(dirname(file),name);
   return load(target+".tsx");
  }
  return require(name);
 }});
 return exports;
}
const {Tabs}=load("design-system/tabs.tsx");
const {PageHeader,Button,Input,Select}=load("design-system/page-header.tsx");
const {Modal,ModalBody,ModalForm,ModalFooter,ModalCancel}=load("design-system/modal.tsx");
const {FormField}=load("design-system/form-field.tsx");
const h=React.createElement;
const sourceCss=["styles/variables.css","styles/globals.css","design-system/styles/tabs.css","design-system/styles/modal.css","design-system/styles/form-field.css","design-system/styles/form-controls.css"].map(path=>readFileSync(resolve(root,"src",path),"utf8").replace(/^@import[^;]+;/gm,"")).join("\n");
const mediaDir=resolve(root,".next/static/media");
const fontCss=readdirSync(resolve(root,".next/static/css")).filter(name=>name.endsWith(".css")).map(name=>readFileSync(resolve(root,".next/static/css",name),"utf8")).join("\n");
const css=process.env.FOODS_PREVIEW_COMPILED_CSS==="1"?fontCss:sourceCss;
const latinFace=fontCss.match(/@font-face\{[^}]*font-family:Manrope;[^}]*unicode-range:u\+00\?\?[^}]*\}/)?.[0];
const font=latinFace?.match(/media\/([^/)]+\.woff2)/)?.[1];
assert.ok(font,"Genera el build para cargar la fuente Manrope real del frontend.");
const noop=()=>{};
function fixture(showModal,long,typography=false){
 const tabs=h(Tabs,{label:"Carta y productos",value:"products",onChange:noop,items:[{value:"products",label:"Productos",count:1234},{value:"categories",label:"Categorías",count:12}]});
 const fields=h("div",{className:"form-grid"},
  h(FormField,{label:"Nombre",className:"span-2"},h(Input,{defaultValue:"Bebidas",placeholder:"Ej. Bebidas"})),
  h(FormField,{label:"Disponible para"},h(Select,{defaultValue:"both"},h("option",{value:"both"},"Platos y bebidas"))),
  h(FormField,{label:"Orden de aparición",help:"Menor número aparece primero en la carta."},h(Input,{type:"number",defaultValue:1})),
  ...(long?Array.from({length:20},(_,i)=>h(FormField,{key:i,label:`Dato adicional ${i+1}`,className:"span-2"},h(Input,{defaultValue:"Contenido de prueba para comprobar el desplazamiento"}))):[]));
 return renderToStaticMarkup(h(React.Fragment,null,
  h("main",{style:{padding:"24px"}},h(PageHeader,{title:"Carta y productos",description:"Gestiona la oferta de tu restaurante."}),h("div",{className:"catalog-tabs-row"},tabs,h(Button,{icon:"plus"},"Nuevo producto")),h("div",{className:"panel",style:{padding:"24px"}},"Vista de comprobación de componentes compartidos; datos ilustrativos.")),
  showModal&&h(Modal,{title:typography?"Jerarquía compartida de datos":long?"Editar categoría con un nombre extenso que conserva toda su información":"Nueva categoría",subtitle:typography?"Datos ilustrativos; sin consultas ni escrituras":"Carta y productos",icon:"dish",size:long||typography?"lg":"md",onClose:noop},h(ModalForm,{onSubmit:noop},h(ModalBody,null,fields,typography&&h("div",{style:{display:"grid",gap:24,marginTop:24}},
   ...["pos-detail-context","sales-detail-context","expense-detail-grid","company-detail-fields","cash-report-identity","order-attribution","product-review","combo-review-section"].map(className=>h("section",{key:className},h("h3",null,className),h("dl",{className},h("div",null,h("dt",{"data-type-label":true},"Atendido por"),h("dd",{"data-type-value":true},"Daniel Machaca Rojasa Fernández de los Ríos"))))),
   ...["customer-detail-fields","purchase-detail-meta","company-detail-summary","combo-detail-availability","order-detail-context"].map(className=>h("section",{key:className},h("h3",null,className),h("div",{className},h("div",null,h("small",{"data-type-label":true},"Descripción"),h("b",{"data-type-value":true},"Contenido ilustrativo que conserva toda su información"))))),
  )),h(ModalFooter,null,h(ModalCancel),h(Button,{type:"submit",icon:"check"},"Guardar"))))));
}
const out=resolve(process.argv[2]??resolve(root,".next/shared-ui-preview"));mkdirSync(out,{recursive:true});
const server=createServer((req,res)=>{
 if(req.url==="/font.woff2"){res.setHeader("content-type","font/woff2");res.end(readFileSync(resolve(mediaDir,font)));return;}
 const url=new URL(req.url,"http://localhost");
 res.setHeader("content-type","text/html; charset=utf-8");
 res.end(`<!doctype html><html lang="es"><head><meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover"><style>@font-face{font-family:Manrope;font-weight:200 800;src:url(/font.woff2)}:root{--font-manrope:Manrope}${css}</style></head><body>${fixture(url.searchParams.has("modal"),url.searchParams.has("long"),url.searchParams.has("typography"))}</body></html>`);
});
await new Promise(resolve=>server.listen(0,"127.0.0.1",resolve));
const {chromium}=require(process.env.FOODS_PLAYWRIGHT_PATH??"playwright");
let browser;
try{
 browser=await chromium.launch({executablePath:process.env.FOODS_BROWSER_PATH,headless:true});
 const results=[];
 for(const [name,width,height,touch,query] of [["tabs-desktop",1440,900,false,""],["tabs-mobile",390,844,true,""],["modal-desktop",1440,900,false,"?modal"],["modal-mobile",390,844,true,"?modal"],["modal-short",390,480,true,"?modal&long"],["modal-tablet",768,1024,true,"?modal&long"],["typography-desktop",1440,900,false,"?modal&typography"],["typography-mobile",390,844,true,"?modal&typography"]]){
  const context=await browser.newContext({viewport:{width,height},hasTouch:touch,reducedMotion:"reduce"});
  const page=await context.newPage();await page.goto(`http://127.0.0.1:${server.address().port}/${query}`);await page.evaluate(()=>document.fonts.ready);
  await page.evaluate(()=>Promise.all(document.getAnimations().filter(animation=>animation.effect?.getTiming().iterations!==Infinity).map(animation=>animation.finished.catch(()=>{}))));
  results.push({name,...await page.evaluate(()=>{
   const rect=el=>{const r=el.getBoundingClientRect();return {x:r.x,y:r.y,width:r.width,height:r.height,right:r.right,bottom:r.bottom};};
   const modal=document.querySelector(".crud-modal"),body=document.querySelector(".modal-body"),footer=document.querySelector(".modal-footer");
   if(body)body.scrollTop=body.scrollHeight;
   const style=el=>getComputedStyle(el);
   const outside=document.createElement("input");outside.className="ds-input";document.body.append(outside);
   const outsideWeight=style(outside).fontWeight;outside.remove();
   if(outsideWeight!=="500")throw new Error("La tipografía de controles fuera del modal no debe cambiar");
   if(modal){
    const header=modal.querySelector("header");
    const close=header.querySelector("button"),icon=header.querySelector(".modal-title-icon");
    const closeOutline=getComputedStyle(close,"::before");
    if(closeOutline.width!=="32px"||closeOutline.height!=="32px"||closeOutline.borderTopWidth!=="1px"||closeOutline.borderTopColor!==style(header.querySelector("h2")).color||style(close).backgroundColor!=="rgba(0, 0, 0, 0)"||style(icon).backgroundColor!=="rgba(0, 0, 0, 0)")throw new Error("Icono sin recuadro y X con solo contorno blanco compacto de 32 px");
    if(rect(close).width<44||rect(close).height<44)throw new Error("La X debe conservar 44 px también en escritorio");
    for(const glyph of [icon.querySelector("svg"),close.querySelector("svg")]){
     if(rect(glyph).width!==20||rect(glyph).height!==20)throw new Error("Glifos de 20 px en la cabecera");
    }
    if(style(icon).color!==style(header.querySelector("h2")).color||style(close).color!==style(header.querySelector("h2")).color)throw new Error("Icono y X blancos de la paleta");
    if(modal.querySelector(".modal-accent")||Math.abs(rect(header).y-rect(modal).y)>0.5||Math.abs(rect(header).width-rect(modal).width)>0.5)throw new Error("Cabecera sin ocupar todo el borde superior");
    const luminance=color=>{
     const rgb=color.match(/[\d.]+/g).slice(0,3).map(Number).map(value=>value/255).map(value=>value<=.04045?value/12.92:((value+.055)/1.055)**2.4);
     return rgb[0]*.2126+rgb[1]*.7152+rgb[2]*.0722;
    };
    const background=luminance(style(header).backgroundColor);
    for(const label of header.querySelectorAll("h2,p,small")){
     const foreground=luminance(style(label).color);
     if((Math.max(foreground,background)+.05)/(Math.min(foreground,background)+.05)<4.5)throw new Error("Cabecera con contraste insuficiente");
    }
    if(header.querySelector("small,p,time,.status"))throw new Error("La cabecera solo admite icono, título y X");
    const actions=[...footer.querySelectorAll("button")].map(rect);
    if(actions.length===2&&(Math.abs(actions[0].y-actions[1].y)>1||Math.abs(actions[1].x-actions[0].right-10)>1))throw new Error("Cancelar y Guardar deben quedar juntos con 10 px, no en extremos");
    if(!location.search.includes("long")&&rect(header).height>65)throw new Error("Cabecera breve sin compactar");
    for(const label of modal.querySelectorAll("[data-type-label],.form-field-label")){
     if(style(label).fontWeight!=="600")throw new Error("Las etiquetas deben usar seminegrita: "+label.textContent);
    }
    for(const value of modal.querySelectorAll("[data-type-value],input,select")){
     if(style(value).fontWeight!=="400")throw new Error("Los valores deben usar peso regular: "+value.textContent);
     if(value.scrollWidth>value.clientWidth+1&&value.matches("[data-type-value]"))throw new Error("Un dato largo quedó recortado");
    }
   }
   return {overflow:document.documentElement.scrollWidth>innerWidth,tabs:[...document.querySelectorAll("[role=tab]")].map(rect),modal:modal&&rect(modal),close:modal&&rect(modal.querySelector("header button")),body:body&&{...rect(body),scrolls:body.scrollHeight>body.clientHeight},footer:footer&&rect(footer),buttons:footer&&[...footer.querySelectorAll("button")].map(rect),insets:modal&&[style(modal.querySelector("header")).paddingLeft,style(body).paddingLeft,style(footer).paddingLeft],selection:style(document.querySelector("[role=tab][aria-selected=true]")).backgroundColor};
  })});
  await page.screenshot({path:resolve(out,`${name}.png`),fullPage:true});
  if(query.includes("modal")){
   await page.keyboard.press("Tab");await page.locator(".modal-close").focus();
   assert.equal(await page.locator(".modal-close").evaluate(el=>getComputedStyle(el).outlineWidth),"2px",`${name}: foco de teclado visible`);
   if(!touch){
    const before=await page.locator(".modal-close").evaluate(el=>({background:getComputedStyle(el).backgroundColor,color:getComputedStyle(el).color}));
    await page.locator(".modal-close").hover();await page.waitForTimeout(180);
    assert.deepEqual(await page.locator(".modal-close").evaluate(el=>({background:getComputedStyle(el).backgroundColor,color:getComputedStyle(el).color})),before,`${name}: hover sin cambio de color`);
   }
  }
  await context.close();
 }
 writeFileSync(resolve(out,"geometry.json"),JSON.stringify(results,null,2));
 for(const result of results){
  assert.equal(result.overflow,false,result.name);
  if(result.modal){
   const height=result.name==="modal-short"?480:result.name==="modal-tablet"?1024:result.name.endsWith("mobile")?844:900;
   assert.ok(result.modal.bottom<=height,`${result.name}: panel dentro del viewport`);
   assert.ok(result.footer.bottom<=height,`${result.name}: acciones visibles tras scroll`);
   assert.equal(new Set(result.insets).size,1,`${result.name}: cabecera, cuerpo y pie alineados`);
   if(result.name!=="modal-desktop")assert.ok(result.close.width>=44&&result.close.height>=44,`${result.name}: cierre táctil`);
  }
  console.log(`${result.name}: sin desborde${result.modal?", cierre y acciones visibles, márgenes alineados":""}`);
 }
}finally{await browser?.close();await new Promise(resolve=>server.close(resolve));}
