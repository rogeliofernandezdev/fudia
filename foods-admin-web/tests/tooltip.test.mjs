import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import postcss from "postcss";
import React from "react";
import {renderToStaticMarkup} from "react-dom/server";
import {resolveTokens} from "./helpers/design-tokens.mjs";

const require=createRequire(import.meta.url);
const read=file=>readFileSync(new URL(`../${file}`,import.meta.url),"utf8");
const load=(file,resolve)=>{const exports={};vm.runInNewContext(ts.transpileModule(read(file),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,esModuleInterop:true}}).outputText,{exports,require:resolve,URLSearchParams});return exports;};
// Los objetos creados en el contexto de vm se comparan como datos planos.
const plain=value=>JSON.parse(JSON.stringify(value));
const {Tooltip,placeTooltip}=load("src/design-system/tooltip.tsx",name=>name.endsWith(".css")?{}:require(name));
const viewport={width:390,height:844};
const bubble={width:280,height:96};

test("la burbuja abre sobre la imagen y baja si no cabe arriba",()=>{
 assert.deepEqual(plain(placeTooltip({left:28,top:400,bottom:480},bubble,viewport)),{placement:"top",left:28,top:296});
 assert.deepEqual(plain(placeTooltip({left:28,top:60,bottom:140},bubble,viewport)),{placement:"bottom",left:28,top:148});
});

test("la burbuja nunca se sale del viewport de 390 px ni del de referencia",()=>{
 for(const size of [viewport,{width:427,height:933}]){
  const right=placeTooltip({left:size.width-40,top:500,bottom:580},bubble,size);
  assert.ok(right.left+bubble.width<=size.width-16,"acotada al borde derecho");
  const left=placeTooltip({left:-20,top:500,bottom:580},bubble,size);
  assert.equal(left.left,16,"acotada al borde izquierdo");
  const tall=placeTooltip({left:28,top:40,bottom:size.height-40},{width:280,height:size.height},size);
  assert.equal(tall.top,16,"nunca arranca fuera del borde superior");
 }
});

test("el disparador es un botón accesible y no muestra la burbuja hasta interactuar",()=>{
 const markup=renderToStaticMarkup(React.createElement(Tooltip,{label:"Ver descripción y alérgenos de Ají de pollo",content:"Pollo en crema de ají.",className:"cmd-item-visual"},React.createElement("img",{alt:""})));
 assert.match(markup,/^<button type="button" class="ds-tooltip-trigger cmd-item-visual" aria-label="Ver descripción y alérgenos de Ají de pollo">/);
 assert.doesNotMatch(markup,/role="tooltip"/,"cerrado no deja contenido vacío ni visible");
 const source=read("src/design-system/tooltip.tsx");
 for(const contract of ['event.pointerType==="mouse"',':focus-visible','onContextMenu','event.key==="Escape"','"pointerdown"','aria-describedby={open?id:undefined}','role="tooltip"','createPortal'])assert.ok(source.includes(contract),contract);
});

test("estilos del tooltip: por encima de modales, sin hover fuera de puntero fino y con movimiento reducido",()=>{
 const css=read("src/design-system/styles/tooltip.css");
 assert.ok(Number(resolveTokens("var(--z-tooltip)"))>Number(resolveTokens("var(--z-sheet)")),"visible sobre la comanda en hoja o pantalla completa");
 assert.ok(Number(resolveTokens("var(--z-tooltip)"))>Number(resolveTokens("var(--z-feedback)")),"también por delante de los diálogos de feedback");
 assert.doesNotMatch(css,/:hover/);
 assert.match(css,/@media \(prefers-reduced-motion:reduce\)\{\.ds-tooltip\[data-ready\]\{animation:none\}\}/);
 assert.match(css,/:where\(\.ds-tooltip-trigger\)/,"el reset no pisa la clase visual del consumidor");
 const rule=postcss.parse(css).nodes.find(node=>node.selector===".ds-tooltip");
 const maxWidth=rule.nodes.find(node=>node.prop==="max-width").value;
 assert.match(maxWidth,/calc\(100vw - var\(--space-8\)\)/,"deja 16 px por lado en móvil");
 assert.match(read("src/app/layout.tsx"),/@\/design-system\/styles\/tooltip\.css/);
 assert.match(read("src/design-system/index.ts"),/export \{Tooltip\} from "\.\/tooltip"/);
});

test("las etiquetas de acciones y ayudas usan una capa global fuera del contenedor que recorta",()=>{
 const {TooltipLayer}=load("src/design-system/tooltip-layer.tsx",name=>name==="./tooltip"?{placeTooltip}:require(name));
 assert.equal(renderToStaticMarkup(React.createElement(TooltipLayer)),"","cerrado no deja burbujas ni cambia el HTML del servidor");
 const source=read("src/design-system/tooltip-layer.tsx");
 for(const contract of ['createPortal','document.body','closest<HTMLElement>("[data-tooltip]")','"pointerover"','"focusin"','"Escape"','"aria-describedby"','MutationObserver','"resize"','"scroll"'])assert.ok(source.includes(contract),contract);
 assert.match(read("src/providers/providers.tsx"),/<TooltipLayer\/>/,"todas las pantallas comparten una capa");
 assert.doesNotMatch(read("src/design-system/styles/table-actions.css"),/\[data-tooltip\].*?:after/,"las acciones ya no generan burbujas recortadas por overflow");
});

test("la carta recibe descripción y alérgenos del API sin catálogos locales",async()=>{
 const calls=[];
 const api=load("src/modules/operations/salon/infrastructure/catalog-api.ts",name=>name==="@/shared/api/client"?{apiFetch:async path=>{calls.push(path);return{total:2,items:[
  {productId:"a",name:"Ají de pollo",description:" Pollo en crema de ají. ",allergens:["Leche"],price:"22.00",categoryId:null,imageUrl:null,status:"available",remaining:null},
  {productId:"b",name:"Chicha morada",description:"",allergens:[],price:"8.00",categoryId:null,imageUrl:null,status:"available",remaining:null},
 ]}}}:require(name));
 const page=await api.listSalonProducts({categoryId:"",q:"",page:1,pageSize:24});
 assert.match(calls[0],/^product-availability\?/);
 assert.deepEqual(plain(page.items.map(item=>[item.description,item.allergens])),[["Pollo en crema de ají.",["Leche"]],["",[]]]);
});
