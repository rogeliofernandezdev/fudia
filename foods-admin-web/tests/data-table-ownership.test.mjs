import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import test from "node:test";
import postcss from "postcss";

const root=new URL("../",import.meta.url);
const read=path=>readFileSync(new URL(path,root),"utf8");

test("las tablas informativas tienen estilos propios y no dependen de globals",()=>{
  const source=read("src/design-system/styles/data-table.css");
  const global=read("src/styles/globals.css");
  const sheet=postcss.parse(source);
  const selectors=new Set();
  sheet.walkRules(rule=>rule.selectors.forEach(selector=>selectors.add(selector)));
  for(const selector of ["table","th","td",".table-wrap",".table-wrap table",".data-table",".data-table-viewport",".data-table-hint"]){
    assert.ok(selectors.has(selector),selector);
  }
  assert.doesNotMatch(global,/\.table-wrap\{|^table\{|^th\{|^td\{/m);
  assert.match(read("src/app/layout.tsx"),/design-system\/styles\/data-table\.css/);
  assert.match(source,/scrollbar-width:thin/,"el desplazamiento no debe ocultarse en móvil");
});

test("DataTable es una tabla nativa con región desplazable etiquetada y composición Tailwind",()=>{
  const table=read("src/design-system/data-table.tsx");
  assert.match(table,/role="region" aria-label=\{label\} aria-describedby=\{hasOverflow\?hintId:undefined\} tabIndex=\{0\}/);
  assert.match(table,/const viewportClassName=/);
  assert.match(table,/min-w-0","max-w-full/);
  assert.match(table,/<table \{\.\.\.tableProps\} className=/);
  assert.match(table,/useRef<HTMLDivElement>/);
  assert.match(table,/new ResizeObserver\(update\)/);
  assert.match(read("src/design-system/index.ts"),/export \{DataTable\} from "\.\/data-table"/);
  const management=read("src/shared/management/management-page.tsx");
  assert.match(management,/<DataTable label=/);
  assert.doesNotMatch(management,/management-cards/);
});

test("Pagination es propietaria de los estilos de paginación informativa",()=>{
  const css=read("src/design-system/styles/pagination.css");
  const global=read("src/styles/globals.css");
  assert.match(css,/\.standard-pagination\{display:flex/);
  assert.match(css,/@media \(width<=700px\)/);
  assert.doesNotMatch(global,/\.standard-pagination\{display:flex/);
});

test("responsive: mantiene las columnas y muestra desplazamiento solo si hay overflow",()=>{
  const source=read("src/design-system/styles/data-table.css");
  const component=read("src/design-system/data-table.tsx");
  const ast=postcss.parse(source);
  assert.match(source,/\.data-table\{width:max-content;min-width:100%\}/);
  assert.match(source,/\.data-table :is\(th,td\):first-child\{position:sticky/);
  assert.match(source,/\.data-table td:first-child b\{[^}]*text-overflow:ellipsis/);
  assert.match(source,/\.data-table-hint\{display:flex/);
  assert.match(source,/\.data-table-viewport\{[^}]*overflow-x:auto/);
  assert.match(component,/const overflowing=viewport.scrollWidth>viewport.clientWidth\+1/);
  assert.match(component,/\{hasOverflow&&\(/);
  assert.doesNotMatch(component,/management-cards|display:none/);
  let responsive=false;
  ast.walkAtRules("media",node=>{if(node.params==="(width<=700px)")responsive=true;});
  assert.equal(responsive,true);
});
