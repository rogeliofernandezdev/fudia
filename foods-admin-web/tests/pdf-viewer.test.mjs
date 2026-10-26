import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createHash} from "node:crypto";
import vm from "node:vm";
import postcss from "postcss";

const root=new URL("../",import.meta.url),read=path=>readFileSync(new URL(path,root),"utf8");
const runtime="src/shared/pdf-viewer/runtime/";
const tick=()=>new Promise(resolve=>setImmediate(resolve));

function setup(){
 const handlers=new Map(),elements=new Map(),messages=[],calls=[];let resizeActions;
 const target=()=>({addEventListener(type,handler,options={}){const list=handlers.get(this)??[];list.push({type,handler,options});handlers.set(this,list);}});
 function element(id=""){
  const classes=new Set(),attributes={};const el={...target(),id,children:[],parentElement:{classList:{remove(){}}},classList:{add:value=>classes.add(value),remove:value=>classes.delete(value)},setAttribute:(key,value)=>{attributes[key]=value;},removeAttribute:key=>{delete attributes[key];},append(...nodes){this.children.push(...nodes);},querySelector:()=>el.span,textContent:"",attributes,classes};
  el.span={removeAttribute(){},textContent:""};return el;
 }
 for(const id of ["printButton","downloadButton","previous","viewBookmark","secondaryToolbarButtonContainer"])elements.set(id,element(id));
 const document={...target(),body:element("body"),documentElement:{style:{setProperty:(...args)=>calls.push(["style",...args])}},createElement:()=>({...element(),getBoundingClientRect:()=>({height:140})}),getElementById:id=>elements.get(id),fullscreenElement:null};
 const parent={document:target(),postMessage:message=>messages.push(message)};
 const options={setAll:value=>calls.push(["options",value])};
 const app={initializedPromise:Promise.resolve(),eventBus:{on(){},dispatch:(...args)=>calls.push(["dispatch",...args])},pdfDocument:{numPages:7},pdfViewer:{printingAllowed:true},supportsPrinting:true,
  open:async args=>calls.push(["open",args]),close:async()=>calls.push(["close"]),unbindEvents:()=>calls.push(["unbindEvents"]),unbindWindowEvents:()=>calls.push(["unbindWindowEvents"]),
  triggerPrinting:async()=>calls.push(["print"]),afterPrint:()=>calls.push(["afterPrint"])};
 const window={...target(),PDFViewerApplication:app,PDFViewerApplicationOptions:options};
 const revoked=[];let nextUrl=0;
 const URL={createObjectURL:()=>`blob:fixture-${++nextUrl}`,revokeObjectURL:url=>revoked.push(url)};
 const ResizeObserver=class{constructor(callback){resizeActions=callback;}observe(){}disconnect(){calls.push(["disconnectObserver"]);}};
 const context={window,parent,document,URL,ResizeObserver,location:{origin:"http://fudia.test"},matchMedia:()=>({matches:true}),AbortController,ArrayBuffer,Uint8Array,console};
 vm.runInNewContext(read(runtime+"fudia-bridge.mjs"),context);
 const emit=async(target,type,event={})=>{for(const entry of handlers.get(target)??[])if(entry.type===type&&!entry.options.signal?.aborted)await entry.handler(event);};
 const message=(type,patch={})=>emit(window,"message",{source:parent,origin:context.location.origin,data:{channel:"fudia-pdf-viewer",session:"current",type,...patch}});
 return {window,parent,document,elements,messages,calls,app,emit,message,URL,revoked,resizeActions:()=>resizeActions(),initialize:async()=>{await emit(parent.document,"webviewerloaded",{detail:{source:window}});await tick();}};
}

test("PDF.js: visor oficial completo, motor y worker legacy tienen la misma versión local",()=>{
 const manifest=JSON.parse(read("vendor/pdfjs/manifest.json")),pkg=JSON.parse(read("package.json"));
 assert.equal(pkg.dependencies["pdfjs-dist"],manifest.version);
 assert.ok(read("vendor/pdfjs/web/viewer.mjs").includes(`pdfjsVersion = ${manifest.version}`));
 assert.match(read("vendor/pdfjs/web/viewer.mjs"),/class PDFPrintService/);
 for(const file of ["pdf.mjs","pdf.worker.mjs"]){
  const hash=path=>createHash("sha256").update(readFileSync(new URL(path,root))).digest("hex");
  assert.equal(hash(`public/pdfjs/build/${file}`),hash(`node_modules/pdfjs-dist/legacy/build/${file}`));
 }
 const html=read("public/pdfjs/web/viewer.html");
 for(const value of ["fudia-bridge.mjs","fudia-viewer.css","viewer.mjs","locale/locale.json"])assert.ok(html.includes(value));
 assert.ok(html.includes("connect-src 'self' blob: data:"));assert.ok(!html.includes("maximum-scale=1"));
 assert.ok(read("vendor/pdfjs/LICENSE").includes("Apache License"));
});

