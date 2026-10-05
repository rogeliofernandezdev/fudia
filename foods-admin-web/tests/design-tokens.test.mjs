import assert from "node:assert/strict";
import {readFileSync,readdirSync} from "node:fs";
import test from "node:test";
import postcss from "postcss";
import {tokens,tokenAst,resolveTokens} from "./helpers/design-tokens.mjs";

const root=new URL("../",import.meta.url);
const read=file=>readFileSync(new URL(file,root),"utf8");
function files(dir){return readdirSync(new URL(dir+"/",root),{withFileTypes:true}).flatMap(entry=>entry.isDirectory()?files(`${dir}/${entry.name}`):[`${dir}/${entry.name}`]);}
const sourceFiles=files("src");

test("variables.css es la única autoridad de tokens y se importa una vez desde la base",()=>{
  const imports=[];
  for(const file of sourceFiles.filter(file=>/\.(css|tsx|ts)$/.test(file))){
    const source=read(file);
    if(source.includes('"./variables.css"'))imports.push(file);
    if(!file.endsWith(".css")||file==="src/styles/variables.css")continue;
    postcss.parse(source).walkDecls(d=>assert.equal(d.prop.startsWith("--"),false,`${file}: ${d.prop}`));
  }
  assert.deepEqual(imports,["src/styles/globals.css"]);
  const globalAst=postcss.parse(read(imports[0]));
  assert.deepEqual(globalAst.nodes.filter(n=>n.type==="atrule"&&n.name==="import").map(n=>n.params),['"tailwindcss"','"./variables.css"']);
});

test("los estilos consumen la paleta y escalas sin volver a escribir valores de diseño",()=>{
  for(const file of sourceFiles.filter(file=>file.endsWith(".css")&&file!=="src/styles/variables.css")){
    postcss.parse(read(file)).walkDecls(d=>{
      if(d.value.includes("url("))return; // SVG masks have their own document context.
      assert.equal(/#[\da-f]{3,8}\b|rgba?\(/i.test(d.value),false,`${file}: ${d.toString()}`);
      assert.equal(/-?\d*\.?\d+px\b/.test(d.value),false,`${file}: ${d.toString()}`);
      if(d.prop==="font-weight")assert.equal(/^\d+$/.test(d.value),false,`${file}: ${d.toString()}`);
    });
  }
});

test("todos los tokens usados existen y sus dependencias no forman ciclos",()=>{
  const names=new Set();tokenAst.walkDecls(d=>names.add(d.prop));
  const external=new Set(["--font-manrope","--font-geist-mono"]);
  for(const file of sourceFiles.filter(file=>/\.(css|tsx|ts)$/.test(file))){
    for(const match of read(file).matchAll(/(?:var\(|readCssToken\(")(--[\w-]+)/g)){
      assert.ok(names.has(match[1])||external.has(match[1]),`${file}: ${match[1]}`);
    }
  }
  tokenAst.walkDecls(d=>assert.doesNotThrow(()=>resolveTokens(d.value)));
  const rawColors=[...tokens.values()].filter(v=>/^#[\da-f]+$/i.test(v));
  assert.equal(new Set(rawColors).size,rawColors.length,"Un mismo color debe tener una sola definición y alias semánticos");
});

test("editar los tokens base actualiza acciones, alias y transparencias compartidas",()=>{
  const changed=new Map(tokens);
  changed.set("--primary-600","#123456");changed.set("--brand-700","#234567");changed.set("--danger-600","#345678");changed.set("--space-12","18px");
  assert.equal(resolveTokens("var(--action-pay)",changed),"#123456");
  assert.equal(resolveTokens("var(--ops-500)",changed),"#123456");
  assert.equal(resolveTokens("var(--success-600)",changed),"#234567");
  assert.equal(resolveTokens("var(--action-cancel)",changed),"#345678");
  assert.match(resolveTokens("var(--primary-600-a46)",changed),/#123456/);
  const dialog=postcss.parse(read("src/design-system/styles/confirm-dialog.css"));
  let headerGap;dialog.walkDecls("gap",d=>{if(d.parent.selector===".confirm-header")headerGap=d.value;});
  assert.equal(resolveTokens(headerGap,changed),"18px");
});

test("los controles conservan la altura táctil y el modal conserva Scale Punch elástico",()=>{
  assert.equal(resolveTokens("var(--control-height)"),"38px");
  const mobileRoot=tokenAst.nodes.find(n=>n.type==="atrule"&&n.params==="(width<=600px)"&&n.nodes.some(r=>r.selectors?.includes(":root")));
  assert.ok(mobileRoot);
  const control=mobileRoot.nodes.find(n=>n.selectors?.includes(":root")).nodes.find(n=>n.prop==="--control-height");
  assert.equal(resolveTokens(control.value),"44px");
  assert.equal(resolveTokens("var(--motion-dialog-duration)"),"520ms");
  assert.equal(resolveTokens("var(--motion-dialog-easing)"),"cubic-bezier(.22,1,.36,1)");
  const css=read("src/design-system/styles/confirm-dialog.css");
  assert.match(css,/@media\(prefers-reduced-motion:no-preference\)/);
  assert.match(css,/42%\{opacity:1;transform:scale\(1\.06\)\}/);
});
