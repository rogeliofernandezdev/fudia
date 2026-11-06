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
// Niveles de estado (2026-11-05): suave por defecto (registros, compras, caja, cuenta y pago)
// y sólido solo para los estados operativos del pedido.
const soft=[
 ["gray","cloud-100","#f0f2f8","ink-500","#59617c"],
 ["blue","indigo-100","#eef0ff","indigo-700","#3946b8"],
 ["sky","sky-100","#e0f2fe","sky-800","#075985"],
 ["violet","digital-100","#f2ecff","digital-700","#5421a8"],
 ["purple","purple-100","#f2e6f8","purple-600","#681a92"],
 ["green","brand-100","#e3f8e9","brand-800","#055e2b"],
 ["orange","warning-100","#fff0d8","warning-800","#744000"],
 ["teal","teal-100","#e6f5f7","teal-700","#0b6b78"],
 ["red","danger-50","#fff2ef","danger-700","#9f2115"],
 ["gray emphasis","cloud-150","#e9ecf4","ink-800","#222842"],
];
const solid=[
 ["gray","ink-500","#59617c"],["blue","indigo-600","#4654cd"],["sky","sky-500","#0ea5e9"],
 ["violet","digital-500","#7c3aed"],["purple","purple-600","#681a92"],["green","brand-600","#03c056"],
 ["orange","warning-500","#fd9f0a"],["teal","teal-500","#0e9aab"],["red","danger-500","#fd3622"],
 ["navy","navigation-900","#052352"],
];
for(const [tone,bgToken,bg,fgToken,fg] of soft){
 test(`Status suave ${tone}: fondo tenue, texto oscuro y contraste 4.5:1`,()=>{
  const selector=tone==="green"?".status":tone==="gray emphasis"?".status.gray.emphasized":`.status.${tone}`;
  const rule=rules.get(selector);
  assert.equal(rule.get("background"),`var(--${bgToken})`);
  assert.equal(rule.get("color"),`var(--${fgToken})`);
  assert.equal(resolveTokens(rule.get("background")),bg);
  assert.equal(resolveTokens(rule.get("color")),fg);
  assert.ok((luminance(bg)+.05)/(luminance(fg)+.05)>=4.5);
 });
}
for(const [tone,bgToken,bg] of solid){
 test(`Status sólido ${tone}: relleno vivo con texto blanco`,()=>{
  const rule=rules.get(`.status.solid.${tone}`);
  assert.equal(rule.get("background"),`var(--${bgToken})`);
  assert.equal(resolveTokens(rule.get("background")),bg);
 });
}
test("solo la variante sólida usa texto blanco; los atributos usan la etiqueta neutra",()=>{
 assert.equal(rules.get(".status.solid").get("color"),"var(--surface)");
 assert.equal(rules.get(".status.solid.gray.emphasized").get("color"),"var(--surface)","el gris enfatizado sólido (Salón «Nuevo») no hereda tinta oscura");
 assert.equal(rules.get(".status.solid i,.status.solid .status-icon").get("color"),"var(--surface)");
 assert.equal(rules.get(".tag").get("background"),"var(--cloud-100)");
 assert.equal(rules.get(".tag").get("color"),"var(--ink-700)");
 const source=read("../src/design-system/page-header.tsx");
 assert.match(source,/solid=false/,"Status es suave salvo que la pantalla pida `solid`");
 const solidUsers=["../src/modules/operations/orders/presentation/orders-manager.tsx","../src/modules/operations/orders/presentation/order-detail-dialog.tsx","../src/modules/operations/salon/presentation/salon-manager.tsx","../src/modules/operations/pos/presentation/pos-page.tsx"];
 for(const file of solidUsers)assert.match(read(file),/<Status [^>]*solid/,`${file} marca el estado operativo como sólido`);
 assert.doesNotMatch(read("../src/modules/menu/products/presentation/catalog-manager.tsx"),/<Status tone=\{p\.quantityControl/,"el control de cantidad es atributo, no estado");
});
test("la documentación canónica conserva la paleta exacta del componente",()=>{
 const standard=read("../docs/02_DESIGN_SYSTEM.md");
 for(const [,bgToken,bg] of solid)assert.ok(standard.includes(`\`${bgToken}\` · \`${bg.toUpperCase()}\``),bgToken);
 assert.match(standard,/\| Cuenta \| Cuenta abierta \| `sky`/);
 assert.match(standard,/\| Pedido \| Confirmado \| `blue`/);
 assert.notEqual(tokens.get("--sky-500"),tokens.get("--indigo-600"));
});
