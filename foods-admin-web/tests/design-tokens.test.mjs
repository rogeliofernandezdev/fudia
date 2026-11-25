import assert from "node:assert/strict";
import {readFileSync,readdirSync} from "node:fs";
import test from "node:test";
import postcss from "postcss";
import {tokens,tokenAst,resolveTokens} from "./helpers/design-tokens.mjs";

const root=new URL("../",import.meta.url);
const read=file=>readFileSync(new URL(file,root),"utf8");
function files(dir){return readdirSync(new URL(dir+"/",root),{withFileTypes:true}).flatMap(entry=>entry.isDirectory()?files(`${dir}/${entry.name}`):[`${dir}/${entry.name}`]);}
const sourceFiles=files("src");

test("los acentos coinciden con el registro de paleta aprobada (FUDIA DS 1.0)",()=>{
  const reference=JSON.parse(read("docs/visual-references/hero-dashboard-palette.json"));
  for(const sample of reference.samples){
    assert.equal(resolveTokens(`var(--${sample.token})`).toUpperCase(),sample.hex,sample.region);
    assert.ok(sample.alpha>=250);
  }
  assert.equal(resolveTokens("var(--amount-paid)"),resolveTokens("var(--brand-600)"),"Los importes pagados usan el verde exacto de la especificación");
});

test("la paleta operativa conserva contraste AA y una rampa índigo luminosa ordenada",()=>{
  const luminance=value=>value.slice(1).match(/../g).map(v=>parseInt(v,16)/255)
    .map(v=>v<=.04045?v/12.92:((v+.055)/1.055)**2.4)
    .reduce((sum,v,i)=>sum+v*[.2126,.7152,.0722][i],0);
  const color=name=>resolveTokens(`var(--${name})`);
  const ramp=[25,50,100,200,300,400,500,600,700,800].map(step=>luminance(color(`primary-${step}`)));
  ramp.slice(1).forEach((value,index)=>assert.ok(value<ramp[index],"Cada paso debe ser más oscuro"));
  for(const [fg,bg] of [["surface","primary-600"],["surface","primary-700"],["primary-700","primary-100"],["primary-700","primary-200"],
    ["surface","danger-600"],["surface","danger-700"],["danger-600","danger-50"],["danger-600","danger-100"],
    ["brand-700","brand-100"],["surface","brand-700"],["ink-950","brand-600"],["ink-800","brand-600"],
    ["warning-600","warning-50"],["warning-800","warning-100"],["surface","danger-500"],["ink-950","warning-500"],
    ["info-600","info-50"],["teal-700","teal-100"],["digital-700","digital-100"],["surface","ink-700"],
    ["brand-800","brand-50"],["sky-800","sky-100"],["digital-700","digital-50"],["peach-700","peach-100"]]){
    const light=[luminance(color(fg)),luminance(color(bg))].sort((a,b)=>b-a);
    assert.ok((light[0]+.05)/(light[1]+.05)>=4.5,`${fg} sobre ${bg}`);
  }
});

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
  const external=new Set(["--font-inter","--font-geist-mono"]);
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
  changed.set("--primary-600","#123456");changed.set("--brand-700","#234567");changed.set("--danger-600","#345678");changed.set("--space-3","18px");
  assert.equal(resolveTokens("var(--action-pay)",changed),"#123456");
  assert.equal(resolveTokens("var(--primary-600)",changed),"#123456");
  assert.equal(resolveTokens("var(--brand-700)",changed),"#234567");
  assert.equal(resolveTokens("var(--danger)",changed),"#345678");
  assert.match(resolveTokens("var(--primary-600-alpha-16)",changed),/#123456/);
  const dialog=postcss.parse(read("src/design-system/styles/confirm-dialog.css"));
  let headerGap;dialog.walkDecls("gap",d=>{if(d.parent.selector===".confirm-header")headerGap=d.value;});
  assert.equal(resolveTokens(headerGap,changed),"18px");
});

test("los controles conservan la altura táctil y las confirmaciones entran sin rebotes",()=>{
  assert.equal(resolveTokens("var(--control-height)"),"40px");
  const mobileRoot=tokenAst.nodes.find(n=>n.type==="atrule"&&n.params==="(width<=600px)"&&n.nodes.some(r=>r.selectors?.includes(":root")));
  assert.ok(mobileRoot);
  const control=mobileRoot.nodes.find(n=>n.selectors?.includes(":root")).nodes.find(n=>n.prop==="--control-height");
  assert.equal(resolveTokens(control.value),"48px");
  assert.equal(resolveTokens("var(--motion-dialog-duration)"),"180ms");
  assert.equal(resolveTokens("var(--motion-dialog-easing)"),"cubic-bezier(.23,1,.32,1)");
  const css=read("src/design-system/styles/confirm-dialog.css");
  assert.match(css,/@media\(prefers-reduced-motion:no-preference\)/);
  assert.match(css,/from\{opacity:0;transform:scale\(\.98\)\}/);
  assert.doesNotMatch(css,/scale\(1\.0\d\)/,"Sin rebotes ni overshoot en confirmaciones (FUDIA DS 1.0)");
});