test("visor: transferencia de bytes solo desde el padre y el mismo origen/sesión",async()=>{
 const s=setup();await s.initialize();
 await s.emit(s.window,"message",{origin:"https://externo.test",source:s.parent,data:{channel:"fudia-pdf-viewer",session:"evil",type:"connect"}});
 assert.equal(s.messages.length,0);
 await s.message("connect");assert.equal(s.messages[0].type,"connected");
 await s.emit(s.window,"message",{origin:"http://fudia.test",source:{},data:{channel:"fudia-pdf-viewer",session:"current",type:"open",data:new ArrayBuffer(4)}});
 await s.message("open",{session:"old",data:new ArrayBuffer(4)});
 assert.equal(s.calls.filter(call=>call[0]==="open").length,0);
 const data=new ArrayBuffer(5);await s.message("open",{data,filename:"ticket.pdf"});
 const opened=s.calls.find(call=>call[0]==="open")[1];assert.equal(opened.data.buffer,data);assert.equal(opened.filename,"ticket.pdf");assert.equal("url" in opened,false);
 assert.equal(s.messages.at(-1).type,"loaded");assert.equal(s.messages.at(-1).pages,7);
});

test("visor: impresión delegada al servicio de PDF.js y fallo permite descargar",async()=>{
 const s=setup();await s.initialize();await s.message("connect");
 let prevented=false;await s.emit(s.elements.get("printButton"),"click",{stopImmediatePropagation:()=>{prevented=true;}});
 assert.equal(prevented,true);assert.equal(s.calls.filter(call=>call[0]==="print").length,1);
 s.app.supportsPrinting=false;await s.emit(s.elements.get("printButton"),"click",{stopImmediatePropagation(){}});
 const footer=s.document.body.children[0],hint=footer.children[1];assert.ok(hint.textContent.includes("Descargar PDF"));
 s.app.supportsPrinting=true;s.app.triggerPrinting=async()=>{throw new Error("not supported");};
 await s.emit(s.elements.get("printButton"),"click",{stopImmediatePropagation(){}});
 assert.ok(hint.textContent.includes("No se pudo abrir"));
 s.app.printService={renderPages:()=>Promise.reject(new Error("out of memory"))};
 await s.emit(s.window,"beforeprint");await assert.rejects(s.app.printService.renderPages());
 assert.ok(hint.textContent.includes("No se pudo preparar"));
 assert.equal(s.elements.get("downloadButton").span.textContent,"Descargar PDF");
});

test("visor: render progresivo, límites de canvas y sin ejecución de scripts o modelos remotos",async()=>{
 const s=setup();await s.initialize();const options=s.calls.find(call=>call[0]==="options")[1];
 assert.equal(options.disableAutoFetch,false);assert.equal(options.maxCanvasPixels,2097152);assert.equal(options.enableDetailCanvas,false);
 assert.match(read("vendor/pdfjs/web/viewer.mjs"),/class PDFRenderingQueue/);
 assert.equal(options.enableScripting,false);assert.equal(options.enablePermissions,true);assert.equal(options.enableAltTextModelDownload,false);assert.equal(options.disablePreferences,true);
 assert.equal(options.defaultUrl,"");assert.equal(options.defaultZoomValue,"page-fit");
});

test("visor: prepara metadata pendiente antes de imprimir y cancela si se cierra",async()=>{
 const s=setup();await s.initialize();
 const views=Array.from({length:7},()=>({pdfPage:null,setPdfPage(page){this.pdfPage=page;}}));
 s.app.pdfViewer.pageViewsReady=false;s.app.pdfViewer.getPageView=index=>views[index];
 s.app.pdfDocument.getPage=async page=>({pageNumber:page});
 await s.emit(s.elements.get("printButton"),"click",{stopImmediatePropagation(){}});
 assert.ok(views.every(view=>view.pdfPage));assert.equal(s.calls.filter(call=>call[0]==="print").length,1);
 const closed=setup();await closed.initialize();
 closed.app.pdfViewer.pageViewsReady=false;closed.app.pdfViewer.getPageView=()=>({setPdfPage(){}});
 closed.app.pdfDocument.getPage=async()=>{await closed.window.FudiaPdfViewer.dispose();return {};};
 await closed.emit(closed.elements.get("printButton"),"click",{stopImmediatePropagation(){}});
 assert.equal(closed.calls.some(call=>call[0]==="print"),false);
});

