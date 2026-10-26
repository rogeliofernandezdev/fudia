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
const palette=[
 ["gray","cloud-100","#f0f2f8","ink-500","#59617c"],
 ["blue","indigo-100","#eef0ff","indigo-700","#3946b8"],
 ["sky","sky-100","#e0f2fe","sky-800","#075985"],
 ["violet","digital-100","#f2ecff","digital-700","#5421a8"],
 ["green","brand-100","#e2f8ec","brand-800","#06613a"],
 ["orange","warning-100","#fff1d2","warning-800","#784400"],
 ["teal","teal-100","#e6f5f7","teal-700","#0b6b78"],
 ["red","danger-50","#fff1f2","danger-700","#b51f33"],
 ["gray emphasis","cloud-150","#e9ecf4","ink-800","#222842"],
];
function luminance(hex){
 const values=hex.slice(1).match(/../g).map(channel=>parseInt(channel,16)/255)
  .map(value=>value<=.04045?value/12.92:((value+.055)/1.055)**2.4);
 return values[0]*.2126+values[1]*.7152+values[2]*.0722;
}
for(const [tone,bgToken,bg,fgToken,fg] of palette){
 test(`Status ${tone}: tokens aprobados, fondo tenue y contraste mínimo 4.5:1`,()=>{
  const selector=tone==="green"?".status":tone==="gray emphasis"?".status.gray.emphasized":`.status.${tone}`;
  const rule=rules.get(selector);
  assert.equal(rule.get("background"),`var(--${bgToken})`);
  assert.equal(rule.get("color"),`var(--${fgToken})`);
  assert.equal(resolveTokens(rule.get("background")),bg);
  assert.equal(resolveTokens(rule.get("color")),fg);
  assert.ok((luminance(bg)+.05)/(luminance(fg)+.05)>=4.5);
 });
}
test("la documentación canónica conserva la paleta exacta del componente",()=>{
 const standard=read("../docs/02_DESIGN_SYSTEM.md");
 for(const [,bgToken,bg,fgToken,fg] of palette){
  assert.ok(standard.includes(`\`${bgToken}\` · \`${bg.toUpperCase()}\``));
  assert.ok(standard.includes(`\`${fgToken}\` · \`${fg.toUpperCase()}\``));
 }
 assert.match(standard,/\| Cuenta \| Cuenta abierta \| `sky`/);
 assert.match(standard,/\| Pedido \| Confirmado \| `blue`/);
 assert.notEqual(tokens.get("--sky-100"),tokens.get("--primary-100"));
});
