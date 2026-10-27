import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import postcss from "postcss";

const read=path=>readFileSync(new URL(`../${path}`,import.meta.url),"utf8");
const css=read("src/providers/styles/feedback.css"),ast=postcss.parse(css);
const rule=selector=>ast.nodes.find(node=>node.selector===selector);
const values=node=>Object.fromEntries(node.nodes.filter(n=>n.type==="decl").map(n=>[n.prop,n.value]));

test("éxito: círculo verde sólido, icono blanco y tamaño centrado compartido",()=>{
 const icon=values(rule(".feedback-icon.success")),shape=values(rule(".feedback-icon.success:before")),base=values(rule(".feedback-icon"));
 assert.equal(shape.background,"var(--brand-600)");
 assert.equal(icon.color,"var(--surface)");
 assert.equal(icon.border,"0");
 assert.equal(shape["border-radius"],"50%");
 assert.equal(shape["clip-path"],undefined);
 assert.equal(values(rule(".feedback-card.success h2")).color,"var(--ink-950)");
 assert.equal(values(rule(".feedback-card.success h2"))["font-weight"],"var(--weight-bold)");
 assert.equal(values(rule(".feedback-card.success p"))["font-size"],"var(--font-size-base)");
 assert.equal(values(rule(".feedback-card.success"))["padding-top"],"var(--space-16)");
 assert.equal(values(rule(".feedback-card.success .feedback-accept")).background,"var(--surface)");
 assert.equal(values(rule(".feedback-card.success .feedback-accept")).color,"var(--ink-950)");
 assert.equal(values(rule(".feedback-card.success .feedback-accept:before")).display,"none");
 assert.equal(values(rule(".feedback-card")).background,"var(--surface)");
 assert.equal(values(rule(".feedback-card p")).color,"var(--ink-600)");
 assert.doesNotMatch(css,/#[0-9a-f]{3,8}\b|\b(?:rgba?|hsla?)\(/i);
 assert.equal(base.width,"var(--size-76)");
 assert.equal(base.height,"var(--size-76)");
 assert.equal(base["place-items"],"center");
 assert.match(read("src/providers/feedback-provider.tsx"),/item.tone==="success"\?"check":"alert"\} size=\{36\}/);
});

test("el círculo empieza grande y se asienta elásticamente sin deformarse",()=>{
 assert.match(values(rule(".feedback-card.success.in")).animation,/var\(--duration-fast\) var\(--easing-out\) both feedback-success-fade/,"El éxito aparece apenas termina el guardado");
 assert.match(values(rule(".feedback-icon.success:before")).animation,/var\(--duration-dialog\) var\(--easing-spring\) both feedback-success-settle/);
 assert.equal(values(rule(".feedback-icon.success")).animation,undefined);
 assert.match(values(rule(".feedback-icon.success svg")).animation,/feedback-success-fade var\(--duration-fast\) var\(--easing-out\) var\(--duration-press\)/);
 assert.equal(values(rule(".feedback-icon.success svg")).transform,undefined);
 assert.ok(!css.includes("infinite"));
 const settle=ast.nodes.find(node=>node.name==="keyframes"&&node.params==="feedback-success-settle");
 assert.equal(values(settle.nodes.find(node=>node.selector==="from")).transform,"scale(1.3)");
 assert.equal(values(settle.nodes.find(node=>node.selector==="to")).transform,"scale(1)");
 settle.walkDecls(decl=>{
  assert.equal(decl.prop,"transform");
  assert.match(decl.value,/^scale\([\d.]+\)$/);
 });
 assert.match(read("src/providers/feedback-provider.tsx"),/<span key=\{item.id\} className=\{`feedback-icon/);
 assert.ok(!css.includes("stroke-dasharray"));
 assert.ok(!css.includes("ring-pulse-success"));
 for(const name of ["feedback-success-settle","feedback-success-fade"]){
  const animation=ast.nodes.find(node=>node.name==="keyframes"&&node.params===name);
  animation.walkDecls(decl=>assert.ok(["opacity","transform"].includes(decl.prop)));
 }
});

test("movimiento reducido preserva el éxito sin expansión ni rebote",()=>{
 const reduced=ast.nodes.filter(n=>n.name==="media"&&n.params==="(prefers-reduced-motion:reduce)");
 const entrance=reduced.flatMap(node=>node.nodes).find(node=>node.selector===".feedback-card.success.in");
 assert.match(values(entrance).animation,/feedback-success-fade var\(--duration-fast\) var\(--easing-out\)/);
 assert.ok(!values(entrance).animation.includes("spring"));
 assert.ok(reduced[0].nodes.some(node=>node.selector.includes(".feedback-icon svg")&&values(node).animation==="none"));
 assert.ok(reduced[0].nodes.some(node=>node.selector.includes(".feedback-icon:before")&&values(node).animation==="none"));
 assert.ok(!rule(".feedback-card.success :is(h2,p)"));
});

test("no modifica error, información, autocierre ni callback posterior",()=>{
 assert.equal(values(rule(".feedback-icon.danger")).background,"var(--danger-50)");
 assert.equal(values(rule(".feedback-icon.info")).background,"var(--primary-100)");
 const source=read("src/providers/feedback-provider.tsx");
 assert.ok(source.includes('if(item.tone!=="success")return;'));
 assert.ok(source.includes("item.duration??4200"));
 assert.ok(source.includes("setItem(null);setClosing(false);callback?.();"));
 const seen=new Set();
 ast.walkRules(node=>{
  if(node.parent.name==="keyframes")return;
  const context=node.parent.type==="atrule"?node.parent.params:"root";
  for(const selector of node.selectors){const key=`${context}|${selector}`;assert.ok(!seen.has(key),key);seen.add(key);}
 });
});

test("confirmación positiva comparte el verde exacto; la destructiva mantiene su intención",()=>{
 const confirm=postcss.parse(read("src/design-system/styles/confirm-dialog.css"));
 const get=selector=>values(confirm.nodes.find(node=>node.selector===selector));
 assert.equal(get(".confirm-icon.success").background,"var(--brand-600)");
 assert.equal(get(".confirm-icon.success").color,"var(--surface)");
 assert.equal(get(".confirm-icon.danger").background,"var(--danger-50)");
 assert.equal(get(".confirm-icon.danger").color,"var(--danger-600)");
 const source=read("src/design-system/confirm-dialog.tsx");
 assert.match(source,/kind=\{tone==="danger"\?"danger":"primary"\}/,"confirmar es acción pendiente; no botón verde");
 assert.match(source,/className=\{`confirm-icon \$\{tone\}`\} aria-hidden="true"/);
});
