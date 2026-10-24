// Real feedback presentation with illustrative data; no API calls or persisted writes.
import {readFileSync,readdirSync,mkdirSync,writeFileSync} from "node:fs";
import {createRequire} from "node:module";
import {dirname,resolve} from "node:path";
import {fileURLToPath} from "node:url";
import {createServer} from "node:http";
import vm from "node:vm";
import ts from "typescript";
import React from "react";
import {renderToStaticMarkup} from "react-dom/server";
import assert from "node:assert/strict";

const require=createRequire(import.meta.url),root=resolve(dirname(fileURLToPath(import.meta.url)),"../..");
function render(tone,closing){
 const cache=new Map();let slot=0;
 const item={id:1,tone,title:tone==="success"?"Producto actualizado":tone==="danger"?"No se pudo registrar":"Información del pedido",message:tone==="success"?"Los cambios ya están disponibles en el catálogo.":"Vista de prueba con datos ilustrativos. No se ha realizado ninguna operación."};
 function load(path){
  const file=resolve(root,"src",path);if(cache.has(file))return cache.get(file);
  const exports={};cache.set(file,exports);
  const code=ts.transpileModule(readFileSync(file,"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,esModuleInterop:true}}).outputText;
  vm.runInNewContext(code,{exports,require:name=>{
   if(name.endsWith(".css"))return {};
   if(name==="react"&&path==="providers/feedback-provider.tsx")return {...React,useState:()=>[slot++===0?item:closing,()=>{}]};
   if(name.startsWith("@/")||name.startsWith(".")){const target=name.startsWith("@/")?name.slice(2):resolve(dirname(file),name);return load(target+".tsx");}
   return require(name);
  }});
  return exports;
 }
 return renderToStaticMarkup(React.createElement(load("providers/feedback-provider.tsx").FeedbackProvider,null,React.createElement("main",null,"Prueba visual; datos ilustrativos.")));
}
const css=readdirSync(resolve(root,".next/static/css")).filter(name=>name.endsWith(".css")).map(name=>readFileSync(resolve(root,".next/static/css",name),"utf8")).join("\n");
assert.ok(css.includes("feedback-success-settle"),"Ejecuta el build para comprobar el CSS final.");
const font=css.match(/@font-face\{[^}]*font-family:Plus Jakarta Sans;[^}]*unicode-range:u\+00\?\?[^}]*\}/)?.[0]?.match(/media\/([^/)]+\.woff2)/)?.[1];
assert.ok(font,"Fuente Plus Jakarta Sans real del frontend.");
const output=resolve(process.argv[2]??resolve(root,".next/feedback-preview"));mkdirSync(output,{recursive:true});
const server=createServer((req,res)=>{
 if(req.url==="/font.woff2"){res.setHeader("content-type","font/woff2");res.end(readFileSync(resolve(root,".next/static/media",font)));return;}
 const url=new URL(req.url,"http://localhost");
 res.setHeader("content-type","text/html; charset=utf-8");
 res.end(`<!doctype html><html lang="es"><head><meta name="viewport" content="width=device-width, initial-scale=1"><style>@font-face{font-family:Plus Jakarta Sans;font-weight:200 800;src:url(/font.woff2)}:root{--font-jakarta:"Plus Jakarta Sans"}${css}.feedback-preview-paused *,.feedback-preview-paused *:before{animation-play-state:paused!important}</style></head><body class="${url.searchParams.has("paused")?"feedback-preview-paused":""}">${render(url.searchParams.get("tone")??"success",url.searchParams.has("out"))}</body></html>`);
});
await new Promise(resolve=>server.listen(0,"127.0.0.1",resolve));
const {chromium}=require(process.env.FOODS_PLAYWRIGHT_PATH??"playwright");let browser;
try{
 browser=await chromium.launch({executablePath:process.env.FOODS_BROWSER_PATH,headless:true});const results=[];
 for(const [name,width,height,tone,reduced,closing] of [["success-desktop",1440,900,"success",false,false],["success-mobile",390,844,"success",false,false],["success-short",390,480,"success",false,false],["success-reduced",390,844,"success",true,false],["success-exit",390,844,"success",false,true],["danger-desktop",1440,900,"danger",true,false],["info-mobile",390,844,"info",true,false]]){
  const context=await browser.newContext({viewport:{width,height},hasTouch:width===390,reducedMotion:reduced?"reduce":"no-preference"});
  const page=await context.newPage();await page.goto(`http://127.0.0.1:${server.address().port}/?tone=${tone}${closing?"&out":""}`);await page.evaluate(()=>document.fonts.ready);
  const result=await page.evaluate(({tone,reduced,closing})=>{
   const card=document.querySelector(".feedback-card"),circle=document.querySelector(".feedback-icon"),glyph=circle.querySelector("svg"),button=document.querySelector(".feedback-accept"),title=card.querySelector("h2");
   const style=el=>getComputedStyle(el),rect=el=>{const r=el.getBoundingClientRect();return {x:r.x,y:r.y,width:r.width,height:r.height,right:r.right,bottom:r.bottom};};
   const animations=document.getAnimations();animations.forEach(animation=>animation.pause());
   const sample=[];
   if(tone==="success"&&!closing){
    for(const time of [0,80,140,260,360,450,520]){
     animations.forEach(animation=>{animation.currentTime=time;});
     const scale=el=>new DOMMatrix(style(el).transform).a;
     const shape=getComputedStyle(circle,"::before"),matrix=new DOMMatrix(shape.transform);
     sample.push({time,panel:scale(card),circle:scale(circle),shapeX:Math.hypot(matrix.a,matrix.c),shapeY:Math.hypot(matrix.b,matrix.d),shapeOffsetX:matrix.e,shapeOffsetY:matrix.f,radius:shape.borderRadius,contour:shape.clipPath,checkScale:scale(glyph),checkOpacity:style(glyph).opacity,textOpacity:style(title).opacity});
    }
    if(!reduced){
     if(sample.some(s=>s.panel!==1||s.circle!==1||s.checkScale!==1))throw new Error("La capa verde no deforma el check ni el panel");
     if(Math.abs(sample[0].shapeX-1.3)>.001)throw new Error("El círculo empieza 30% más grande");
     if(sample.some(s=>Math.abs(s.shapeX-s.shapeY)>.001||s.shapeOffsetX!==0||s.shapeOffsetY!==0||s.radius!=="50%"||s.contour!=="none"))throw new Error("Círculo perfecto y centrado en toda la entrada");
     if(!sample.some(s=>s.shapeX<.995)||sample.some(s=>s.shapeX<.96||s.shapeX>1.301))throw new Error("Rebote elástico leve y acotado");
     if(Math.abs(sample.at(-1).shapeX-1)>.001||Math.abs(sample.at(-1).shapeY-1)>.001||sample.at(-1).checkOpacity!=="1")throw new Error("El círculo se asienta a los 520 ms");
     if(Number(sample[5].checkOpacity)<=0||sample.some(s=>s.textOpacity!=="1"))throw new Error("Check fundido y textos sin esperas propias");
    }else if(sample.some(s=>s.panel!==1||s.circle!==1||s.shapeX!==1||s.shapeY!==1||s.shapeOffsetY!==0))throw new Error("Movimiento reducido sin deformación");
   }
   // Inspect the stable presentation (and the start of an exit), not a transparent frame.
   animations.forEach(animation=>{animation.currentTime=closing&&animation.effect?.target?.matches?.(".feedback-card,.feedback-overlay")?0:1000;});
   if(tone==="success"){
    if(style(circle).color!=="rgb(255, 255, 255)"||style(glyph).color!==style(circle).color)throw new Error("Check blanco de la paleta");
    const canvas=document.createElement("canvas"),ctx=canvas.getContext("2d");ctx.fillStyle=style(document.documentElement).getPropertyValue("--brand-700").trim();const green=ctx.fillStyle;
    const rgb=color=>color.startsWith("#")?[1,3,5].map(i=>parseInt(color.slice(i,i+2),16)):color.match(/[\d.]+/g).slice(0,3).map(Number);
    const lum=color=>rgb(color).map(v=>v/255).map(v=>v<=.04045?v/12.92:((v+.055)/1.055)**2.4).reduce((sum,v,i)=>sum+v*[.2126,.7152,.0722][i],0);
    if(lum(getComputedStyle(circle,"::before").backgroundColor)!==lum(green)||(1.05/(lum(green)+.05))<3)throw new Error("Círculo verde sólido y contraste gráfico accesible");
    if(Math.abs(rect(glyph).x+rect(glyph).width/2-(rect(circle).x+rect(circle).width/2))>.5||Math.abs(rect(glyph).y+rect(glyph).height/2-(rect(circle).y+rect(circle).height/2))>.5)throw new Error("Icono centrado");
    if(rect(circle).y-rect(card).y<64||rect(circle).y<rect(card.querySelector(".feedback-close")).bottom)throw new Error("Círculo debajo de la fila de cierre y separado del borde superior");
    if(style(button).backgroundColor!=="rgb(255, 255, 255)"||getComputedStyle(button,"::before").display!=="none")throw new Error("Aceptar secundario neutro, no CTA verde");
    if(innerWidth===390&&parseFloat(style(button).borderRadius)<22)throw new Error("Aceptar redondeado en móvil");
   }
   if(rect(card).x<0||rect(card).right>innerWidth||rect(card).y<0||rect(card).bottom>innerHeight||document.documentElement.scrollWidth>innerWidth)throw new Error("Aviso recortado o desbordado");
   if(innerWidth===390&&rect(button).height<44)throw new Error("Aceptar conserva área táctil");
   return {card:rect(card),circle:rect(circle),check:rect(glyph),button:rect(button),sample,background:tone==="success"?getComputedStyle(circle,"::before").backgroundColor:style(circle).backgroundColor,color:style(glyph).color};
  },{tone,reduced,closing});
  if(name==="success-mobile"){
   for(const [phase,time] of [["large",0],["contract",80],["rebound",260],["settled",520]]){
    await page.evaluate(time=>document.getAnimations().forEach(animation=>{animation.currentTime=time;}),time);
    await page.screenshot({path:resolve(output,`circle-${phase}.png`),fullPage:true});
   }
   await page.evaluate(()=>document.getAnimations().forEach(animation=>{animation.currentTime=1000;}));
  }
  await page.locator(".feedback-accept").focus();
  await page.screenshot({path:resolve(output,`${name}.png`),fullPage:true});
  results.push({name,...result});console.log(`${name}: geometría, color y movimiento correctos`);await context.close();
 }
 writeFileSync(resolve(output,"geometry.json"),JSON.stringify(results,null,2));
 // A natural-speed clip of the same verified CSS sequence, for feel-checking.
 const videoContext=await browser.newContext({viewport:{width:390,height:844},hasTouch:true,reducedMotion:"no-preference",recordVideo:{dir:output,size:{width:390,height:844}}});
 const videoPage=await videoContext.newPage(),video=videoPage.video();
 await videoPage.goto(`http://127.0.0.1:${server.address().port}/?tone=success&paused`);
 await videoPage.evaluate(()=>document.fonts.ready);
 // Start only once recording and fonts are ready, so the clip includes the entrance.
 await videoPage.waitForTimeout(250);
 await videoPage.evaluate(()=>{document.body.classList.remove("feedback-preview-paused");document.getAnimations().forEach(animation=>{animation.currentTime=0;animation.play();});});
 await videoPage.evaluate(()=>Promise.all(document.getAnimations().map(animation=>animation.finished.catch(()=>{}))));
 await videoPage.waitForTimeout(1000);
 await videoContext.close();await video.saveAs(resolve(output,"success-mobile.webm"));
}finally{await browser?.close();await new Promise(resolve=>server.close(resolve));}
