import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync,readdirSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import postcss from "postcss";
import ts from "typescript";
import React from "react";
import {renderToStaticMarkup} from "react-dom/server";
import {tokens,resolveTokens} from "./helpers/design-tokens.mjs";

const root=new URL("../",import.meta.url);
const read=file=>readFileSync(new URL(file,root),"utf8");
const files=(dir,ext)=>readdirSync(new URL(dir+"/",root),{withFileTypes:true}).flatMap(entry=>entry.isDirectory()?files(`${dir}/${entry.name}`,ext):entry.name.endsWith(ext)?[`${dir}/${entry.name}`]:[]);
const sheets=files("src",".css").map(file=>({file,ast:postcss.parse(read(file))}));
const insideHoverMedia=node=>{for(let parent=node.parent;parent;parent=parent.parent)if(parent.type==="atrule"&&/hover:\s*hover/.test(parent.params))return true;return false;};

const require=createRequire(import.meta.url);
const exports={};
vm.runInNewContext(ts.transpileModule(read("src/design-system/page-header.tsx"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,esModuleInterop:true}}).outputText,{exports,require:name=>name.endsWith(".css")?{}:name==="next/link"?{}:name==="@/design-system/icons"?{Icon:({name})=>React.createElement("svg",{"data-icon":name})}:require(name)});
const {Button}=exports;
const render=props=>renderToStaticMarkup(React.createElement(Button,props,"Guardar"));

test("tokens de microinteracción: una sola respuesta de superficie, presión y opacidad",()=>{
 for(const name of ["--transition-surface","--transition-press","--press-scale","--press-scale-icon","--opacity-disabled","--opacity-busy"])assert.ok(tokens.has(name),name);
 const surface=resolveTokens("var(--transition-surface)");
 for(const property of ["background-color","border-color","color","box-shadow","opacity"])assert.match(surface,new RegExp(`\\b${property} 160ms cubic-bezier`),property);
 assert.doesNotMatch(surface,/\btransform\b|\ball\b/,"La superficie no anima geometría");
 assert.equal(resolveTokens("var(--transition-press)"),"transform 120ms cubic-bezier(.23,1,.32,1)");
 assert.ok(Number(resolveTokens("var(--opacity-busy)"))>Number(resolveTokens("var(--opacity-disabled)")),"En curso se distingue de no disponible");
});

test("hover solo con puntero fino: en táctil no queda pegado tras el toque",()=>{
 const loose=[];
 for(const {file,ast} of sheets)ast.walkRules(rule=>{if(rule.parent.type!=="rule"&&rule.selectors.some(s=>/:hover\b/.test(s))&&!insideHoverMedia(rule))loose.push(`${file}: ${rule.selector}`);});
 assert.deepEqual(loose,[]);
});

test("transiciones explícitas con la curva del sistema: sin all, sin ease genérico",()=>{
 const issues=[];
 for(const {file,ast} of sheets)ast.walkDecls(/^(transition|animation)$/,decl=>{
  if(decl.prop==="transition"&&/(^|,)\s*all\b/.test(decl.value))issues.push(`${file}: ${decl.parent.selector} transition all`);
  if(/(^|[\s,])(ease|ease-in|ease-out)(?=[\s,]|$)/.test(decl.value))issues.push(`${file}: ${decl.parent.selector} ${decl.value}`);
 });
 assert.deepEqual(issues,[]);
});

test("presionar responde con la escala del sistema en botones e iconos",()=>{
 const globals=read("src/styles/globals.css");
 assert.ok(globals.includes(".button:active:not(:disabled){transform:scale(var(--press-scale))}"));
 assert.ok(globals.includes(".icon-button:active{transform:scale(var(--press-scale-icon))}"));
 const literals=[];
 for(const {file,ast} of sheets)ast.walkRules(/:active/,rule=>rule.walkDecls("transform",decl=>{if(/scale\(\.\d+\)/.test(decl.value))literals.push(`${file}: ${rule.selector}`);}));
 assert.deepEqual(literals,[],"La escala de presión sale de --press-scale*");
});

