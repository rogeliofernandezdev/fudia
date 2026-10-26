// Local specimen of the real design-system primitives. No API or business writes.
// Run after a build: node tests/helpers/palette-preview.mjs
import {readFileSync,readdirSync} from "node:fs";
import {createRequire} from "node:module";
import {resolve,dirname} from "node:path";
import {fileURLToPath} from "node:url";
import {createServer} from "node:http";
import vm from "node:vm";
import ts from "typescript";
import React from "react";
import {renderToStaticMarkup} from "react-dom/server";
import {resolveTokens} from "./design-tokens.mjs";

const root=resolve(dirname(fileURLToPath(import.meta.url)),"../.."),require=createRequire(import.meta.url),cache=new Map();
function load(file){
 if(cache.has(file))return cache.get(file);
 const exports={};cache.set(file,exports);
 const source=ts.transpileModule(readFileSync(file,"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,esModuleInterop:true}}).outputText;
 vm.runInNewContext(source,{exports,require:name=>{
  if(name.endsWith(".css"))return {};
  if(name.startsWith(".")||name.startsWith("@/"))return load((name.startsWith("@/")?resolve(root,"src",name.slice(2)):resolve(dirname(file),name))+".tsx");
  return require(name);
 }});return exports;
}
const {Button,Status}=load(resolve(root,"src/design-system/page-header.tsx"));
const {Icon}=load(resolve(root,"src/design-system/icons.tsx"));
const h=React.createElement;
const css=["styles/variables.css","styles/globals.css","design-system/styles/table-actions.css","design-system/styles/pagination.css"].map(path=>readFileSync(resolve(root,"src",path),"utf8").replace(/^@import[^;]+;/gm,"")).join("\n");
const fontCss=readdirSync(resolve(root,".next/static/css")).filter(n=>n.endsWith(".css")).map(n=>readFileSync(resolve(root,".next/static/css",n),"utf8")).join("\n");
const font=fontCss.match(/@font-face\{[^}]*font-family:Plus Jakarta Sans;[^}]*unicode-range:u\+00\?\?[^}]*\}/)?.[0].match(/media\/([^/)]+\.woff2)/)?.[1];
if(!font)throw new Error("Genera el build para cargar Plus Jakarta Sans.");
const families=[
 ["Azul operativo","primary",[100,200,600,700],"Acciones y selección"],
 ["Verde esmeralda","brand",[100,200,600,800],"Éxito y disponibilidad"],
 ["Rojo coral","danger",[50,100,500,700],"Acento vivo y texto oscuro para errores"],
 ["Ámbar dorado","warning",[50,100,500,800],"Acento vivo y texto oscuro para advertencias"],
];
const specimen=renderToStaticMarkup(h("main",null,
 h("header",null,h("h1",null,"Paleta de FudIA"),h("p",null,"Tonos vivos, fondos suaves y texto con contraste.")),
 h("div",{className:"palette-families"},families.map(([name,family,steps,role])=>h("section",{key:family},h("h2",null,name),h("p",null,role),h("div",{className:"swatches"},steps.map(step=>h("div",{key:step},h("div",{className:"swatch",style:{background:`var(--${family}-${step})`}}),h("b",null,`${family}-${step}`),h("small",null,resolveTokens(`var(--${family}-${step})`).toUpperCase()))))))),
 h("section",{className:"palette-states"},h("h2",null,"Estados compartidos"),h("div",{className:"samples"},h(Status,{tone:"green",icon:"circleCheck"},"Disponible"),h(Status,{tone:"red",icon:"cancel"},"Cancelado"),h(Status,{tone:"orange",icon:"clock"},"Pendiente"),h(Status,{tone:"blue",icon:"orders"},"Confirmado"))),
 h("section",null,h("h2",null,"Acciones y superficies"),h("div",{className:"samples"},h(Button,{icon:"check"},"Guardar cambios"),h(Button,{kind:"danger",icon:"cancel"},"Confirmar anulación"),h("span",{className:"palette-ready"},h(Icon,{name:"circleCheck",size:20}),"Por entregar"))),
 h("footer",null,"Muestra de componentes reales. Sin datos ni operaciones del restaurante.")
));
const layout=`@font-face{font-family:Plus Jakarta Sans;font-weight:200 800;src:url(/font.woff2)}:root{--font-jakarta:"Plus Jakarta Sans"}body{background:var(--cloud-50)}main{max-width:var(--size-1120);margin:auto;padding:var(--space-8)}header{margin-bottom:var(--space-8)}h1{font-size:var(--font-size-2xl);margin:0}h2{font-size:var(--font-size-md);margin:0 0 var(--space-2)}p,footer{color:var(--ink-500);margin:var(--space-2) 0 0}section{margin-bottom:var(--space-6)}.palette-families{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:var(--space-6)}.swatches{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:var(--space-2);margin-top:var(--space-3)}.swatch{height:var(--size-56);border-radius:var(--radius-control)}.swatches b,.swatches small{display:block;font-size:var(--font-size-xs);margin-top:var(--space-1)}.swatches small{color:var(--ink-500)}.samples{display:flex;flex-wrap:wrap;gap:var(--space-3);align-items:center}.palette-ready{display:flex;align-items:center;gap:var(--space-2);padding:var(--space-3);border-radius:var(--radius-control);background:var(--brand-600);color:var(--ink-800);font-weight:var(--weight-semibold)}footer{border-top:var(--stroke-1) solid var(--line);padding-top:var(--space-4);font-size:var(--font-size-xs)}@media(width<=600px){main{padding:var(--space-5)}.palette-families{grid-template-columns:1fr}.samples>.button{width:100%}}`;
createServer((req,res)=>{
 if(req.url==="/font.woff2"){res.setHeader("content-type","font/woff2");res.end(readFileSync(resolve(root,".next/static/media",font)));return;}
 res.setHeader("content-type","text/html; charset=utf-8");res.end(`<!doctype html><html lang="es"><head><meta name="viewport" content="width=device-width,initial-scale=1"><title>Paleta de FudIA</title><style>${css}\n${layout}</style></head><body>${specimen}</body></html>`);
}).listen(5196,"localhost",()=>console.log("Palette QA: http://localhost:5196"));