test("visor: un PDF dañado comunica error sin filtrar detalles internos",async()=>{
 const s=setup();await s.initialize();await s.message("connect");
 s.app.open=async()=>{throw new Error("private internal information");};
 await s.message("open",{data:new ArrayBuffer(4)});
 assert.equal(s.messages.at(-1).type,"error");assert.ok(!JSON.stringify(s.messages).includes("private internal"));
});

test("visor: el ajuste de página se recalcula al cambiar la altura real del pie",async()=>{
 const s=setup();await s.initialize();s.resizeActions();
 assert.ok(s.calls.some(call=>call[0]==="style"&&call[1]==="--pdf-document-actions-height"&&call[2]==="140px"));
 assert.ok(s.calls.some(call=>call[0]==="dispatch"&&call[1]==="resize"));
 await s.window.FudiaPdfViewer.dispose();assert.equal(s.calls.filter(call=>call[0]==="disconnectObserver").length,1);
});

test("visor: cerrar cancela impresión/documento y retira listeners; no acepta PDFs posteriores",async()=>{
 const s=setup();await s.initialize();await s.message("connect");s.app.printService={destroy:()=>s.calls.push(["destroyPrint"])};
 const temporary=s.URL.createObjectURL({});
 await s.window.FudiaPdfViewer.dispose();await s.message("open",{data:new ArrayBuffer(4)});
 assert.ok(s.revoked.includes(temporary));
 const late=s.URL.createObjectURL({});assert.ok(s.revoked.includes(late),"un callback tardío tampoco deja una URL viva");
 for(const name of ["destroyPrint","unbindEvents","unbindWindowEvents","close"])assert.equal(s.calls.filter(call=>call[0]===name).length,1,name);
 assert.equal(s.calls.filter(call=>call[0]==="open").length,0);
 await s.window.FudiaPdfViewer.dispose();assert.equal(s.calls.filter(call=>call[0]==="close").length,1);
 const parent=read("src/shared/pdf-viewer/pdf-viewer.tsx");
 assert.match(parent,/removeEventListener\("message",listener\)/);assert.match(parent,/clearTimeout\(timeout\)/);assert.match(parent,/cancelled=true/);
 assert.match(parent,/event\.source!==element\.contentWindow/);assert.match(parent,/event\.origin!==window\.location\.origin/);
 assert.match(parent,/\[data\]/,"transferible sin duplicar el buffer");
});

test("visor: acciones siempre visibles, áreas táctiles, sin impresión de la pantalla de FudIA",()=>{
 const bridge=read(runtime+"fudia-bridge.mjs"),css=read("src/design-system/styles/pdfjs-viewer.css"),modal=read("src/shared/pdf-viewer/pdf-viewer.tsx");
 assert.doesNotMatch(bridge,/window\.print\(|window\.open\(|Abrir externo|https:\/\//);
 assert.doesNotMatch(modal,/window\.print\(|<Button[^>]*print/);
 assert.match(bridge,/Todas las impresoras/);assert.match(bridge,/app\.triggerPrinting\(\)/);
 assert.match(css,/@media print\{#fudiaDocumentActions\{display:none!important\}\}/);
 assert.match(css,/min-height:var\(--size-44\)/);assert.match(css,/\.fudia-document-action\{width:100%/);
 assert.match(read("src/shared/pdf-viewer/index.tsx"),/ssr:false/);
 assert.match(read("src/app/layout.tsx"),/import "@\/design-system\/styles\/pdf-viewer\.css"/,
  "el tamaño del iframe debe cargarse con los estilos globales, antes del chunk dinámico");
 const seen=new Set();postcss.parse(css).walkRules(rule=>{let scope="";for(let parent=rule.parent;parent;parent=parent.parent)if(parent.type==="atrule")scope+=parent.params;for(const selector of rule.selectors){const key=scope+selector;assert.equal(seen.has(key),false,key);seen.add(key);}});
});