test("loading: indicador en el lugar del icono, bloqueo y anuncio de petición en curso",()=>{
 const busy=render({icon:"check",loading:true});
 assert.match(busy,/aria-busy="true"/);
 assert.match(busy,/disabled=""/);
 assert.match(busy,/<svg class="ds-spinner"[^>]*aria-hidden="true"><circle class="ds-spinner-track"[^>]*><\/circle><path class="ds-spinner-arc"[^>]*><\/path><\/svg><span>Guardar<\/span>/);
 assert.doesNotMatch(busy,/data-icon="check"/);
 const trailing=render({action:"continue",loading:true});
 assert.ok(trailing.indexOf("<span>Guardar</span>")<trailing.indexOf("ds-spinner"),"Continuar conserva la posición de su flecha");
 assert.match(render({"aria-busy":true}),/ds-spinner/,"aria-busy explícito también muestra el indicador");
 const idle=render({icon:"check"});
 assert.doesNotMatch(idle,/ds-spinner|aria-busy|disabled/);
 const globals=read("src/styles/globals.css");
 assert.match(globals,/\.button\[aria-busy=true\]\{cursor:progress;opacity:1\}/,"Con indicador conserva el color completo");
 assert.match(globals,/\.ds-spinner\{display:block;flex:none;animation:spinner-in var\(--duration-fast\) var\(--easing-out\),spin /);
 assert.match(globals,/\.ds-spinner :is\(circle,path\)\{fill:none;stroke:currentColor;stroke-width:2\.5;stroke-linecap:round\}/);
});

test("todo botón que cambia a gerundio declara loading o aria-busy",()=>{
 const missing=[];
 for(const file of files("src",".tsx")){
  for(const match of read(file).matchAll(/<Button((?:[^>]|=>)*?)>\{[\w.!&|() =]+?\?"[^"]*…"/g)){
   if(!/\bloading=|aria-busy=/.test(match[1]))missing.push(`${file}: ${match[0].slice(0,80)}`);
  }
 }
 assert.deepEqual(missing,[]);
});

test("error, foco y deshabilitado ganan al hover de los campos",()=>{
 const globals=read("src/styles/globals.css");
 for(const field of [".ds-input",".ds-select",".ds-textarea"])assert.ok(globals.includes(`${field}:hover:not(:disabled,:focus,[aria-invalid=true])`),field);
 assert.ok(globals.includes(":where(input:is([type=checkbox],[type=radio],[type=range],[type=file])):focus-visible{outline:var(--stroke-2) solid var(--primary-600)"));
 assert.match(globals,/\.ds-icon-button:disabled\{[^}]*opacity:var\(--opacity-disabled\)/);
 assert.match(globals,/\.button:disabled\{[^}]*opacity:var\(--opacity-disabled\)/);
 assert.match(read("src/design-system/styles/form-field.css"),/\.field-error\{animation:feedback-reveal var\(--duration-fast\) var\(--easing-out\)/);
 assert.match(globals,/@keyframes feedback-reveal\{from\{opacity:0;transform:translateY/);
});

test("guardado en curso sobre el formulario: velo sin texto y spinner solo en esperas largas",()=>{
 const modal=read("src/design-system/modal.tsx");
 assert.match(modal,/export function ModalBusy\(\{className=""\}/);
 assert.ok(modal.includes('<div className={`modal-busy ${className}`.trim()} aria-hidden="true"><span className="modal-busy-indicator"><Spinner size={20}/></span></div>'),"Sin texto: el botón es el único que dice «Guardando…»");
 const css=read("src/design-system/styles/modal.css");
 const busy=css.match(/\.modal-busy\{([^}]*)\}/)[1];
 assert.match(busy,/background:var\(--surface-alpha-64\)/,"Velo neutro: el verde se reserva para éxito confirmado");
 assert.doesNotMatch(busy,/backdrop-filter|brand/,"Sin blur ni verde");
 assert.match(busy,/animation:modal-busy-in var\(--duration-base\) var\(--easing-out\) var\(--duration-fast\) both/,"Bloquea al instante y atenúa tras 160 ms");
 assert.match(css,/\.modal-busy-indicator\{[^}]*animation:modal-busy-indicator-in var\(--duration-base\) var\(--easing-out\) var\(--delay-long-wait\) both\}/,"El spinner central solo aparece si la espera supera 1 s");
 assert.equal(resolveTokens("var(--delay-long-wait)"),"1000ms");
 assert.doesNotMatch(css,/modal-busy-chip/);
 const handmade=files("src",".tsx").filter(file=>/className="modal-busy"/.test(read(file))&&file!=="src/design-system/modal.tsx");
 assert.deepEqual(handmade,[],"Los formularios usan ModalBusy, no marcado propio");
});

test("todo modal que guarda muestra ModalBusy desde la cabecera compartida",()=>{
 const modal=read("src/design-system/modal.tsx");
 assert.match(modal,/busy=false,busyOverlay=true/);
 assert.doesNotMatch(modal,/busyLabel/,"El velo no repite el texto del botón");
 assert.ok(modal.includes("{busy&&busyOverlay&&<ModalBusy/>}"),"ModalHeader cubre el cuerpo mientras guarda");
 const purchases=read("src/modules/supply/purchases/presentation/purchases-page.tsx");
 for(const [pending,text] of [["approve","Aprobando…"],["pending_approval","Enviando…"],["draft","Volviendo a borrador…"]])assert.ok(purchases.includes(`loading={pending==="${pending}"}>{pending==="${pending}"?"${text}"`),text);
 const manual=files("src/modules",".tsx").filter(file=>read(file).includes("<ModalBusy"));
 assert.deepEqual(manual,[],"Los módulos no repiten el bloqueo: lo aporta ModalHeader");
 assert.match(read("src/modules/operations/orders/presentation/comanda-view.tsx"),/busy=\{busy\} busyOverlay=\{false\}/,"La Comanda es pantalla de tarea: solo el botón muestra el estado");
 for(const [file,title] of [["customers/presentation/customers-manager.tsx","Información del cliente"],["identity/presentation/users-roles-manager.tsx","Datos y accesos"],["configuration/presentation/payment-methods-page.tsx","Configuración del medio de pago"],["organizations/presentation/organization-admin.tsx","Datos del local"],["organizations/presentation/organization-admin.tsx","Jurisdicción fiscal"],["organizations/presentation/organization-admin.tsx","Tipo de cambio"]]){
  const source=read(`src/modules/${file}`),start=source.indexOf(`<ModalHeader title="${title}"`);
  assert.ok(start>=0,title);
  assert.match(source.slice(start,source.indexOf("/>",start)),/busy=\{busy\}/,title);
 }
});

test("el éxito se notifica apenas termina el guardado, sin esperar recargas",()=>{
 const late=[];
 for(const file of files("src",".tsx")){
  const source=read(file);
  for(const match of source.matchAll(/notify\(\{\s*tone:"success"/g)){
   const before=source.slice(Math.max(0,match.index-700),match.index);
   const start=Math.max(before.lastIndexOf("onSuccess"),before.lastIndexOf("async function"));
   if(start>=0&&/\bawait\b/.test(before.slice(start)))late.push(`${file}:${source.slice(0,match.index).split("\n").length}`);
  }
 }
 assert.deepEqual(late,[],"Las recargas posteriores corren en segundo plano después de notify");
});
