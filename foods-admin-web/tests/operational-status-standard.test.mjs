import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import test from "node:test";
import postcss from "postcss";
import {resolveTokens,tokens} from "./helpers/design-tokens.mjs";

const read=path=>readFileSync(new URL(path,import.meta.url),"utf8");
const css=postcss.parse(read("../src/styles/globals.css"));
const rules=new Map();
css.walkRules(rule=>{
 const declarations=new Map();
 rule.each(node=>{if(node.type==="decl")declarations.set(node.prop,node.value);});
 rules.set(rule.selector,declarations);
});
function luminance(hex){
 const values=hex.slice(1).match(/../g).map(channel=>parseInt(channel,16)/255)
  .map(value=>value<=.04045?value/12.92:((value+.055)/1.055)**2.4);
 return values[0]*.2126+values[1]*.7152+values[2]*.0722;
}
// Matriz FUDIA DS 1.0 (aprobada 2026-11-24): estados en superficies pastel con texto
// oscuro e indicador de su familia. No existe la variante sólida; los atributos usan `Tag`.
const soft=[
 ["gray","cloud-100","#f4f5f7","ink-600","#4b5563"],
 ["blue","info-bg","#eff6ff","info-fg","#1d4ed8"],
 ["sky","sky-100","#e0f2fe","sky-800","#075985"],
 ["violet","violet-bg","#f5f3ff","violet-fg","#6d28d9"],
 ["green","success-bg","#ecfdf5","success-fg","#047857"],
 ["orange","warning-bg","#fffbeb","warning-fg","#92400e"],
 ["teal","teal-bg","#f0fdfa","teal-fg","#0f766e"],
 ["red","danger-bg","#fef2f2","danger-fg","#b91c1c"],
 ["gray emphasis","cloud-150","#e3e5e9","ink-800","#1f2937"],
];
for(const [tone,bgToken,bg,fgToken,fg] of soft){
 test(`Status suave ${tone}: superficie pastel, texto oscuro y contraste 4.5:1`,()=>{
  const selector=tone==="green"?".status":tone==="gray emphasis"?".status.gray.emphasized":`.status.${tone}`;
  const rule=rules.get(selector);
  assert.equal(rule.get("background"),`var(--${bgToken})`,`${tone} usa el alias semántico de fondo`);
  assert.equal(rule.get("color"),`var(--${fgToken})`,`${tone} usa el alias semántico de texto`);
  assert.equal(resolveTokens(rule.get("background")),bg);
  assert.equal(resolveTokens(rule.get("color")),fg);
  assert.ok((luminance(bg)+.05)/(luminance(fg)+.05)>=4.5);
 });
}
test("no existe la variante sólida; el indicador acompaña al texto y los atributos usan la etiqueta neutra",()=>{
 for(const selector of rules.keys())assert.doesNotMatch(selector,/\.status\.solid|\.status\.(purple|navy)/,`Regla legacy sin dueño: ${selector}`);
 assert.equal(rules.get(".tag").get("background"),"var(--cloud-100)");
 assert.equal(rules.get(".tag").get("color"),"var(--ink-700)");
 const source=read("../src/design-system/page-header.tsx");
 assert.doesNotMatch(source,/solid/,"Status ya no expone variante sólida: todos los estados son pastel con etiqueta");
 for(const file of ["../src/modules/operations/orders/presentation/orders-manager.tsx","../src/modules/operations/orders/presentation/order-detail-dialog.tsx","../src/modules/operations/salon/presentation/salon-manager.tsx","../src/modules/delivery-panel/presentation/courier-panel.tsx"]){
  assert.doesNotMatch(read(file),/<Status [^>]*solid/,`${file} no marca estados como sólidos`);
 }
 assert.doesNotMatch(read("../src/modules/operations/pos/presentation/pos-page.tsx"),/<Status [^>]*solid/,"Cobros muestra solo el estado de cobro; la preparación vive en Pedidos");
 assert.doesNotMatch(read("../src/modules/menu/products/presentation/catalog-manager.tsx"),/<Status tone=\{p\.quantityControl/,"el control de cantidad es atributo, no estado");
});
test("la documentación canónica conserva la paleta exacta del componente",()=>{
 const standard=read("../docs/02_DESIGN_SYSTEM.md");
 for(const [,bgToken,bg] of soft)assert.ok(standard.includes(`\`${bgToken}\``)||standard.includes(bg.toUpperCase()),bgToken);
 assert.match(standard,/\| Cuenta \| Cuenta abierta \| `teal`/);
 assert.match(standard,/\| Pedido \| Confirmado \| `blue`/);
 assert.notEqual(tokens.get("--teal-500"),tokens.get("--info-500"),"La cuenta abierta (petróleo) no comparte tono con Confirmado (información)");
});
